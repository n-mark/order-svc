-- Orders: created by users, processed asynchronously via billing-svc
CREATE TABLE IF NOT EXISTS orders (
    id           UUID PRIMARY KEY,
    user_id      BIGINT         NOT NULL,
    price        NUMERIC(18, 2) NOT NULL,
    status       VARCHAR(32)    NOT NULL DEFAULT 'pending',
    created_at   TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ    NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id);
CREATE INDEX IF NOT EXISTS idx_orders_status  ON orders(status);

-- Outbox of processed events for idempotency
CREATE TABLE IF NOT EXISTS processed_events (
    event_id     UUID PRIMARY KEY,
    event_type   TEXT        NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
