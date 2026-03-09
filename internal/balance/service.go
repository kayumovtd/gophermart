package balance

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"

	"github.com/kayumovtd/gophermart/internal/ordervalidation"
	"github.com/kayumovtd/gophermart/internal/repository"
)

var (
	ErrInvalidInput       = errors.New("invalid input")
	ErrInvalidOrderNumber = errors.New("invalid order number")
	ErrInsufficientFunds  = errors.New("insufficient funds")
)

type Balance = repository.Balance
type Withdrawal = repository.Withdrawal

type Service struct {
	repo repository.BalanceRepository
	now  func() time.Time
}

func NewService(repo repository.BalanceRepository) *Service {
	return &Service{
		repo: repo,
		now:  time.Now,
	}
}

func (s *Service) Get(ctx context.Context, userID int64) (Balance, error) {
	if userID <= 0 {
		return Balance{}, ErrInvalidInput
	}

	return s.repo.GetBalance(ctx, userID)
}

func (s *Service) Withdraw(ctx context.Context, userID int64, order string, sum float64) error {
	if userID <= 0 || math.IsNaN(sum) || math.IsInf(sum, 0) || sum <= 0 {
		return ErrInvalidInput
	}

	order = strings.TrimSpace(order)
	if !ordervalidation.IsValidOrderNumber(order) {
		return ErrInvalidOrderNumber
	}

	err := s.repo.Withdraw(ctx, userID, order, sum, s.now().UTC())
	switch {
	case errors.Is(err, repository.ErrInsufficientFunds):
		return ErrInsufficientFunds
	case errors.Is(err, repository.ErrWithdrawalOrderExists):
		return ErrInvalidOrderNumber
	}

	return err
}

func (s *Service) ListWithdrawals(ctx context.Context, userID int64) ([]Withdrawal, error) {
	if userID <= 0 {
		return nil, ErrInvalidInput
	}

	return s.repo.ListWithdrawals(ctx, userID)
}
