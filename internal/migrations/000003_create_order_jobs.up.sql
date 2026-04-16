CREATE TABLE IF NOT EXISTS order_jobs (
    order_id BIGINT PRIMARY KEY REFERENCES orders(id) ON DELETE CASCADE,
    next_attempt_at TIMESTAMPTZ NOT NULL,
    attempt INTEGER NOT NULL DEFAULT 0,
    locked_until TIMESTAMPTZ,
    locked_by TEXT,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_order_jobs_queue
    ON order_jobs (next_attempt_at, order_id);

CREATE INDEX IF NOT EXISTS idx_order_jobs_locked_until
    ON order_jobs (locked_until);

INSERT INTO order_jobs (order_id, next_attempt_at, attempt, created_at, updated_at)
SELECT o.id, NOW(), 0, NOW(), NOW()
FROM orders AS o
WHERE o.status IN ('NEW', 'PROCESSING')
  AND NOT EXISTS (
      SELECT 1
      FROM order_jobs AS j
      WHERE j.order_id = o.id
  );
