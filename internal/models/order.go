package models

import "github.com/google/uuid"

type Order struct {
	EventId uuid.UUID `json:"event_id"`
	UserId int64 `json:"user_id"`
	OrderId uuid.UUID `json:"order_id"`
	ToWithdraw float64 `json:"to_withdraw"`
}