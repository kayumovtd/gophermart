CREATE TABLE IF NOT EXISTS withdrawals (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    order_number TEXT NOT NULL UNIQUE,
    sum DOUBLE PRECISION NOT NULL CHECK (sum > 0),
    processed_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_withdrawals_user_processed_at_desc
    ON withdrawals (user_id, processed_at DESC);
