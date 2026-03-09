package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kayumovtd/gophermart/internal/repository"
)

type BalanceStorage struct {
	pool *pgxpool.Pool
}

func NewBalanceStorage(pool *pgxpool.Pool) *BalanceStorage {
	return &BalanceStorage{pool: pool}
}

func (s *BalanceStorage) GetBalance(ctx context.Context, userID int64) (repository.Balance, error) {
	return s.getBalance(ctx, s.pool, userID, "get balance")
}

func (s *BalanceStorage) Withdraw(
	ctx context.Context,
	userID int64,
	order string,
	sum float64,
	processedAt time.Time,
) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const lockUserQuery = `
SELECT id
FROM users
WHERE id = $1
FOR UPDATE;
	`

	var lockedUserID int64
	if err := tx.QueryRow(ctx, lockUserQuery, userID).Scan(&lockedUserID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return repository.ErrUserNotFound
		}
		return fmt.Errorf("lock user: %w", err)
	}

	balance, err := s.getBalance(ctx, tx, userID, "get balance in tx")
	if err != nil {
		return err
	}
	if balance.Current < sum {
		return repository.ErrInsufficientFunds
	}

	const insertQuery = `
INSERT INTO withdrawals (user_id, order_number, sum, processed_at)
VALUES ($1, $2, $3, $4);
	`

	if _, err := tx.Exec(ctx, insertQuery, userID, order, sum, processedAt); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return repository.ErrWithdrawalOrderExists
		}
		return fmt.Errorf("insert withdrawal: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

func (s *BalanceStorage) ListWithdrawals(ctx context.Context, userID int64) ([]repository.Withdrawal, error) {
	const query = `
SELECT order_number, sum, processed_at
FROM withdrawals
WHERE user_id = $1
ORDER BY processed_at DESC;
	`

	rows, err := s.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list withdrawals: %w", err)
	}
	defer rows.Close()

	items := make([]repository.Withdrawal, 0)
	for rows.Next() {
		var item repository.Withdrawal
		if scanErr := rows.Scan(&item.Order, &item.Sum, &item.ProcessedAt); scanErr != nil {
			return nil, fmt.Errorf("scan withdrawal: %w", scanErr)
		}
		items = append(items, item)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("withdrawals rows error: %w", rows.Err())
	}

	return items, nil
}

type balanceQueryer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func (s *BalanceStorage) getBalance(
	ctx context.Context,
	queryer balanceQueryer,
	userID int64,
	op string,
) (repository.Balance, error) {
	const query = `
SELECT
	COALESCE((
		SELECT SUM(o.accrual)
		FROM orders AS o
		WHERE o.user_id = $1
		  AND o.status = 'PROCESSED'
	), 0),
	COALESCE((
		SELECT SUM(w.sum)
		FROM withdrawals AS w
		WHERE w.user_id = $1
	), 0);
	`

	var balance repository.Balance
	if err := queryer.QueryRow(ctx, query, userID).Scan(&balance.Current, &balance.Withdrawn); err != nil {
		return repository.Balance{}, fmt.Errorf("%s: %w", op, err)
	}

	balance.Current -= balance.Withdrawn
	return balance, nil
}
