package balance

import (
	"context"

	corebalance "github.com/kayumovtd/gophermart/internal/balance"
)

type Service interface {
	Get(ctx context.Context, userID int64) (corebalance.Balance, error)
	Withdraw(ctx context.Context, userID int64, order string, sum float64) error
	ListWithdrawals(ctx context.Context, userID int64) ([]corebalance.Withdrawal, error)
}
