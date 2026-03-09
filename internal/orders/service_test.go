package orders

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kayumovtd/gophermart/internal/repository"
	"go.uber.org/mock/gomock"
)

func TestUploadAccepted(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMockOrderRepository(ctrl)
	svc := NewService(repo)

	repo.EXPECT().
		GetByNumber(gomock.Any(), "79927398713").
		Return(repository.Order{}, repository.ErrOrderNotFound)
	repo.EXPECT().
		Create(gomock.Any(), int64(1), "79927398713", StatusNew).
		Return(nil)

	result, err := svc.Upload(context.Background(), 1, "79927398713")
	if err != nil {
		t.Fatalf("Upload() error = %v", err)
	}

	if result != UploadResultAccepted {
		t.Fatalf("Upload() result = %v, want %v", result, UploadResultAccepted)
	}
}

func TestUploadInvalidOrderNumber(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc := NewService(NewMockOrderRepository(ctrl))

	_, err := svc.Upload(context.Background(), 1, "123")
	if !errors.Is(err, ErrInvalidOrderNumber) {
		t.Fatalf("Upload() error = %v, want %v", err, ErrInvalidOrderNumber)
	}
}

func TestUploadAlreadyUploadedByUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMockOrderRepository(ctrl)
	svc := NewService(repo)

	repo.EXPECT().
		GetByNumber(gomock.Any(), "79927398713").
		Return(repository.Order{UserID: 7, Number: "79927398713", Status: StatusNew}, nil)

	result, err := svc.Upload(context.Background(), 7, "79927398713")
	if err != nil {
		t.Fatalf("Upload() error = %v", err)
	}

	if result != UploadResultAlreadyUploadedByUser {
		t.Fatalf("Upload() result = %v, want %v", result, UploadResultAlreadyUploadedByUser)
	}
}

func TestUploadAlreadyUploadedByAnotherUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMockOrderRepository(ctrl)
	svc := NewService(repo)

	repo.EXPECT().
		GetByNumber(gomock.Any(), "79927398713").
		Return(repository.Order{UserID: 11, Number: "79927398713", Status: StatusNew}, nil)

	_, err := svc.Upload(context.Background(), 7, "79927398713")
	if !errors.Is(err, ErrOrderUploadedByAnotherUser) {
		t.Fatalf("Upload() error = %v, want %v", err, ErrOrderUploadedByAnotherUser)
	}
}

func TestUploadInvalidUserID(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc := NewService(NewMockOrderRepository(ctrl))

	_, err := svc.Upload(context.Background(), 0, "79927398713")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("Upload() error = %v, want %v", err, ErrInvalidInput)
	}
}

func TestListSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMockOrderRepository(ctrl)
	svc := NewService(repo)
	ts := time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC)

	repo.EXPECT().
		ListByUserID(gomock.Any(), int64(1)).
		Return([]repository.Order{
			{
				Number:     "79927398713",
				Status:     StatusNew,
				UploadedAt: ts,
			},
		}, nil)

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
