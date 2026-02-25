package auth

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/kayumovtd/gophermart/internal/repository"
)

type stubUserRepo struct {
	users map[string]repository.User
	next  int64
}

func (r *stubUserRepo) Create(_ context.Context, login, passwordHash string) (repository.User, error) {
	if _, ok := r.users[login]; ok {
		return repository.User{}, repository.ErrUserAlreadyExists
	}

	r.next++
	user := repository.User{
		ID:           r.next,
		Login:        login,
		PasswordHash: passwordHash,
	}
	r.users[login] = user
	return user, nil
}

func (r *stubUserRepo) GetByLogin(_ context.Context, login string) (repository.User, error) {
	user, ok := r.users[login]
	if !ok {
		return repository.User{}, repository.ErrUserNotFound
	}
	return user, nil
}

type stubTokenGenerator struct{}

func (stubTokenGenerator) Generate(userID int64, _ time.Time) (string, error) {
	return strconv.Itoa(int(userID)), nil
}

func TestServiceRegisterSuccess(t *testing.T) {
	repo := &stubUserRepo{users: make(map[string]repository.User)}
	svc := NewService(repo, stubTokenGenerator{})
	svc.now = func() time.Time { return time.Unix(1700000000, 0).UTC() }

	token, err := svc.Register(context.Background(), "pupa", "password")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if token == "" {
		t.Fatalf("Register() token is empty")
	}

	user, ok := repo.users["pupa"]
	if !ok {
		t.Fatalf("user not stored")
	}

	if user.PasswordHash == "password" {
		t.Fatalf("password hash not generated")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("password")); err != nil {
		t.Fatalf("stored hash does not match password: %v", err)
	}
}

func TestServiceRegisterDuplicateLogin(t *testing.T) {
	repo := &stubUserRepo{
		users: map[string]repository.User{
			"pupa": {ID: 1, Login: "pupa", PasswordHash: "hash"},
		},
	}
	svc := NewService(repo, stubTokenGenerator{})

	_, err := svc.Register(context.Background(), "pupa", "password")
	if !errors.Is(err, ErrLoginTaken) {
		t.Fatalf("Register() error = %v, want %v", err, ErrLoginTaken)
	}
}

func TestServiceLoginInvalidCredentials(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword() error = %v", err)
	}

	repo := &stubUserRepo{
		users: map[string]repository.User{
			"pupa": {ID: 1, Login: "pupa", PasswordHash: string(hash)},
		},
	}
	svc := NewService(repo, stubTokenGenerator{})

	_, err = svc.Login(context.Background(), "pupa", "wrong")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v, want %v", err, ErrInvalidCredentials)
	}
}
