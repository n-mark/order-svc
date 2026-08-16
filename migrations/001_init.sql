-- Orders: created by users, processed asynchronously via billing-svc
CREATE TABLE IF NOT EXISTS orders (
    id           UUID PRIMARY KEY,
    user_id      BIGINT         NOT NULL,
    seller_id    BIGINT         NOT NULL DEFAULT 0,
    receiver_id  BIGINT         NOT NULL DEFAULT 0,
    price        NUMERIC(18, 2) NOT NULL,
    status       VARCHAR(32)    NOT NULL DEFAULT 'CREATED',
    items        JSONB          NOT NULL DEFAULT '[]'::jsonb,
    delivery     JSONB,
    created_at   TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ    NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id);
CREATE INDEX IF NOT EXISTS idx_orders_status  ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_items_advert_id ON orders USING GIN ((items::jsonb));

-- Outbox of processed events for idempotency
CREATE TABLE IF NOT EXISTS processed_events (
    event_id     UUID PRIMARY KEY,
    event_type   TEXT        NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
