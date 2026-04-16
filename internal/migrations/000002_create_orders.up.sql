DO $$
BEGIN
    CREATE TYPE order_status AS ENUM ('NEW', 'PROCESSING', 'INVALID', 'PROCESSED');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END$$;

CREATE TABLE IF NOT EXISTS orders (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    number TEXT NOT NULL UNIQUE,
    status order_status NOT NULL,
    accrual DOUBLE PRECISION,
    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_orders_user_uploaded_at_desc
    ON orders (user_id, uploaded_at DESC);

-- Скорее всего ещё пригодится частичный индекс на status когда дойду до процессора заказов