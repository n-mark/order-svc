package ver1

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"order-svc/internal/clients"
	"order-svc/internal/models"
	"order-svc/internal/service"
	"order-svc/internal/store"
)

type OrderHandler struct {
	order service.OrderService
}

func NewOrderHandler(order service.OrderService) *OrderHandler {
	return &OrderHandler{order: order}
}

// --- broker-driven handlers (signature is dictated by messaging.HandlerFunc) ---

// OnOrderPayment is invoked for every message of the `order-payment` topic.
func (h *OrderHandler) OnOrderPayment(body []byte) (bool, error) {
	payload := models.OrderPaymentEvent{}
	if err := json.Unmarshal(body, &payload); err != nil {
		slog.Error("failed to unmarshal order-payment event", "err", err)
		return false, err
	}
	return h.order.OnOrderPayment(context.Background(), payload)
}

// OnDeliveryStatus is invoked for every message of the `delivery` topic.
func (h *OrderHandler) OnDeliveryStatus(body []byte) (bool, error) {
	payload := models.DeliveryStatusEvent{}
	if err := json.Unmarshal(body, &payload); err != nil {
		slog.Error("failed to unmarshal delivery event", "err", err)
		return false, err
	}
	return h.order.OnDeliveryStatus(context.Background(), payload)
}

// --- HTTP handlers (Gin) ---

func parseUserID(c *gin.Context) (int64, bool) {
	s := c.GetHeader("x-user-id")
	if s == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing x-user-id header"})
		return 0, false
	}
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "x-user-id must be an integer"})
		return 0, false
	}
	return id, true
}

// CreateOrderHandler handles POST /api/v1/order.
func (h *OrderHandler) CreateOrderHandler(c *gin.Context) {
	userId, ok := parseUserID(c)
	if !ok {
		return
	}

	req := createOrderRequest{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ord, err := h.order.CreateOrder(c.Request.Context(), userId, req.AdvertId)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, ord)
}

// SetDeliveryHandler handles PATCH /api/v1/order/:id/delivery.
func (h *OrderHandler) SetDeliveryHandler(c *gin.Context) {
	orderId, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
		return
	}

	req := setDeliveryRequest{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ord, err := h.order.SetDelivery(c.Request.Context(), orderId, req.ProviderId, req.To)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, ord)
}

// CancelOrderHandler handles POST /api/v1/order/:id/cancel.
func (h *OrderHandler) CancelOrderHandler(c *gin.Context) {
	orderId, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
		return
	}

	ord, err := h.order.CancelOrder(c.Request.Context(), orderId)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, ord)
}

// PayOrderHandler handles POST /api/v1/order/:id/pay.
// The order amount is taken from the order record, not from the request body,
// so clients cannot tamper with the payment sum.
func (h *OrderHandler) PayOrderHandler(c *gin.Context) {
	userId, ok := parseUserID(c)
	if !ok {
		return
	}

	orderId, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
		return
	}

	transactionId, err := h.order.PayOrder(c.Request.Context(), orderId, userId)
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"transaction_id": transactionId})
}

// GetOrderHandler handles GET /api/v1/order/:id.
func (h *OrderHandler) GetOrderHandler(c *gin.Context) {
	orderId, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
		return
	}

	ord, err := h.order.GetOrder(c.Request.Context(), orderId)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, ord)
}

// ListOrdersHandler handles GET /api/v1/order (all orders for x-user-id).
func (h *OrderHandler) ListOrdersHandler(c *gin.Context) {
	userId, ok := parseUserID(c)
	if !ok {
		return
	}

	orders, err := h.order.ListOrders(c.Request.Context(), userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, orders)
}

func (h *OrderHandler) writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
	case errors.Is(err, clients.ErrAdvertNotFound):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "advert not found"})
	case errors.Is(err, service.ErrDeliveryUnavailable):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrDuplicateOrder):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrEmptyOrder):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, clients.ErrProviderNotFound):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrCannotCancel):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrOrderNotPayable):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, clients.ErrBillingFailed):
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
