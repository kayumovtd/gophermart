package orders

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/kayumovtd/gophermart/internal/ordervalidation"
	"github.com/kayumovtd/gophermart/internal/repository"
)

const (
	StatusNew        = "NEW"
	StatusProcessing = "PROCESSING"
	StatusInvalid    = "INVALID"
	StatusProcessed  = "PROCESSED"
)

var (
	ErrInvalidInput               = errors.New("invalid input")
	ErrInvalidOrderNumber         = errors.New("invalid order number")
	ErrOrderUploadedByAnotherUser = errors.New("order uploaded by another user")
)

type UploadResult int

const (
	UploadResultAccepted UploadResult = iota
	UploadResultAlreadyUploadedByUser
)

type Order struct {
	Number     string
	Status     string
	Accrual    *float64
	UploadedAt time.Time
}

type Service struct {
	repo repository.OrderRepository
}

func NewService(repo repository.OrderRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Upload(ctx context.Context, userID int64, number string) (UploadResult, error) {
	if userID <= 0 {
		return 0, ErrInvalidInput
	}

	number = strings.TrimSpace(number)
	if !ordervalidation.IsValidOrderNumber(number) {
		return 0, ErrInvalidOrderNumber
	}

	existing, err := s.repo.GetByNumber(ctx, number)
	switch {
	case err == nil:
		if existing.UserID == userID {
			return UploadResultAlreadyUploadedByUser, nil
		}
		return 0, ErrOrderUploadedByAnotherUser
	case !errors.Is(err, repository.ErrOrderNotFound):
		return 0, err
	}

	if err := s.repo.Create(ctx, userID, number, StatusNew); err != nil {
		if errors.Is(err, repository.ErrOrderAlreadyExists) {
			return s.handleConcurrentCreate(ctx, userID, number)
		}
		return 0, err
	}

	return UploadResultAccepted, nil
}

func (s *Service) List(ctx context.Context, userID int64) ([]Order, error) {
	if userID <= 0 {
		return nil, ErrInvalidInput
	}

	items, err := s.repo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	out := make([]Order, 0, len(items))
	for _, item := range items {
		out = append(out, Order{
			Number:     item.Number,
			Status:     item.Status,
			Accrual:    item.Accrual,
			UploadedAt: item.UploadedAt,
		})
	}

	return out, nil
}

func (s *Service) handleConcurrentCreate(ctx context.Context, userID int64, number string) (UploadResult, error) {
	existing, err := s.repo.GetByNumber(ctx, number)
	if err != nil {
		return 0, err
	}

	if existing.UserID == userID {
		return UploadResultAlreadyUploadedByUser, nil
	}

	return 0, ErrOrderUploadedByAnotherUser
}
