package service

import (
	"context"

	"github.com/google/uuid"

	"billing-svc/internal/models"
)

type Store interface {
	CreateOrder(ctx context.Context, o models.Order) (models.Order, error)
	UpdateOrderStatus(ctx context.Context, orderId uuid.UUID, status string) (models.Order, error)
	GetOrder(ctx context.Context, orderId uuid.UUID) (models.Order, error)
	ListOrdersByUser(ctx context.Context, userId int64) ([]models.Order, error)

	EventAlreadyProcessed(ctx context.Context, eventID uuid.UUID) (bool, error)
	MarkEventProcessed(ctx context.Context, eventID uuid.UUID, eventType string) error
}
