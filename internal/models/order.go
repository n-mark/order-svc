package models

import (
	"time"

	"github.com/google/uuid"
)

// Order statuses.
const (
	OrderStatusPending = "pending"
	OrderStatusPaid    = "paid"
	OrderStatusFailed  = "failed"
)

// Order is the persisted order.
type Order struct {
	ID         uuid.UUID `json:"id"`
	UserId     int64     `json:"user_id"`
	ToWithdraw float64   `json:"to_withdraw"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// OrderCreatedEvent is published to billing-svc to request payment.
type OrderCreatedEvent struct {
	EventId    uuid.UUID `json:"event_id"`
	EventType  string    `json:"event_type"`
	OrderId    uuid.UUID `json:"order_id"`
	UserId     int64     `json:"user_id"`
	ToWithdraw float64   `json:"to_withdraw"`
}

// OrderUpdatedEvent is published to notification-svc after billing responded.
type OrderUpdatedEvent struct {
	EventId    uuid.UUID `json:"event_id"`
	EventType  string    `json:"event_type"`
	OrderId    uuid.UUID `json:"order_id"`
	UserId     int64     `json:"user_id"`
	ToWithdraw float64   `json:"to_withdraw"`
	Status     string    `json:"status"`
}
