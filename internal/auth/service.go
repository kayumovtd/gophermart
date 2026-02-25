package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/kayumovtd/gophermart/internal/repository"
)

var (
	ErrInvalidInput       = errors.New("invalid input")
	ErrLoginTaken         = errors.New("login already taken")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// tokenGenerator генерирует токен по идентификатору пользователя
type tokenGenerator interface {
	Generate(userID int64, now time.Time) (string, error)
}

type Service struct {
	repo   repository.UserRepository
	tokens tokenGenerator
	now    func() time.Time
}

func NewService(repo repository.UserRepository, tokens tokenGenerator) *Service {
	return &Service{
		repo:   repo,
		tokens: tokens,
		now:    time.Now,
	}
}

func (s *Service) Register(ctx context.Context, login, password string) (string, error) {
	login = strings.TrimSpace(login)
	if login == "" || password == "" {
		return "", ErrInvalidInput
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	user, err := s.repo.Create(ctx, login, string(passwordHash))
	if err != nil {
		if errors.Is(err, repository.ErrUserAlreadyExists) {
			return "", ErrLoginTaken
		}
		return "", err
	}

	token, err := s.tokens.Generate(user.ID, s.now().UTC())
	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *Service) Login(ctx context.Context, login, password string) (string, error) {
	login = strings.TrimSpace(login)
	if login == "" || password == "" {
		return "", ErrInvalidInput
	}

	user, err := s.repo.GetByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return "", ErrInvalidCredentials
		}
		return "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}

	token, err := s.tokens.Generate(user.ID, s.now().UTC())
	if err != nil {
		return "", err
	}

	return token, nil
}
