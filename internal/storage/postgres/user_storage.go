package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kayumovtd/gophermart/internal/repository"
)

type UserStorage struct {
	pool *pgxpool.Pool
}

func NewUserStorage(pool *pgxpool.Pool) *UserStorage {
	return &UserStorage{pool: pool}
}

func (r *UserStorage) Create(ctx context.Context, login, passwordHash string) (repository.User, error) {
	const query = `
INSERT INTO users (login, password_hash)
VALUES ($1, $2)
RETURNING id, login, password_hash;
	`

	var user repository.User
	err := r.pool.QueryRow(ctx, query, login, passwordHash).Scan(&user.ID, &user.Login, &user.PasswordHash)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return repository.User{}, repository.ErrUserAlreadyExists
		}
		return repository.User{}, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}

func (r *UserStorage) GetByLogin(ctx context.Context, login string) (repository.User, error) {
	const query = `
SELECT id, login, password_hash
FROM users
WHERE login = $1;
	`

	var user repository.User
	err := r.pool.QueryRow(ctx, query, login).Scan(&user.ID, &user.Login, &user.PasswordHash)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return repository.User{}, repository.ErrUserNotFound
		}
		return repository.User{}, fmt.Errorf("get user by login: %w", err)
	}

	return user, nil
}
