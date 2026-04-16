package auth

//go:generate mockgen -source=../repository/user_repository.go -package=auth -destination=mock_user_repository_test.go -mock_names UserRepository=MockUserRepository
//go:generate mockgen -source=service.go -package=auth -destination=mock_token_generator_test.go -mock_names tokenGenerator=MockTokenGenerator

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/kayumovtd/gophermart/internal/repository"
	"go.uber.org/mock/gomock"
)

func TestServiceRegisterSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMockUserRepository(ctrl)
	tokens := NewMockTokenGenerator(ctrl)
	svc := NewService(repo, tokens)
	now := time.Unix(1700000000, 0).UTC()
	svc.now = func() time.Time { return now }

	repo.EXPECT().
		Create(gomock.Any(), "pupa", gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, passwordHash string) (repository.User, error) {
			if passwordHash == "password" {
				t.Fatalf("password hash not generated")
			}

			if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte("password")); err != nil {
				t.Fatalf("stored hash does not match password: %v", err)
			}

			return repository.User{
				ID:           1,
				Login:        "pupa",
				PasswordHash: passwordHash,
			}, nil
		})

	tokens.EXPECT().
		Generate(int64(1), now).
		Return("1", nil)

	token, err := svc.Register(context.Background(), "pupa", "password")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if token == "" {
		t.Fatalf("Register() token is empty")
	}
}

func TestServiceRegisterDuplicateLogin(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMockUserRepository(ctrl)
	tokens := NewMockTokenGenerator(ctrl)
	svc := NewService(repo, tokens)

	repo.EXPECT().
		Create(gomock.Any(), "pupa", gomock.Any()).
		Return(repository.User{}, repository.ErrUserAlreadyExists)

	_, err := svc.Register(context.Background(), "pupa", "password")
	if !errors.Is(err, ErrLoginTaken) {
		t.Fatalf("Register() error = %v, want %v", err, ErrLoginTaken)
	}
}

func TestServiceLoginInvalidCredentials(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMockUserRepository(ctrl)
	tokens := NewMockTokenGenerator(ctrl)
	svc := NewService(repo, tokens)

	hash, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword() error = %v", err)
	}

	repo.EXPECT().
		GetByLogin(gomock.Any(), "pupa").
		Return(repository.User{
			ID:           1,
			Login:        "pupa",
			PasswordHash: string(hash),
		}, nil)

	_, err = svc.Login(context.Background(), "pupa", "wrong")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v, want %v", err, ErrInvalidCredentials)
	}
}
