package store

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"billing-svc/internal/models"
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

// CreateOrder inserts a new order and returns the persisted row.
func (p *PgStore) CreateOrder(ctx context.Context, o models.Order) (models.Order, error) {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	if o.Status == "" {
		o.Status = models.OrderStatusPending
	}

	row := p.db.QueryRow(ctx, `
		INSERT INTO orders (id, user_id, price, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, price, status, created_at, updated_at
	`, o.ID, o.UserId, o.Price, o.Status)

	out := models.Order{}
	if err := row.Scan(&out.ID, &out.UserId, &out.Price, &out.Status, &out.CreatedAt, &out.UpdatedAt); err != nil {
		return models.Order{}, err
	}
	return out, nil
}

// UpdateOrderStatus flips an order status and bumps updated_at.
func (p *PgStore) UpdateOrderStatus(ctx context.Context, orderId uuid.UUID, status string) (models.Order, error) {
	row := p.db.QueryRow(ctx, `
		UPDATE orders
		SET status = $2, updated_at = now()
		WHERE id = $1
		RETURNING id, user_id, price, status, created_at, updated_at
	`, orderId, status)

	out := models.Order{}
	if err := row.Scan(&out.ID, &out.UserId, &out.Price, &out.Status, &out.CreatedAt, &out.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Order{}, ErrNotFound
		}
		return models.Order{}, err
	}
	return out, nil
}

// GetOrder loads a single order by id.
func (p *PgStore) GetOrder(ctx context.Context, orderId uuid.UUID) (models.Order, error) {
	row := p.db.QueryRow(ctx, `
		SELECT id, user_id, price, status, created_at, updated_at
		FROM orders
		WHERE id = $1
	`, orderId)

	out := models.Order{}
	if err := row.Scan(&out.ID, &out.UserId, &out.Price, &out.Status, &out.CreatedAt, &out.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Order{}, ErrNotFound
		}
		return models.Order{}, err
	}
	return out, nil
}

// ListOrdersByUser returns all orders belonging to the given user, newest first.
func (p *PgStore) ListOrdersByUser(ctx context.Context, userId int64) ([]models.Order, error) {
	rows, err := p.db.Query(ctx, `
		SELECT id, user_id, price, status, created_at, updated_at
		FROM orders
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.Order, 0)
	for rows.Next() {
		var o models.Order
		if err := rows.Scan(&o.ID, &o.UserId, &o.Price, &o.Status, &o.CreatedAt, &o.UpdatedAt); err != nil {
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
