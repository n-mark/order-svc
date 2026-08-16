package service

import (
	"context"

	"order-svc/internal/clients"
	"order-svc/internal/models"
)

// ResponseWriter publishes order events to the `order` topic so delivery-svc
// can react (create / cancel a delivery).
type ResponseWriter interface {
	PublishOrderEvent(e models.OrderEvent) error
}

// BillingClient abstracts the HTTP client used to initiate payment transactions
// with billing-svc. Implemented by clients.BillingClient.
type BillingClient interface {
	CreateTransaction(ctx context.Context, userID int64, orderID string, amount float64) (clients.CreateTransactionResponse, error)
}
