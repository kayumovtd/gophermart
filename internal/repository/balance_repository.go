package repository

import (
	"context"
	"errors"
	"time"
)

var (
	ErrInsufficientFunds     = errors.New("insufficient funds")
	ErrWithdrawalOrderExists = errors.New("withdrawal order already exists")
)

type Balance struct {
	Current   float64
	Withdrawn float64
}

type Withdrawal struct {
	Order       string
	Sum         float64
	ProcessedAt time.Time
}

type BalanceRepository interface {
	GetBalance(ctx context.Context, userID int64) (Balance, error)
	Withdraw(ctx context.Context, userID int64, order string, sum float64, processedAt time.Time) error
	ListWithdrawals(ctx context.Context, userID int64) ([]Withdrawal, error)
}
