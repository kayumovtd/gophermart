package repository

import (
	"context"
	"errors"
)

var (
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrUserNotFound      = errors.New("user not found")
)

type User struct {
	ID           int64
	Login        string
	PasswordHash string
}

// UserRepository описывает операции доступа к данным пользователей
type UserRepository interface {
	Create(ctx context.Context, login, passwordHash string) (User, error)
	GetByLogin(ctx context.Context, login string) (User, error)
}
