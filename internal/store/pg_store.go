package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"order-svc/internal/models"
)

var (
	ErrNotFound = errors.New("order not found")
)

type PgStore struct {
	db *pgxpool.Pool
}

func NewPgStore(db *pgxpool.Pool) *PgStore {
	return &PgStore{db: db}
}

const orderColumns = `id, user_id, seller_id, receiver_id, price, status, items, delivery, created_at, updated_at`

func scanOrder(row pgx.Row) (models.Order, error) {
	var (
		o        models.Order
		items    []byte
		delivery []byte
	)
	if err := row.Scan(&o.ID, &o.UserId, &o.SellerId, &o.ReceiverId, &o.Price,
		&o.Status, &items, &delivery, &o.CreatedAt, &o.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Order{}, ErrNotFound
		}
		return models.Order{}, err
	}

	if len(items) > 0 {
		if err := json.Unmarshal(items, &o.Items); err != nil {
			return models.Order{}, err
		}
	}
	if len(delivery) > 0 {
		var d models.DeliveryDetails
		if err := json.Unmarshal(delivery, &d); err != nil {
			return models.Order{}, err
		}
		o.Delivery = &d
	}
	return o, nil
}

// CreateOrder inserts a new order and returns the persisted row.
func (p *PgStore) CreateOrder(ctx context.Context, o models.Order) (models.Order, error) {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	if o.Status == "" {
		o.Status = models.OrderStatusCreated
	}

	items, err := json.Marshal(o.Items)
	if err != nil {
		return models.Order{}, err
	}
	var delivery []byte
	if o.Delivery != nil {
		if delivery, err = json.Marshal(o.Delivery); err != nil {
			return models.Order{}, err
		}
	}

	row := p.db.QueryRow(ctx, `
		INSERT INTO orders (id, user_id, seller_id, receiver_id, price, status, items, delivery)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING `+orderColumns,
		o.ID, o.UserId, o.SellerId, o.ReceiverId, o.Price, o.Status, items, delivery)

	return scanOrder(row)
}

// UpdateOrderStatus flips an order status and bumps updated_at.
func (p *PgStore) UpdateOrderStatus(ctx context.Context, orderId uuid.UUID, status string) (models.Order, error) {
	return scanOrder(p.db.QueryRow(ctx, `
		UPDATE orders SET status = $2, updated_at = now()
		WHERE id = $1
		RETURNING `+orderColumns, orderId, status))
}

// SetDeliveryDetails stores the delivery option chosen by the user, updates
// the final order price (advert price + delivery price) and moves the order to
// PAYMENT_REQUIRED so it can be paid.
func (p *PgStore) SetDeliveryDetails(ctx context.Context, orderId uuid.UUID, d models.DeliveryDetails, price float64) (models.Order, error) {
	delivery, err := json.Marshal(d)
	if err != nil {
		return models.Order{}, err
	}
	return scanOrder(p.db.QueryRow(ctx, `
		UPDATE orders SET delivery = $2, price = $3, status = $4, updated_at = now()
		WHERE id = $1
		RETURNING `+orderColumns, orderId, delivery, price, models.OrderStatusPaymentRequired))
}

// HasActiveOrderForAdvert reports whether there is an active (non-terminal)
// order that already references the given advert id.
func (p *PgStore) HasActiveOrderForAdvert(ctx context.Context, advertId string) (bool, error) {
	var n int
	err := p.db.QueryRow(ctx, `
		SELECT 1 FROM orders
		WHERE items::jsonb @> $1::jsonb
		  AND status NOT IN ('CANCELLED', 'FAILED')
		LIMIT 1`, fmt.Sprintf(`[{"advert_id":"%s"}]`, advertId)).Scan(&n)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// GetOrder loads a single order by id.
func (p *PgStore) GetOrder(ctx context.Context, orderId uuid.UUID) (models.Order, error) {
	return scanOrder(p.db.QueryRow(ctx,
		`SELECT `+orderColumns+` FROM orders WHERE id = $1`, orderId))
}

// ListOrdersByUser returns all orders belonging to the given user, newest first.
func (p *PgStore) ListOrdersByUser(ctx context.Context, userId int64) ([]models.Order, error) {
	rows, err := p.db.Query(ctx,
		`SELECT `+orderColumns+` FROM orders WHERE user_id = $1 ORDER BY created_at DESC`, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.Order, 0)
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func (p *PgStore) EventAlreadyProcessed(ctx context.Context, eventID uuid.UUID) (bool, error) {
	var n int
	err := p.db.QueryRow(ctx, `SELECT 1 FROM processed_events WHERE event_id = $1`, eventID).Scan(&n)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (p *PgStore) MarkEventProcessed(ctx context.Context, eventID uuid.UUID, eventType string) error {
	_, err := p.db.Exec(ctx, `
		INSERT INTO processed_events (event_id, event_type)
		VALUES ($1, $2)
		ON CONFLICT (event_id) DO NOTHING
	`, eventID, eventType)
	return err
}
