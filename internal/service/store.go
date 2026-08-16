package service

import (
	"context"

	"github.com/google/uuid"

	"order-svc/internal/models"
)

// Store describes the persistence operations required by OrderService.
type Store interface {
	CreateOrder(ctx context.Context, order models.Order) (models.Order, error)
	UpdateOrderStatus(ctx context.Context, orderId uuid.UUID, status string) (models.Order, error)
	SetDeliveryDetails(ctx context.Context, orderId uuid.UUID, d models.DeliveryDetails, price float64) (models.Order, error)
	GetOrder(ctx context.Context, orderId uuid.UUID) (models.Order, error)
	ListOrdersByUser(ctx context.Context, userId int64) ([]models.Order, error)
	HasActiveOrderForAdvert(ctx context.Context, advertId string) (bool, error)
	EventAlreadyProcessed(ctx context.Context, eventID uuid.UUID) (bool, error)
	MarkEventProcessed(ctx context.Context, eventID uuid.UUID, eventType string) error
}
