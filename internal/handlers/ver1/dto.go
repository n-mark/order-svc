package ver1

// createOrderRequest is the body for POST /api/v1/order.
type createOrderRequest struct {
	Price float64 `json:"price" binding:"required,gt=0"`
}
