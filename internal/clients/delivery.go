package clients

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// ErrProviderNotFound is returned when the requested provider is not active or unknown.
var ErrProviderNotFound = errors.New("provider not found")

// ProviderQuote mirrors delivery-service's provider quote response.
type ProviderQuote struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	BasePrice     float64 `json:"base_price"`
	PricePerKg    float64 `json:"price_per_kg"`
	DaysEstimated int     `json:"days_estimated"`
	Active        bool    `json:"active"`
	Price         float64 `json:"price"`
}

// DeliveryClient talks to delivery-service over HTTP.
type DeliveryClient struct {
	baseURL string
	http    *http.Client
}

func NewDeliveryClient(baseURL string) *DeliveryClient {
	return &DeliveryClient{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

// ProviderQuote returns the calculated price for a specific provider and weight.
func (c *DeliveryClient) ProviderQuote(ctx context.Context, providerID int64, weightGrams int64) (float64, error) {
	u, err := url.Parse(fmt.Sprintf("%s/internal/v1/providers", c.baseURL))
	if err != nil {
		return 0, err
	}
	q := u.Query()
	q.Set("weight", strconv.FormatInt(weightGrams, 10))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return 0, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, fmt.Errorf("call delivery-service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("delivery-service returned %d", resp.StatusCode)
	}

	var quotes []ProviderQuote
	if err := json.NewDecoder(resp.Body).Decode(&quotes); err != nil {
		return 0, fmt.Errorf("decode provider quotes: %w", err)
	}

	for _, q := range quotes {
		if q.ID == providerID {
			return q.Price, nil
		}
	}
	return 0, ErrProviderNotFound
}
