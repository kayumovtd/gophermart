//go:generate mockgen -source=../repository/order_repository.go -package=orders -destination=mock_order_repository_test.go -mock_names OrderRepository=MockOrderRepository
//go:generate mockgen -source=processor.go -package=orders -destination=mock_accrual_client_test.go -mock_names accrualClient=MockAccrualClient

package orders

import (
	"context"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/kayumovtd/gophermart/internal/accrual"
	"github.com/kayumovtd/gophermart/internal/logger"
	"github.com/kayumovtd/gophermart/internal/repository"
)

func TestProcessorProcessOneProcessed(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMockOrderRepository(ctrl)
	accrualClient := NewMockAccrualClient(ctrl)
	log := logger.NewNoOp()

	processor := NewProcessor(repo, accrualClient, log, 1)
	now := time.Date(2026, 2, 28, 10, 0, 0, 0, time.UTC)
	processor.now = func() time.Time { return now }

	accrualClient.EXPECT().
		GetOrder(gomock.Any(), "79927398713").
		Return(accrual.Order{
			Number:  "79927398713",
			Status:  accrual.StatusProcessed,
			Accrual: floatPtr(42.5),
		}, nil)

	repo.EXPECT().
		Complete(gomock.Any(), int64(1), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ int64, completion repository.OrderCompletion) error {
			if completion.Status != StatusProcessed {
				t.Fatalf("status = %q, want %q", completion.Status, StatusProcessed)
			}
			if completion.Accrual == nil || *completion.Accrual != 42.5 {
				t.Fatalf("accrual = %v, want %v", completion.Accrual, 42.5)
			}
			return nil
		})

	nextAttemptAt, stopBatch, err := processor.processOne(context.Background(), repository.Order{
		ID:     1,
		Number: "79927398713",
		Status: StatusProcessing,
	})
	if err != nil {
		t.Fatalf("processOne() error = %v", err)
	}
	if stopBatch {
		t.Fatalf("stopBatch = true, want false")
	}
	if nextAttemptAt != now {
		t.Fatalf("nextAttemptAt = %v, want %v", nextAttemptAt, now)
	}
}

func TestProcessorProcessOneNotRegistered(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMockOrderRepository(ctrl)
	accrualClient := NewMockAccrualClient(ctrl)
	log := logger.NewNoOp()

	processor := NewProcessor(repo, accrualClient, log, 1)
	now := time.Date(2026, 2, 28, 10, 0, 0, 0, time.UTC)
	processor.now = func() time.Time { return now }

	accrualClient.EXPECT().
		GetOrder(gomock.Any(), "79927398713").
		Return(accrual.Order{}, accrual.ErrOrderNotRegistered)

	repo.EXPECT().
		Requeue(gomock.Any(), gomock.Eq([]int64{1}), now.Add(defaultRetryDelay), now, gomock.Nil()).
		Return(nil)

	nextAttemptAt, stopBatch, err := processor.processOne(context.Background(), repository.Order{
		ID:     1,
		Number: "79927398713",
		Status: StatusProcessing,
	})
	if err != nil {
		t.Fatalf("processOne() error = %v", err)
	}
	if stopBatch {
		t.Fatalf("stopBatch = true, want false")
	}
	if nextAttemptAt != now.Add(defaultRetryDelay) {
		t.Fatalf("nextAttemptAt = %v, want %v", nextAttemptAt, now.Add(defaultRetryDelay))
	}
}

func TestProcessorProcessOneRateLimitedPauses(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMockOrderRepository(ctrl)
	accrualClient := NewMockAccrualClient(ctrl)
	log := logger.NewNoOp()

	processor := NewProcessor(repo, accrualClient, log, 1)
	now := time.Date(2026, 2, 28, 10, 0, 0, 0, time.UTC)
	processor.now = func() time.Time { return now }

	accrualClient.EXPECT().
		GetOrder(gomock.Any(), "79927398713").
		Return(accrual.Order{}, &accrual.RateLimitError{RetryAfter: time.Minute})

	repo.EXPECT().
		Requeue(gomock.Any(), gomock.Eq([]int64{1}), now.Add(time.Minute), now, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ []int64, _ time.Time, _ time.Time, lastError *string) error {
			if lastError == nil {
				t.Fatalf("lastError is nil")
			}
			if *lastError != "accrual rate limited" {
				t.Fatalf("lastError = %q, want %q", *lastError, "accrual rate limited")
			}
			return nil
		})

	nextAttemptAt, stopBatch, err := processor.processOne(context.Background(), repository.Order{
		ID:     1,
		Number: "79927398713",
		Status: StatusProcessing,
	})
	if err != nil {
		t.Fatalf("processOne() error = %v", err)
	}
	if !stopBatch {
		t.Fatalf("stopBatch = false, want true")
	}
	if nextAttemptAt != now.Add(time.Minute) {
		t.Fatalf("nextAttemptAt = %v, want %v", nextAttemptAt, now.Add(time.Minute))
	}
	if processor.pauseTime() != now.Add(time.Minute) {
		t.Fatalf("pauseUntil = %v, want %v", processor.pauseTime(), now.Add(time.Minute))
	}
}

func TestProcessorProcessBatchStopsOnPauseAndReleasesRemaining(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMockOrderRepository(ctrl)
	accrualClient := NewMockAccrualClient(ctrl)
	log := logger.NewNoOp()

	now := time.Date(2026, 2, 28, 10, 0, 0, 0, time.UTC)
	processor := NewProcessor(repo, accrualClient, log, 1)
	processor.now = func() time.Time { return now }

	gomock.InOrder(
		accrualClient.EXPECT().
			GetOrder(gomock.Any(), "1111111116").
			Return(accrual.Order{}, &accrual.RateLimitError{RetryAfter: time.Minute}),
		repo.EXPECT().
			Requeue(gomock.Any(), gomock.Eq([]int64{1}), now.Add(time.Minute), now, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ []int64, _ time.Time, _ time.Time, lastError *string) error {
				if lastError == nil {
					t.Fatalf("first lastError is nil")
				}
				return nil
			}),
		repo.EXPECT().
			Requeue(gomock.Any(), gomock.Eq([]int64{2, 3}), now.Add(time.Minute), now, gomock.Nil()).
			Return(nil),
	)

	processor.processBatch(context.Background(), []repository.Order{
		{ID: 1, Number: "1111111116", Status: StatusProcessing},
		{ID: 2, Number: "2222222222", Status: StatusProcessing},
		{ID: 3, Number: "3333333333", Status: StatusProcessing},
	})
}

func TestProcessorWorkerClaimsWithLease(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMockOrderRepository(ctrl)
	accrualClient := NewMockAccrualClient(ctrl)
	log := logger.NewNoOp()

	now := time.Date(2026, 2, 28, 10, 0, 0, 0, time.UTC)
	processor := NewProcessor(repo, accrualClient, log, 1)
	processor.now = func() time.Time { return now }
	processor.pollInterval = time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	processor.runWorker(ctx, 1)
}

func floatPtr(v float64) *float64 {
	return &v
}
