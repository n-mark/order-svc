package service

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"

	"billing-svc/internal/models"
)

type OrderService struct {
	store          Store
	responseWriter ResponseWriter
}

func NewOrderService(s Store, rw ResponseWriter) *OrderService {
	return &OrderService{store: s, responseWriter: rw}
}

// CreateOrder persists a new order in `pending` status and publishes an
// `order.created` event so billing-svc can attempt to withdraw the money.
func (o *OrderService) CreateOrder(ctx context.Context, userId int64, price float64) (models.Order, error) {
	if price <= 0 {
		return models.Order{}, errors.New("price must be positive")
	}

	order := models.Order{
		ID:     uuid.New(),
		UserId: userId,
		Price:  price,
		Status: models.OrderStatusPending,
	}

	saved, err := o.store.CreateOrder(ctx, order)
	if err != nil {
		return models.Order{}, err
	}

	evt := models.OrderCreatedEvent{
		EventId:   uuid.New(),
		EventType: "ORDER_CREATED",
		OrderId:   saved.ID,
		UserId:    saved.UserId,
		Price:     saved.Price,
	}
	if err := o.responseWriter.ReportOrderCreated(evt); err != nil {
		slog.Error("failed to publish order.created", "err", err, "order_id", saved.ID)
		return models.Order{}, err
	}

	slog.Info("order created", "order_id", saved.ID, "user_id", saved.UserId, "price", saved.Price)
	return saved, nil
}

// GetOrder returns a single order by id.
func (o *OrderService) GetOrder(ctx context.Context, orderId uuid.UUID) (models.Order, error) {
	return o.store.GetOrder(ctx, orderId)
}

// ListOrders returns all orders for the given user.
func (o *OrderService) ListOrders(ctx context.Context, userId int64) ([]models.Order, error) {
	return o.store.ListOrdersByUser(ctx, userId)
}

// UpdateOrderStatusOnBillingResponse consumes billing-svc's payment result,
// flips the order status to `paid`/`failed`, and publishes an
// `order.updated` event for the notification-svc.
func (o *OrderService) UpdateOrderStatusOnBillingResponse(ctx context.Context, resp models.BillingResponse) (bool, error) {
	slog.Info("handling billing response", "order_id", resp.OrderId, "status", resp.Status)

	// Idempotency check.
	if dup, err := o.store.EventAlreadyProcessed(ctx, resp.EventId); err != nil {
		return false, err
	} else if dup {
		slog.Info("billing response already processed, skipping", "event_id", resp.EventId)
		return true, nil
	}

	newStatus := models.OrderStatusFailed
	if resp.Status == models.BillingStatusPaid {
		newStatus = models.OrderStatusPaid
	}

	_, err := o.store.UpdateOrderStatus(ctx, resp.OrderId, newStatus)
	if err != nil {
		return false, err
	}

	// evt := models.OrderUpdatedEvent{
	// 	EventId:   uuid.New(),
	// 	EventType: "ORDER_UPDATED",
	// 	OrderId:   updated.ID,
	// 	UserId:    updated.UserId,
	// 	Price:     updated.Price,
	// 	Status:    updated.Status,
	// }
	// if err := o.responseWriter.ReportOrderUpdated(evt); err != nil {
	// 	return false, err
	// }

	if err := o.store.MarkEventProcessed(ctx, resp.EventId, "BILLING_RESPONSE"); err != nil {
		return false, err
	}
	return true, nil
}
