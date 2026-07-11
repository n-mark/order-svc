package ver1

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"billing-svc/internal/models"
	"billing-svc/internal/service"
	"billing-svc/internal/store"

	"encoding/json"
)

type BillingHandler struct {
	order service.OrderService
}

func NewBillingHandler(order service.OrderService) *BillingHandler {
	return &BillingHandler{
		order: order}
}

// --- AMQP-driven handlers (signature is dictated by messaging.HandlerFunc) ---

func (h *BillingHandler) UpdateOrderAfterBillingResponse(body []byte) (bool, error) {
	payload := models.BillingResponse{}
	if err := json.Unmarshal(body, &payload); err != nil {
		slog.Error("failed to unmarshal", "error", err)
		return false, err
	}

	h.order.UpdateOrderStatusOnBillingResponse()

	return true, nil
	//h.billing.CreateBillingAccount(context.Background(), payload)
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

func (h *BillingHandler) AddMoneyHandler(c *gin.Context) {
	userId, ok := parseUserID(c)
	if !ok {
		return
	}

	req := moneyRequest{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	slog.Info("ADD MONEY HANDLER INVOKED", "user_id", userId, "amount", req.Amount)

	b, err := h.billing.AddMoney(c.Request.Context(), userId, req.Amount)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "billing account not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, b)
}

func (h *BillingHandler) WithdrawMoneyHandler(c *gin.Context) {
	userId, ok := parseUserID(c)
	if !ok {
		return
	}

	req := moneyRequest{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	slog.Info("WITHDRAW MONEY HANDLER INVOKED", "user_id", userId, "amount", req.Amount)

	b, err := h.billing.Withdraw(c.Request.Context(), userId, req.Amount)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "billing account not found"})
		case errors.Is(err, store.ErrInsufficientFunds):
			c.JSON(http.StatusPaymentRequired, gin.H{"error": "insufficient funds"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, b)
}

func (h *BillingHandler) GetBillingHandler(c *gin.Context) {
	userId, ok := parseUserID(c)
	if !ok {
		return
	}

	b, err := h.billing.GetByUserId(c.Request.Context(), userId)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "billing account not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, b)
}
