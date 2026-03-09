package orders

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"github.com/kayumovtd/gophermart/internal/accrual"
	"github.com/kayumovtd/gophermart/internal/logger"
	"github.com/kayumovtd/gophermart/internal/repository"
)

const (
	defaultWorkerCount  = 4
	defaultBatchSize    = 20
	defaultLeaseTTL     = 30 * time.Second
	defaultRetryDelay   = 5 * time.Second
	defaultPollInterval = time.Second
)

type accrualClient interface {
	GetOrder(ctx context.Context, number string) (accrual.Order, error)
}

type Processor struct {
	repo         repository.OrderRepository
	accrual      accrualClient
	log          *logger.Logger
	workerCount  int
	batchSize    int
	leaseTTL     time.Duration
	retryDelay   time.Duration
	pollInterval time.Duration
	now          func() time.Time
	instanceID   string
	pauseUntil   atomic.Int64
}

func NewProcessor(
	repo repository.OrderRepository,
	accrualClient accrualClient,
	log *logger.Logger,
	workerCount int,
) *Processor {
	if workerCount <= 0 {
		workerCount = defaultWorkerCount
	}

	return &Processor{
		repo:         repo,
		accrual:      accrualClient,
		log:          log,
		workerCount:  workerCount,
		batchSize:    defaultBatchSize,
		leaseTTL:     defaultLeaseTTL,
		retryDelay:   defaultRetryDelay,
		pollInterval: defaultPollInterval,
		now:          time.Now,
		instanceID:   fmt.Sprintf("pid-%d-%d", os.Getpid(), time.Now().UnixNano()),
	}
}

func (p *Processor) Run(ctx context.Context) {
	var wg sync.WaitGroup
	for workerIndex := 0; workerIndex < p.workerCount; workerIndex++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			p.runWorker(ctx, id)
		}(workerIndex + 1)
	}

	wg.Wait()
}

func (p *Processor) runWorker(ctx context.Context, workerID int) {
	workerName := fmt.Sprintf("%s-worker-%d", p.instanceID, workerID)

	for {
		if err := ctx.Err(); err != nil {
			return
		}
		if err := p.waitForPause(ctx); err != nil {
			return
		}

		now := p.now().UTC()
		claimed, err := p.repo.ClaimForProcessing(ctx, repository.OrderClaim{
			WorkerID:   workerName,
			Now:        now,
			Limit:      p.batchSize,
			LeaseUntil: now.Add(p.leaseTTL),
		})
		if err != nil {
			p.log.Error("claim orders for processing", zap.Error(err))
			if err := p.sleep(ctx, p.pollInterval); err != nil {
				return
			}
			continue
		}
		if len(claimed) == 0 {
			if err := p.sleep(ctx, p.pollInterval); err != nil {
				return
			}
			continue
		}

		p.processBatch(ctx, claimed)
	}
}

func (p *Processor) processBatch(ctx context.Context, claimed []repository.Order) {
	for index, order := range claimed {
		if err := ctx.Err(); err != nil {
			p.releaseRemaining(ctx, claimed[index:], p.now().UTC())
			return
		}
		if err := p.waitForPause(ctx); err != nil {
			nextAttemptAt := p.pauseTime()
			if nextAttemptAt.IsZero() {
				nextAttemptAt = p.now().UTC()
			}
			p.releaseRemaining(ctx, claimed[index:], nextAttemptAt)
			return
		}

		nextAttemptAt, stopBatch, err := p.processOne(ctx, order)
		if err != nil {
			p.log.Error("process order", zap.String("number", order.Number), zap.Error(err))
		}
		if stopBatch {
			p.releaseRemaining(ctx, claimed[index+1:], nextAttemptAt)
			return
		}
	}
}

func (p *Processor) processOne(ctx context.Context, order repository.Order) (time.Time, bool, error) {
	now := p.now().UTC()

	result, err := p.accrual.GetOrder(ctx, order.Number)
	if err != nil {
		var rateLimitErr *accrual.RateLimitError
		switch {
		case errors.Is(err, accrual.ErrOrderNotRegistered):
			return now.Add(p.retryDelay), false, p.repo.Requeue(ctx, []int64{order.ID}, now.Add(p.retryDelay), now, nil)
		case errors.As(err, &rateLimitErr):
			pauseUntil := now.Add(rateLimitErr.RetryAfter)
			p.extendPauseUntil(pauseUntil)
			lastError := "accrual rate limited"
			return pauseUntil, true, p.repo.Requeue(ctx, []int64{order.ID}, pauseUntil, now, &lastError)
		default:
			lastError := err.Error()
			return now.Add(p.retryDelay), false, p.repo.Requeue(ctx, []int64{order.ID}, now.Add(p.retryDelay), now, &lastError)
		}
	}

	switch result.Status {
	case accrual.StatusInvalid:
		return now, false, p.repo.Complete(ctx, order.ID, repository.OrderCompletion{
			Status:  StatusInvalid,
			Accrual: nil,
		})
	case accrual.StatusProcessed:
		return now, false, p.repo.Complete(ctx, order.ID, repository.OrderCompletion{
			Status:  StatusProcessed,
			Accrual: result.Accrual,
		})
	case accrual.StatusRegistered, accrual.StatusProcessing:
		return now.Add(p.retryDelay), false, p.repo.Requeue(ctx, []int64{order.ID}, now.Add(p.retryDelay), now, nil)
	default:
		lastError := "unknown accrual status"
		return now.Add(p.retryDelay), false, p.repo.Requeue(ctx, []int64{order.ID}, now.Add(p.retryDelay), now, &lastError)
	}
}

func (p *Processor) releaseRemaining(ctx context.Context, orders []repository.Order, nextAttemptAt time.Time) {
	if len(orders) == 0 {
		return
	}

	releaseCtx := ctx
	if ctx.Err() != nil {
		var cancel context.CancelFunc
		releaseCtx, cancel = context.WithTimeout(context.Background(), time.Second)
		defer cancel()
	}

	ids := make([]int64, 0, len(orders))
	for _, order := range orders {
		ids = append(ids, order.ID)
	}

	if err := p.repo.Requeue(releaseCtx, ids, nextAttemptAt, p.now().UTC(), nil); err != nil && !errors.Is(err, context.Canceled) {
		p.log.Error("release remaining orders", zap.Error(err))
	}
}

func (p *Processor) waitForPause(ctx context.Context) error {
	for {
		now := p.now().UTC()
		pauseUntil := p.pauseTime()
		if !now.Before(pauseUntil) {
			return nil
		}
		if err := p.sleep(ctx, pauseUntil.Sub(now)); err != nil {
			return err
		}
	}
}

func (p *Processor) extendPauseUntil(until time.Time) {
	target := until.UnixNano()
	for {
		current := p.pauseUntil.Load()
		if current >= target {
			return
		}
		if p.pauseUntil.CompareAndSwap(current, target) {
			return
		}
	}
}

func (p *Processor) pauseTime() time.Time {
	value := p.pauseUntil.Load()
	if value == 0 {
		return time.Time{}
	}

	return time.Unix(0, value).UTC()
}

func (p *Processor) sleep(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
