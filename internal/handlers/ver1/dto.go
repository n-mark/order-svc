package ver1

import "order-svc/internal/models"

// createOrderRequest is the body for POST /api/v1/order.
type createOrderRequest struct {
	AdvertId string `json:"advert_id" binding:"required,uuid"`
}

// setDeliveryRequest is the body for PATCH /api/v1/order/:id/delivery.
type setDeliveryRequest struct {
	ProviderId int64          `json:"provider_id" binding:"required"`
	To         models.Address `json:"to"`
}
