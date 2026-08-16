package models

import (
	"time"

	"github.com/google/uuid"
)

// Order statuses (v2: payment is initiated over HTTP against billing-svc; the
// order only reacts to the `order-payment` and `delivery` topics).
const (
	OrderStatusCreated           = "CREATED"             // just created, delivery not yet chosen / not paid
	OrderStatusPaymentRequired   = "PAYMENT_REQUIRED"    // delivery chosen, ready to pay
	OrderStatusProcessingPayment = "PROCESSING_PAYMENT"  // payment attempt in progress
	OrderStatusPaid              = "PAID"                // billing reported a successful payment
	OrderStatusAwaitDelivery     = "AWAIT_DELIVERY"      // paid, delivery published, waiting for delivery-svc
	OrderStatusDelivering        = "DELIVERING"          // delivery-svc reported the parcel is sent
	OrderStatusDelivered         = "DELIVERED"           // delivery-svc reported delivered
	OrderStatusCancelled         = "CANCELLED"           // cancelled by user
	OrderStatusFailed            = "FAILED"              // terminal: payment failed and no retries allowed (legacy/optional)
)

// Address is a delivery endpoint (from / to). Mirrors delivery-svc.
type Address struct {
	City    string `json:"city"`
	Address string `json:"address"`
	Zip     string `json:"zip,omitempty"`
}

// Item is a single ordered position; weight is in grams. Mirrors delivery-svc.
// AdvertId is the advert UUID (as stored in advert-cmd-svc).
type Item struct {
	AdvertId string `json:"advert_id"`
	Qty      int    `json:"qty"`
	Weight   int64  `json:"weight"`
}

// DeliveryDetails is the delivery part of an order, chosen by the user before
// payment and carried in the outgoing `order` event. Mirrors delivery-svc.
type DeliveryDetails struct {
	ProviderId int64   `json:"provider_id"`
	From       Address `json:"from"`
	To         Address `json:"to"`
	Price      float64 `json:"price"`
}

// Order is the persisted order.
type Order struct {
	ID         uuid.UUID        `json:"id"`
	UserId     int64            `json:"user_id"`
	SellerId   int64            `json:"seller_id"`
	ReceiverId int64            `json:"receiver_id"`
	Price      float64          `json:"price"`
	Status     string           `json:"status"`
	Items      []Item           `json:"items"`
	Delivery   *DeliveryDetails `json:"delivery,omitempty"`
	CreatedAt  time.Time        `json:"created_at"`
	UpdatedAt  time.Time        `json:"updated_at"`
}
