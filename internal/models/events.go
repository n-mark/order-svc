package models

import "github.com/google/uuid"

// Incoming event types from the `order-payment` topic (published by billing-svc).
const (
	EventTypePaymentSuccess = "PAYMENT_SUCCESS"
	EventTypePaymentFailed  = "PAYMENT_FAILED"
)

// Incoming event types from the `delivery` topic (published by delivery-svc).
const (
	EventTypeDeliverySent      = "DELIVERY_SENT"
	EventTypeDeliveryDelivered = "DELIVERY_DELIVERED"
	EventTypeDeliveryCancelled = "DELIVERY_CANCELLED"
)

// Outgoing event types published to the `order` topic (consumed by delivery-svc).
const (
	EventTypeOrderPaid      = "ORDER_PAID"
	EventTypeOrderCancelled = "ORDER_CANCELLED"
)

// OrderPaymentEvent is consumed from the `order-payment` topic. Mirrors
// billing-svc's contract.
type OrderPaymentEvent struct {
	EventId       uuid.UUID `json:"event_id"`
	EventType     string    `json:"event_type"`
	Version       string    `json:"version"`
	OrderId       uuid.UUID `json:"order_id"`
	UserId        int64     `json:"user_id"`
	TransactionId uuid.UUID `json:"transaction_id"`
	Status        string    `json:"status"`
	Message       string    `json:"message"`
}

// DeliveryStatusEvent is consumed from the `delivery` topic. Mirrors
// delivery-svc's DeliveryEvent contract.
type DeliveryStatusEvent struct {
	EventId    uuid.UUID `json:"event_id"`
	EventType  string    `json:"event_type"`
	Version    string    `json:"version"`
	OrderId    uuid.UUID `json:"order_id"`
	DeliveryId uuid.UUID `json:"delivery_id"`
	Status     string    `json:"status"`
}

// OrderEvent is published to the `order` topic. `Delivery` is only present for
// ORDER_PAID events. Schema mirrors delivery-svc's OrderEvent.
type OrderEvent struct {
	EventId    uuid.UUID        `json:"event_id"`
	EventType  string           `json:"event_type"`
	Version    string           `json:"version"`
	OrderId    uuid.UUID        `json:"order_id"`
	SellerId   int64            `json:"seller_id"`
	ReceiverId int64            `json:"receiver_id"`
	Items      []Item           `json:"items"`
	Delivery   *DeliveryDetails `json:"delivery,omitempty"`
}
