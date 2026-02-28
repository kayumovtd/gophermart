package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kayumovtd/gophermart/internal/repository"
)

type OrderStorage struct {
	pool *pgxpool.Pool
}

func NewOrderStorage(pool *pgxpool.Pool) *OrderStorage {
	return &OrderStorage{pool: pool}
}

func (r *OrderStorage) Create(ctx context.Context, userID int64, number, status string) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const insertOrderQuery = `
INSERT INTO orders (user_id, number, status)
VALUES ($1, $2, $3)
RETURNING id;
	`

	var orderID int64
	if err := tx.QueryRow(ctx, insertOrderQuery, userID, number, status).Scan(&orderID); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return repository.ErrOrderAlreadyExists
		}
		return fmt.Errorf("insert order: %w", err)
	}

	now := time.Now().UTC()
	const insertJobQuery = `
INSERT INTO order_jobs (order_id, next_attempt_at, attempt, created_at, updated_at)
VALUES ($1, $2, 0, $2, $2);
	`
	if _, err := tx.Exec(ctx, insertJobQuery, orderID, now); err != nil {
		return fmt.Errorf("insert order job: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

func (r *OrderStorage) GetByNumber(ctx context.Context, number string) (repository.Order, error) {
	const query = `
SELECT id, user_id, number, status, accrual, uploaded_at
FROM orders
WHERE number = $1;
	`

	order, err := scanOrder(r.pool.QueryRow(ctx, query, number))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return repository.Order{}, repository.ErrOrderNotFound
		}
		return repository.Order{}, fmt.Errorf("get order by number: %w", err)
	}

	return order, nil
}

func (r *OrderStorage) ListByUserID(ctx context.Context, userID int64) ([]repository.Order, error) {
	const query = `
SELECT id, user_id, number, status, accrual, uploaded_at
FROM orders
WHERE user_id = $1
ORDER BY uploaded_at DESC;
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list orders by user id: %w", err)
	}
	defer rows.Close()

	orders := make([]repository.Order, 0)
	for rows.Next() {
		order, scanErr := scanOrder(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan order: %w", scanErr)
		}
		orders = append(orders, order)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("rows error: %w", rows.Err())
	}

	return orders, nil
}

func (r *OrderStorage) ClaimForProcessing(ctx context.Context, claim repository.OrderClaim) ([]repository.Order, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	orderIDs, err := r.claimOrderIDs(ctx, tx, claim)
	if err != nil {
		return nil, err
	}

	if len(orderIDs) == 0 {
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit empty claim tx: %w", err)
		}
		return nil, nil
	}

	if err := r.markClaimedJobs(ctx, tx, orderIDs, claim); err != nil {
		return nil, err
	}

	if err := r.markOrdersProcessing(ctx, tx, orderIDs); err != nil {
		return nil, err
	}

	orders, err := r.listOrdersByIDs(ctx, tx, orderIDs)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit claim tx: %w", err)
	}

	return orders, nil
}

func (r *OrderStorage) Complete(ctx context.Context, orderID int64, completion repository.OrderCompletion) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const updateOrderQuery = `
UPDATE orders
SET status = $2,
    accrual = $3
WHERE id = $1;
	`
	tag, err := tx.Exec(ctx, updateOrderQuery, orderID, completion.Status, completion.Accrual)
	if err != nil {
		return fmt.Errorf("update order: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return repository.ErrOrderNotFound
	}

	const deleteJobQuery = `
DELETE FROM order_jobs
WHERE order_id = $1;
	`
	if _, err := tx.Exec(ctx, deleteJobQuery, orderID); err != nil {
		return fmt.Errorf("delete order job: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

func (r *OrderStorage) Requeue(
	ctx context.Context,
	orderIDs []int64,
	nextAttemptAt, updatedAt time.Time,
	lastError *string,
) error {
	if len(orderIDs) == 0 {
		return nil
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const updateOrdersQuery = `
UPDATE orders
SET status = 'NEW'
WHERE id = ANY($1);
	`
	if _, err := tx.Exec(ctx, updateOrdersQuery, orderIDs); err != nil {
		return fmt.Errorf("set orders NEW: %w", err)
	}

	const updateJobsQuery = `
UPDATE order_jobs
SET next_attempt_at = $2,
    locked_by = NULL,
    locked_until = NULL,
    updated_at = $3,
    last_error = $4
WHERE order_id = ANY($1);
	`
	if _, err := tx.Exec(ctx, updateJobsQuery, orderIDs, nextAttemptAt, updatedAt, lastError); err != nil {
		return fmt.Errorf("update order jobs: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

func (r *OrderStorage) claimOrderIDs(
	ctx context.Context,
	tx pgx.Tx,
	claim repository.OrderClaim,
) ([]int64, error) {
	const selectIDsQuery = `
SELECT j.order_id
FROM order_jobs AS j
JOIN orders AS o ON o.id = j.order_id
WHERE o.status IN ('NEW', 'PROCESSING')
  AND j.next_attempt_at <= $1
  AND (j.locked_until IS NULL OR j.locked_until < $1)
ORDER BY j.next_attempt_at ASC, j.order_id ASC
LIMIT $2
FOR UPDATE OF j SKIP LOCKED;
	`

	rows, err := tx.Query(ctx, selectIDsQuery, claim.Now, claim.Limit)
	if err != nil {
		return nil, fmt.Errorf("select claim ids: %w", err)
	}
	defer rows.Close()

	orderIDs := make([]int64, 0, claim.Limit)
	for rows.Next() {
		var id int64
		if scanErr := rows.Scan(&id); scanErr != nil {
			return nil, fmt.Errorf("scan claim id: %w", scanErr)
		}
		orderIDs = append(orderIDs, id)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("claim ids rows error: %w", rows.Err())
	}

	return orderIDs, nil
}

func (r *OrderStorage) markClaimedJobs(
	ctx context.Context,
	tx pgx.Tx,
	orderIDs []int64,
	claim repository.OrderClaim,
) error {
	const updateJobsQuery = `
UPDATE order_jobs
SET attempt = attempt + 1,
    locked_by = $2,
    locked_until = $3,
    updated_at = $4,
    last_error = NULL
WHERE order_id = ANY($1);
	`

	if _, err := tx.Exec(ctx, updateJobsQuery, orderIDs, claim.WorkerID, claim.LeaseUntil, claim.Now); err != nil {
		return fmt.Errorf("update claimed jobs: %w", err)
	}

	return nil
}

func (r *OrderStorage) markOrdersProcessing(
	ctx context.Context,
	tx pgx.Tx,
	orderIDs []int64,
) error {
	const updateOrdersQuery = `
UPDATE orders
SET status = 'PROCESSING'
WHERE id = ANY($1);
	`

	if _, err := tx.Exec(ctx, updateOrdersQuery, orderIDs); err != nil {
		return fmt.Errorf("update claimed orders: %w", err)
	}

	return nil
}

func (r *OrderStorage) listOrdersByIDs(
	ctx context.Context,
	tx pgx.Tx,
	orderIDs []int64,
) ([]repository.Order, error) {
	const selectOrdersQuery = `
SELECT id, user_id, number, status, accrual, uploaded_at
FROM orders
WHERE id = ANY($1)
ORDER BY id ASC;
	`

	rows, err := tx.Query(ctx, selectOrdersQuery, orderIDs)
	if err != nil {
		return nil, fmt.Errorf("select claimed orders: %w", err)
	}
	defer rows.Close()

	orders := make([]repository.Order, 0, len(orderIDs))
	for rows.Next() {
		order, scanErr := scanOrder(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan claimed order: %w", scanErr)
		}
		orders = append(orders, order)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("claimed orders rows error: %w", rows.Err())
	}

	return orders, nil
}

type orderScanner interface {
	Scan(dest ...any) error
}

func scanOrder(scanner orderScanner) (repository.Order, error) {
	var (
		order        repository.Order
		accrualValue sql.NullFloat64
	)

	if err := scanner.Scan(
		&order.ID,
		&order.UserID,
		&order.Number,
		&order.Status,
		&accrualValue,
		&order.UploadedAt,
	); err != nil {
		return repository.Order{}, err
	}

	if accrualValue.Valid {
		accrual := accrualValue.Float64
		order.Accrual = &accrual
	}

	return order, nil
}
