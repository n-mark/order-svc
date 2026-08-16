-- v2: orders carry seller/receiver, item list and the chosen delivery details.
-- These columns are now created in 001_init.sql; kept here for idempotent upgrades.
ALTER TABLE orders ADD COLUMN IF NOT EXISTS seller_id   BIGINT NOT NULL DEFAULT 0;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS receiver_id BIGINT NOT NULL DEFAULT 0;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS items       JSONB  NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS delivery    JSONB;

-- New default status for v2 (was 'pending').
ALTER TABLE orders ALTER COLUMN status SET DEFAULT 'CREATED';

-- Index for deduplication by advert_id inside items JSONB.
CREATE INDEX IF NOT EXISTS idx_orders_items_advert_id ON orders USING GIN ((items::jsonb));
