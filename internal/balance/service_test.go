package balance

//go:generate mockgen -source=../repository/balance_repository.go -package=balance -destination=mock_balance_repository_test.go -mock_names BalanceRepository=MockBalanceRepository

import (
	"context"
	"errors"
	"testing"

	"github.com/kayumovtd/gophermart/internal/repository"
	"go.uber.org/mock/gomock"
)

func TestGetBalanceSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMockBalanceRepository(ctrl)

	repo.EXPECT().
		GetBalance(gomock.Any(), int64(1)).
		Return(repository.Balance{Current: 10, Withdrawn: 2}, nil)

	svc := NewService(repo)

	balance, err := svc.Get(context.Background(), 1)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if balance.Current != 10 {
		t.Fatalf("Get().Current = %v, want 10", balance.Current)
	}

	if balance.Withdrawn != 2 {
		t.Fatalf("Get().Withdrawn = %v, want 2", balance.Withdrawn)
	}
}

func TestGetBalanceInvalidInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc := NewService(NewMockBalanceRepository(ctrl))

	_, err := svc.Get(context.Background(), 0)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("Get() error = %v, want %v", err, ErrInvalidInput)
	}
}

func TestWithdrawInvalidOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc := NewService(NewMockBalanceRepository(ctrl))

	err := svc.Withdraw(context.Background(), 1, "123", 10)
	if !errors.Is(err, ErrInvalidOrderNumber) {
		t.Fatalf("Withdraw() error = %v, want %v", err, ErrInvalidOrderNumber)
	}
}

func TestWithdrawInsufficientFunds(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMockBalanceRepository(ctrl)

	repo.EXPECT().
		Withdraw(gomock.Any(), int64(1), "79927398713", 10.0, gomock.Any()).
		Return(repository.ErrInsufficientFunds)

	svc := NewService(repo)
	err := svc.Withdraw(context.Background(), 1, "79927398713", 10)
	if !errors.Is(err, ErrInsufficientFunds) {
		t.Fatalf("Withdraw() error = %v, want %v", err, ErrInsufficientFunds)
	}
}

func TestWithdrawOrderAlreadyExists(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMockBalanceRepository(ctrl)

	repo.EXPECT().
		Withdraw(gomock.Any(), int64(1), "79927398713", 10.0, gomock.Any()).
		Return(repository.ErrWithdrawalOrderExists)

	svc := NewService(repo)
	err := svc.Withdraw(context.Background(), 1, "79927398713", 10)
	if !errors.Is(err, ErrInvalidOrderNumber) {
		t.Fatalf("Withdraw() error = %v, want %v", err, ErrInvalidOrderNumber)
	}
}

func TestListWithdrawalsInvalidInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc := NewService(NewMockBalanceRepository(ctrl))

	_, err := svc.ListWithdrawals(context.Background(), -1)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("ListWithdrawals() error = %v, want %v", err, ErrInvalidInput)
	}
}
