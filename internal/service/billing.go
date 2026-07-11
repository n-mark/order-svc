package service

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"

	"billing-svc/internal/models"
	"billing-svc/internal/store"
)

type BillingService struct {
	store          Store
	responseWriter ResponseWriter
}

func NewBillingService(s Store, rw ResponseWriter) *BillingService {
	return &BillingService{store: s, responseWriter: rw}
}

func (o *BillingService) CreateBillingAccount(ctx context.Context, p models.Order) (bool, error) {
	slog.Info("CREATING BILLING ACCOUNT...", "userid", p.UserId)

	b, err := o.store.Save(ctx, p)
	if err != nil {
		slog.Error("failed to save billing account", "err", err)
		return false, err
	}

	response := models.Billing{
		EventId:   uuid.New(),
		EventType: "BILLING_ACCOUNT_CREATED",
		UserId:    b.UserId,
		Status:    "OK",
	}

	if err := o.responseWriter.ReportOrderCreated(response); err != nil {
		slog.Error("failed to publish profile update", "err", err)
		return false, err
	}
	return true, nil
}

// AddMoney increases the user's balance and returns the updated account.
func (o *BillingService) AddMoney(ctx context.Context, userId int64, amount float64) (models.Billing, error) {
	if amount <= 0 {
		return models.Billing{}, errors.New("amount must be positive")
	}
	return o.store.Increase(ctx, userId, amount)
}

// Withdraw decreases the user's balance if funds are sufficient.
func (o *BillingService) Withdraw(ctx context.Context, userId int64, amount float64) (models.Billing, error) {
	if amount <= 0 {
		return models.Billing{}, errors.New("amount must be positive")
	}
	return o.store.Withdraw(ctx, userId, amount)
}

// GetByUserId proxies the store; useful for a "GET /billing" handler.
func (o *BillingService) GetByUserId(ctx context.Context, userId int64) (models.Billing, error) {
	return o.store.GetByUserId(ctx, userId)
}

// Compile-time guard: store errors must be addressable from this package.
var _ = store.ErrNotFound
