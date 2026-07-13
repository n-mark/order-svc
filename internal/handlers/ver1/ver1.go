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

	"billing-svc/internal/models"
	"billing-svc/internal/service"
	"billing-svc/internal/store"
)

type OrderHandler struct {
	order service.OrderService
}

func NewOrderHandler(order service.OrderService) *OrderHandler {
	return &OrderHandler{order: order}
}

// --- broker-driven handlers (signature is dictated by messaging.HandlerFunc) ---

// UpdateOrderAfterBillingResponse is invoked when billing-svc reports a
// payment result via broker.
func (h *OrderHandler) UpdateOrderAfterBillingResponse(body []byte) (bool, error) {
	payload := models.BillingResponse{}
	if err := json.Unmarshal(body, &payload); err != nil {
		slog.Error("failed to unmarshal billing response", "error", err)
		return false, err
	}

	return h.order.UpdateOrderStatusOnBillingResponse(context.Background(), payload)
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

	slog.Info("CREATE ORDER HANDLER INVOKED", "user_id", userId, "price", req.Price)

	ord, err := h.order.CreateOrder(c.Request.Context(), userId, req.Price)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, ord)
}

// GetOrderHandler handles GET /api/v1/order/:id.
func (h *OrderHandler) GetOrderHandler(c *gin.Context) {
	idStr := c.Param("id")
	orderId, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
		return
	}

	ord, err := h.order.GetOrder(c.Request.Context(), orderId)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
