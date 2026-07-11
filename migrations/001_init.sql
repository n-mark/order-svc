-- Billing accounts: one per user
CREATE TABLE IF NOT EXISTS orders (
    id          UUID PRIMARY KEY,
    user_id     BIGINT      NOT NULL UNIQUE,
    status      VARCHAR,
    total_cost     NUMERIC(18, 2) NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Outbox of processed events for idempotency
CREATE TABLE IF NOT EXISTS processed_events (
    event_id    UUID PRIMARY KEY,
    event_type  TEXT        NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_billing_accounts_user_id ON billing_accounts(user_id);
