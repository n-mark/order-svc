package service

import (
	"context"

	"order-svc/internal/clients"
)

// AdvertClient fetches advert data from advert-cmd-svc. Used at order creation
// to verify the advert exists and that delivery (shipping) is enabled for it.
type AdvertClient interface {
	GetAdvert(ctx context.Context, id string) (clients.Advert, error)
}

// DeliveryClient fetches delivery provider quotes from delivery-service.
type DeliveryClient interface {
	ProviderQuote(ctx context.Context, providerID int64, weightGrams int64) (float64, error)
}
