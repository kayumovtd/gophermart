package repository

import (
	"context"
	"errors"
	"time"
)

var (
	ErrOrderNotFound      = errors.New("order not found")
	ErrOrderAlreadyExists = errors.New("order already exists")
)

type Order struct {
	ID         int64
	UserID     int64
	Number     string
	Status     string
	Accrual    *float64
	UploadedAt time.Time
}

type OrderClaim struct {
	WorkerID   string
	Now        time.Time
	Limit      int
	LeaseUntil time.Time
}

type OrderCompletion struct {
	Status  string
	Accrual *float64
}

type OrderRepository interface {
	// Create создает заказ пользователя и задачу на обработку
	Create(ctx context.Context, userID int64, number, status string) error

	// GetByNumber возвращает заказ по номеру
	GetByNumber(ctx context.Context, number string) (Order, error)

	// ListByUserID возвращает заказы пользователя по убыванию времени добавления
	ListByUserID(ctx context.Context, userID int64) ([]Order, error)

	// ClaimForProcessing блокирует пачку заказов для обработки воркером
	ClaimForProcessing(ctx context.Context, claim OrderClaim) ([]Order, error)

	// Complete завершает обработку заказа и удаляет задачу из очереди
	Complete(ctx context.Context, orderID int64, completion OrderCompletion) error

	// Requeue возвращает заказы в очередь с новыми параметрами повтора
	Requeue(ctx context.Context, orderIDs []int64, nextAttemptAt, updatedAt time.Time, lastError *string) error
}
