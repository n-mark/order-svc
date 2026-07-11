package store

import (
	"billing-svc/internal/models"
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PgStore struct {
	db *pgxpool.Pool
}

func NewPgStore(db *pgxpool.Pool) *PgStore {
	return &PgStore{db: db}
}

func (p *PgStore) Save(ctx context.Context, profile models.Order) (models.Billing, error) {
	id := uuid.New()

	row := p.db.QueryRow(ctx, `
		INSERT INTO orders (id, user_id, balance)
		VALUES ($1, $2, 0)
		ON CONFLICT (id) DO UPDATE SET updated_at = now()
		RETURNING id, user_id, balance, created_at, updated_at
	`, id, profile.UserId)

	b := models.Billing{}
	if err := row.Scan(&b.ID, &b.UserId, &b.Balance, &b.CreatedAt, &b.UpdatedAt); err != nil {
		return models.Billing{}, err
	}
	return b, nil
}

func (p *PgStore) Update(ctx context.Context, profile models.Order) (models.Billing, error) {
	id := uuid.New()

	row := p.db.QueryRow(ctx, `
		INSERT INTO billing_accounts (id, user_id, balance)
		VALUES ($1, $2, 0)
		ON CONFLICT (user_id) DO UPDATE SET updated_at = now()
		RETURNING id, user_id, balance, created_at, updated_at
	`, id, profile.UserId)

	b := models.Billing{}
	if err := row.Scan(&b.ID, &b.UserId, &b.Balance, &b.CreatedAt, &b.UpdatedAt); err != nil {
		return models.Billing{}, err
	}
	return b, nil
}

func (p *PgStore) GetByUserId(ctx context.Context, userId int64) (models.Billing, error) {
	row := p.db.QueryRow(ctx, `
		SELECT id, user_id, balance, created_at, updated_at
		FROM billing_accounts
		WHERE user_id = $1
	`, userId)

	b := models.Billing{}
	if err := row.Scan(&b.ID, &b.UserId, &b.Balance, &b.CreatedAt, &b.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Billing{}, ErrNotFound
		}
		return models.Billing{}, err
	}
	return b, nil
}

var (
	ErrNotFound          = errors.New("billing account not found")
	ErrInsufficientFunds = errors.New("insufficient funds")
)

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
