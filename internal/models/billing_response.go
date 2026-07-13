package models

import "github.com/google/uuid"

// Billing payment result statuses (as sent by billing-svc).
const (
	BillingStatusPaid   = "paid"
	BillingStatusFailed = "failed"
)

// BillingResponse is what billing-svc publishes after processing a payment.
type BillingResponse struct {
	EventId   uuid.UUID `json:"event_id"`
	EventType string    `json:"event_type,omitempty"`
	UserId    int64     `json:"user_id"`
	OrderId   uuid.UUID `json:"order_id"`
	Status    string    `json:"status"`
	Balance   float64   `json:"balance,omitempty"`
}
