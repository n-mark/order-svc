package models

import (
	"time"

	"github.com/google/uuid"
)

// Billing is the persisted billing account, plus optional fields used when
// it is serialized as an event (EventType/EventId/Status/OrderId).
type Billing struct {
	ID        uuid.UUID `json:"id,omitempty"`
	UserId    int64     `json:"user_id"`
	Balance   float64   `json:"balance"`
	EventType string    `json:"event_type,omitempty"`
	OrderId   uuid.UUID `json:"order_id,omitempty"`
	EventId   uuid.UUID `json:"event_id,omitempty"`
	Status    string    `json:"status,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}
