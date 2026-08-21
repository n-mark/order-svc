package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"order-svc/internal/models"
)

var (
	// ErrCannotCancel is returned when an order can no longer be cancelled
	// (delivery already in progress or terminal state).
	ErrCannotCancel = errors.New("order cannot be cancelled in its current status")

	// ErrDeliveryUnavailable is returned when an advert in the order does not
	// support delivery (shipping_available == false in advert-cmd-svc).
	ErrDeliveryUnavailable = errors.New("delivery is not available for this advert")

	// ErrDuplicateOrder is returned when an active order for the same advert already exists.
	ErrDuplicateOrder = errors.New("active order for this advert already exists")

	// ErrEmptyOrder is returned when an order has no items.
	ErrEmptyOrder = errors.New("order must contain at least one item")

	// ErrForbidden is returned when the caller is not the owner of the order.
	ErrForbidden = errors.New("forbidden")

	// ErrOrderNotPayable is returned when the order cannot be paid in its
	// current state (e.g. already paid, cancelled, or delivery not selected).
	ErrOrderNotPayable = errors.New("order cannot be paid in its current state")
)

type OrderService struct {
	store          Store
	responseWriter ResponseWriter
	adverts        AdvertClient
	delivery       DeliveryClient
	billing        BillingClient
}

func NewOrderService(s Store, rw ResponseWriter, adverts AdvertClient, delivery DeliveryClient, billing BillingClient) *OrderService {
	return &OrderService{store: s, responseWriter: rw, adverts: adverts, delivery: delivery, billing: billing}
}

// CreateOrder persists a new order in CREATED status. The request only carries
// the advert id; buyer id comes from X-User-Id, seller id and item price come
// from advert-cmd-svc, and the origin address is taken from the advert's address.
func (o *OrderService) CreateOrder(ctx context.Context, userId int64, advertId string) (models.Order, error) {
	advert, err := o.adverts.GetAdvert(ctx, advertId)
	if err != nil {
		return models.Order{}, fmt.Errorf("check advert %s: %w", advertId, err)
	}
	if !advert.ShippingAvailable {
		return models.Order{}, fmt.Errorf("advert %s: %w", advertId, ErrDeliveryUnavailable)
	}

	exists, err := o.store.HasActiveOrderForAdvert(ctx, advertId)
	if err != nil {
		return models.Order{}, err
	}
	if exists {
		return models.Order{}, ErrDuplicateOrder
	}

	address := advert.Address.Street
	if advert.Address.Housenumber != "" {
		address += ", " + advert.Address.Housenumber
	}

	in := models.Order{
		ID:         uuid.New(),
		UserId:     userId,
		SellerId:   advert.CreatedBy,
		ReceiverId: userId,
		Price:      float64(advert.Price),
		Status:     models.OrderStatusCreated,
		Items: []models.Item{
			{AdvertId: advertId, Qty: 1, Weight: 0},
		},
		Delivery: &models.DeliveryDetails{
			ProviderId: 0,
			From: models.Address{
				City:    advert.Address.City,
				Address: address,
				Zip:     advert.Address.Postcode,
			},
			To:    models.Address{},
			Price: 0,
		},
	}

	saved, err := o.store.CreateOrder(ctx, in)
	if err != nil {
		return models.Order{}, err
	}

	slog.Info("order created", "order_id", saved.ID, "user_id", saved.UserId, "advert_id", advertId, "price", saved.Price)
	return saved, nil
}

// SetDelivery stores the delivery option chosen by the user and calculates the
// final order price as advert price + selected provider delivery price.
func (o *OrderService) SetDelivery(ctx context.Context, orderId uuid.UUID, providerId int64, to models.Address) (models.Order, error) {
	order, err := o.store.GetOrder(ctx, orderId)
	if err != nil {
		return models.Order{}, err
	}

	if len(order.Items) == 0 {
		return models.Order{}, ErrEmptyOrder
	}

	var weight int64
	for _, item := range order.Items {
		weight += item.Weight * int64(item.Qty)
	}

	deliveryPrice, err := o.delivery.ProviderQuote(ctx, providerId, weight)
	if err != nil {
		return models.Order{}, err
	}

	finalPrice := order.Price + deliveryPrice

	d := models.DeliveryDetails{
		ProviderId: providerId,
		From:       order.Delivery.From,
		To:         to,
		Price:      deliveryPrice,
	}

	return o.store.SetDeliveryDetails(ctx, orderId, d, finalPrice)
}

// GetOrder returns a single order by id.
func (o *OrderService) GetOrder(ctx context.Context, orderId uuid.UUID) (models.Order, error) {
	return o.store.GetOrder(ctx, orderId)
}

// PayOrder initiates payment for an order. The amount is taken from the order
// record itself, never from the client, which prevents amount tampering.
// Only the order owner can pay, and only when the order is in PAYMENT_REQUIRED
// status (delivery already selected). The order is moved to PROCESSING_PAYMENT
// while the request is in flight; if billing-svc rejects the request
// synchronously, the status is rolled back to PAYMENT_REQUIRED so the user can
// retry.
func (o *OrderService) PayOrder(ctx context.Context, orderId uuid.UUID, userId int64) (string, error) {
	order, err := o.store.GetOrder(ctx, orderId)
	if err != nil {
		return "", err
	}

	if order.UserId != userId {
		return "", ErrForbidden
	}

	if order.Status != models.OrderStatusPaymentRequired {
		return "", ErrOrderNotPayable
	}

	if order.Delivery == nil || order.Delivery.ProviderId == 0 || order.Price <= 0 {
		return "", ErrOrderNotPayable
	}

	// Move to PROCESSING_PAYMENT to prevent concurrent payment attempts.
	if _, err := o.store.UpdateOrderStatus(ctx, orderId, models.OrderStatusProcessingPayment); err != nil {
		return "", err
	}

	tx, err := o.billing.CreateTransaction(ctx, userId, order.ID.String(), order.Price)
	if err != nil {
		// Rollback so the user can retry; the payment never started.
		if _, rbErr := o.store.UpdateOrderStatus(ctx, orderId, models.OrderStatusPaymentRequired); rbErr != nil {
			slog.Error("failed to rollback order status after billing error", "order_id", orderId, "err", rbErr)
		}
		return "", err
	}

	slog.Info("payment initiated", "order_id", order.ID, "user_id", userId, "amount", order.Price, "transaction_id", tx.TransactionID)
	return tx.TransactionID, nil
}

// ListOrders returns all orders for the given user.
func (o *OrderService) ListOrders(ctx context.Context, userId int64) ([]models.Order, error) {
	return o.store.ListOrdersByUser(ctx, userId)
}

// OnOrderPayment consumes billing-svc's payment result from the `order-payment`
// topic. On success the order becomes AWAIT_DELIVERY and an ORDER_PAID event
// (carrying the delivery details) is published to the `order` topic so
// delivery-svc creates the delivery. On failure the order returns to
// PAYMENT_REQUIRED so the user can retry with a different funding source.
func (o *OrderService) OnOrderPayment(ctx context.Context, e models.OrderPaymentEvent) (bool, error) {
	switch e.EventType {
	case models.EventTypePaymentSuccess, models.EventTypePaymentFailed:
	default:
		slog.Debug("skipping foreign payment event", "event_type", e.EventType)
		return false, nil
	}

	if e.EventId == uuid.Nil {
		return false, fmt.Errorf("missing event id")
	}

	// Idempotency.
	if dup, err := o.store.EventAlreadyProcessed(ctx, e.EventId); err != nil {
		return false, err
	} else if dup {
		slog.Info("payment event already processed, skipping", "event_id", e.EventId)
		return true, nil
	}

	if e.EventType == models.EventTypePaymentFailed {
		// Return to PAYMENT_REQUIRED so the user can retry. The transaction
		// itself is marked failed in billing-svc, but the order stays payable.
		if _, err := o.store.UpdateOrderStatus(ctx, e.OrderId, models.OrderStatusPaymentRequired); err != nil {
			return false, err
		}
		slog.Info("order payment failed, retry allowed", "order_id", e.OrderId)
		return true, o.markProcessed(ctx, e.EventId, e.EventType)
	}

	// Payment succeeded: move to AWAIT_DELIVERY and publish ORDER_PAID.
	order, err := o.store.UpdateOrderStatus(ctx, e.OrderId, models.OrderStatusAwaitDelivery)
	if err != nil {
		return false, err
	}

	if order.Delivery == nil {
		slog.Warn("order paid without delivery details, delivery will not be created", "order_id", order.ID)
	}

	evt := models.OrderEvent{
		EventId:    uuid.New(),
		EventType:  models.EventTypeOrderPaid,
		Version:    "1.0",
		OrderId:    order.ID,
		SellerId:   order.SellerId,
		ReceiverId: order.ReceiverId,
		Items:      order.Items,
		Delivery:   order.Delivery,
	}
	if err := o.responseWriter.PublishOrderEvent(evt); err != nil {
		slog.Error("failed to publish ORDER_PAID", "err", err, "order_id", order.ID)
		return false, err
	}

	slog.Info("order paid", "order_id", order.ID)
	return true, o.markProcessed(ctx, e.EventId, e.EventType)
}

// OnDeliveryStatus consumes delivery-svc's status updates from the `delivery`
// topic and keeps the order status in sync.
func (o *OrderService) OnDeliveryStatus(ctx context.Context, e models.DeliveryStatusEvent) (bool, error) {
	var newStatus string
	switch e.EventType {
	case models.EventTypeDeliverySent:
		newStatus = models.OrderStatusDelivering
	case models.EventTypeDeliveryDelivered:
		newStatus = models.OrderStatusDelivered
	case models.EventTypeDeliveryCancelled:
		newStatus = models.OrderStatusCancelled
	default:
		slog.Debug("skipping foreign delivery event", "event_type", e.EventType)
		return false, nil
	}

	if e.EventId == uuid.Nil {
		return false, fmt.Errorf("missing event id")
	}

	if dup, err := o.store.EventAlreadyProcessed(ctx, e.EventId); err != nil {
		return false, err
	} else if dup {
		slog.Info("delivery event already processed, skipping", "event_id", e.EventId)
		return true, nil
	}

	if _, err := o.store.UpdateOrderStatus(ctx, e.OrderId, newStatus); err != nil {
		return false, err
	}

	slog.Info("order status synced from delivery", "order_id", e.OrderId, "status", newStatus)
	return true, o.markProcessed(ctx, e.EventId, e.EventType)
}

// CancelOrder cancels an order and notifies delivery-svc. Orders that are
// already being delivered (or in a terminal state) cannot be cancelled.
func (o *OrderService) CancelOrder(ctx context.Context, orderId uuid.UUID) (models.Order, error) {
	order, err := o.store.GetOrder(ctx, orderId)
	if err != nil {
		return models.Order{}, err
	}

	switch order.Status {
	case models.OrderStatusDelivering, models.OrderStatusDelivered, models.OrderStatusCancelled:
		return models.Order{}, ErrCannotCancel
	}

	updated, err := o.store.UpdateOrderStatus(ctx, orderId, models.OrderStatusCancelled)
	if err != nil {
		return models.Order{}, err
	}

	evt := models.OrderEvent{
		EventId:   uuid.New(),
		EventType: models.EventTypeOrderCancelled,
		Version:   "1.0",
		OrderId:   updated.ID,
	}
	if err := o.responseWriter.PublishOrderEvent(evt); err != nil {
		slog.Error("failed to publish ORDER_CANCELLED", "err", err, "order_id", updated.ID)
		return models.Order{}, err
	}

	slog.Info("order cancelled", "order_id", updated.ID)
	return updated, nil
}

func (o *OrderService) markProcessed(ctx context.Context, eventId uuid.UUID, eventType string) error {
	return o.store.MarkEventProcessed(ctx, eventId, eventType)
}
