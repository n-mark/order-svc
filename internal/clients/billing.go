package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// ErrBillingFailed is returned when billing-svc replies with a non-2xx status
// that is not a client error we can map more specifically.
var ErrBillingFailed = errors.New("billing request failed")

// CreateTransactionResponse is the body returned by billing-svc on 202 Accepted.
type CreateTransactionResponse struct {
	TransactionID string `json:"transaction_id"`
	OrderID       string `json:"order_id"`
	Status        string `json:"status"`
}

// BillingClient talks to billing-svc over HTTP.
type BillingClient struct {
	baseURL string
	http    *http.Client
}

func NewBillingClient(baseURL string) *BillingClient {
	return &BillingClient{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

type createTransactionRequest struct {
	OrderID string  `json:"order_id"`
	Amount  float64 `json:"amount"`
}

// CreateTransaction asks billing-svc to start a payment transaction for the
// given order. The amount is taken from order-svc's own record, never from the
// client, so billing-svc can trust it as the source of truth.
func (c *BillingClient) CreateTransaction(ctx context.Context, userID int64, orderID string, amount float64) (CreateTransactionResponse, error) {
	url := fmt.Sprintf("%s/api/v1/transaction", c.baseURL)

	body, err := json.Marshal(createTransactionRequest{
		OrderID: orderID,
		Amount:  amount,
	})
	if err != nil {
		return CreateTransactionResponse{}, fmt.Errorf("marshal transaction request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return CreateTransactionResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-user-id", strconv.FormatInt(userID, 10))

	resp, err := c.http.Do(req)
	if err != nil {
		return CreateTransactionResponse{}, fmt.Errorf("call billing-svc: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return CreateTransactionResponse{}, fmt.Errorf("%w: billing-svc returned %d", ErrBillingFailed, resp.StatusCode)
	}

	var tx CreateTransactionResponse
	if err := json.NewDecoder(resp.Body).Decode(&tx); err != nil {
		return CreateTransactionResponse{}, fmt.Errorf("decode transaction response: %w", err)
	}
	return tx, nil
}
