package orders

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kayumovtd/gophermart/internal/repository"
)

type stubOrderRepo struct {
	ordersByNumber map[string]repository.Order
	listByUserID   map[int64][]repository.Order
	createErr      error
}

func (r *stubOrderRepo) Create(_ context.Context, userID int64, number, status string) error {
	if r.createErr != nil {
		return r.createErr
	}

	if _, exists := r.ordersByNumber[number]; exists {
		return repository.ErrOrderAlreadyExists
	}

	r.ordersByNumber[number] = repository.Order{
		UserID:     userID,
		Number:     number,
		Status:     status,
		UploadedAt: time.Now().UTC(),
	}

	return nil
}

func (r *stubOrderRepo) GetByNumber(_ context.Context, number string) (repository.Order, error) {
	order, ok := r.ordersByNumber[number]
	if !ok {
		return repository.Order{}, repository.ErrOrderNotFound
	}
	return order, nil
}

func (r *stubOrderRepo) ListByUserID(_ context.Context, userID int64) ([]repository.Order, error) {
	return r.listByUserID[userID], nil
}

func (r *stubOrderRepo) ClaimForProcessing(_ context.Context, _ repository.OrderClaim) ([]repository.Order, error) {
	return nil, nil
}

func (r *stubOrderRepo) Complete(_ context.Context, _ int64, _ repository.OrderCompletion) error {
	return nil
}

func (r *stubOrderRepo) Requeue(_ context.Context, _ []int64, _ time.Time, _ time.Time, _ *string) error {
	return nil
}

func TestUploadAccepted(t *testing.T) {
	repo := &stubOrderRepo{
		ordersByNumber: make(map[string]repository.Order),
		listByUserID:   make(map[int64][]repository.Order),
	}
	svc := NewService(repo)

	result, err := svc.Upload(context.Background(), 1, "79927398713")
	if err != nil {
		t.Fatalf("Upload() error = %v", err)
	}

	if result != UploadResultAccepted {
		t.Fatalf("Upload() result = %v, want %v", result, UploadResultAccepted)
	}
}

func TestUploadInvalidOrderNumber(t *testing.T) {
	repo := &stubOrderRepo{
		ordersByNumber: make(map[string]repository.Order),
		listByUserID:   make(map[int64][]repository.Order),
	}
	svc := NewService(repo)

	_, err := svc.Upload(context.Background(), 1, "123")
	if !errors.Is(err, ErrInvalidOrderNumber) {
		t.Fatalf("Upload() error = %v, want %v", err, ErrInvalidOrderNumber)
	}
}

func TestUploadAlreadyUploadedByUser(t *testing.T) {
	repo := &stubOrderRepo{
		ordersByNumber: map[string]repository.Order{
			"79927398713": {UserID: 7, Number: "79927398713", Status: StatusNew},
		},
		listByUserID: make(map[int64][]repository.Order),
	}
	svc := NewService(repo)

	result, err := svc.Upload(context.Background(), 7, "79927398713")
	if err != nil {
		t.Fatalf("Upload() error = %v", err)
	}

	if result != UploadResultAlreadyUploadedByUser {
		t.Fatalf("Upload() result = %v, want %v", result, UploadResultAlreadyUploadedByUser)
	}
}

func TestUploadAlreadyUploadedByAnotherUser(t *testing.T) {
	repo := &stubOrderRepo{
		ordersByNumber: map[string]repository.Order{
			"79927398713": {UserID: 11, Number: "79927398713", Status: StatusNew},
		},
		listByUserID: make(map[int64][]repository.Order),
	}
	svc := NewService(repo)

	_, err := svc.Upload(context.Background(), 7, "79927398713")
	if !errors.Is(err, ErrOrderUploadedByAnotherUser) {
		t.Fatalf("Upload() error = %v, want %v", err, ErrOrderUploadedByAnotherUser)
	}
}

func TestUploadInvalidUserID(t *testing.T) {
	repo := &stubOrderRepo{
		ordersByNumber: make(map[string]repository.Order),
		listByUserID:   make(map[int64][]repository.Order),
	}
	svc := NewService(repo)

	_, err := svc.Upload(context.Background(), 0, "79927398713")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("Upload() error = %v, want %v", err, ErrInvalidInput)
	}
}

func TestListSuccess(t *testing.T) {
	ts := time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC)
	repo := &stubOrderRepo{
		ordersByNumber: make(map[string]repository.Order),
		listByUserID: map[int64][]repository.Order{
			1: {
				{
					Number:     "79927398713",
					Status:     StatusNew,
					UploadedAt: ts,
				},
			},
		},
	}
	svc := NewService(repo)

	got, err := svc.List(context.Background(), 1)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("List() len = %d, want %d", len(got), 1)
	}

	if got[0].Number != "79927398713" {
		t.Fatalf("List()[0].Number = %q, want %q", got[0].Number, "79927398713")
	}

	if got[0].UploadedAt != ts {
		t.Fatalf("List()[0].UploadedAt = %v, want %v", got[0].UploadedAt, ts)
	}
}
