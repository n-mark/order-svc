package service

import (
	"billing-svc/internal/models"
	"context"

	"github.com/google/uuid"
)

type Store interface {
	Save(ctx context.Context, p models.Order) (models.Billing, error)
	Update(ctx context.Context, p models.Order) (models.Billing, error)
	GetByUserId(ctx context.Context, userId int64) (models.Billing, error)

	EventAlreadyProcessed(ctx context.Context, eventID uuid.UUID) (bool, error)
	MarkEventProcessed(ctx context.Context, eventID uuid.UUID, eventType string) error
}
