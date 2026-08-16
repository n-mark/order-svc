package clients

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// ErrAdvertNotFound is returned when advert-cmd-svc replies 404.
var ErrAdvertNotFound = errors.New("advert not found")

// Address is the address part of an advert response.
type Address struct {
	City        string `json:"city"`
	Street      string `json:"street"`
	Housenumber string `json:"housenumber"`
	Postcode    string `json:"postcode"`
}

// Advert is the subset of advert-cmd-svc's GET /adverts/{id} response we need.
type Advert struct {
	ID                string  `json:"id"`
	Price             int64   `json:"price"`
	CreatedBy         int64   `json:"created_by"`
	ShippingAvailable bool    `json:"shipping_available"`
	Address           Address `json:"address"`
}

// AdvertClient talks to advert-cmd-svc over HTTP.
type AdvertClient struct {
	baseURL string
	http    *http.Client
}

func NewAdvertClient(baseURL string) *AdvertClient {
	return &AdvertClient{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

// GetAdvert fetches a single advert by its UUID.
func (c *AdvertClient) GetAdvert(ctx context.Context, id string) (Advert, error) {
	url := fmt.Sprintf("%s/adverts/%s", c.baseURL, id)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Advert{}, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return Advert{}, fmt.Errorf("call advert-cmd-svc: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return Advert{}, ErrAdvertNotFound
	default:
		return Advert{}, fmt.Errorf("advert-cmd-svc returned %d", resp.StatusCode)
	}

	var a Advert
	if err := json.NewDecoder(resp.Body).Decode(&a); err != nil {
		return Advert{}, fmt.Errorf("decode advert: %w", err)
	}
	return a, nil
}
