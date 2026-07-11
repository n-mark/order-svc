package models

import "github.com/google/uuid"

type BillingResponse struct {
	EventId   uuid.UUID `json:"event_id"`
	EventType string    `json:"event_type,omitempty"`
	UserId    int64     `json:"user_id"`
	OrderId   uuid.UUID `json:"order_id,omitempty"`
	Status    string    `json:"status,omitempty"`
	Balance   float64   `json:"balance"`
}
