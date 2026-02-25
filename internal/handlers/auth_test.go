package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/kayumovtd/gophermart/internal/auth"
	"github.com/kayumovtd/gophermart/internal/middleware"
)

type fakeAuthService struct {
	registerToken string
	registerErr   error
	loginToken    string
	loginErr      error
}

func (s fakeAuthService) Register(_ context.Context, _, _ string) (string, error) {
	return s.registerToken, s.registerErr
}

func (s fakeAuthService) Login(_ context.Context, _, _ string) (string, error) {
	return s.loginToken, s.loginErr
}

func TestRegisterSuccess(t *testing.T) {
	h := newTestRouter(fakeAuthService{registerToken: "token"}, func(next http.Handler) http.Handler { return next })

	req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewBufferString(`{"login":"pupa","password":"lupa"}`))
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	res := rr.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusOK)
	}

	authHeader := res.Header.Get(auth.AuthorizationHeader)
	expected := auth.BearerPrefix + "token"
	if authHeader != expected {
		t.Fatalf("Authorization header = %q, want %q", authHeader, expected)
	}
}

func TestRegisterConflict(t *testing.T) {
	h := newTestRouter(fakeAuthService{registerErr: auth.ErrLoginTaken}, func(next http.Handler) http.Handler { return next })

	req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewBufferString(`{"login":"pupa","password":"lupa"}`))
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusConflict)
	}
}

func TestLoginUnauthorized(t *testing.T) {
	h := newTestRouter(fakeAuthService{loginErr: auth.ErrInvalidCredentials}, func(next http.Handler) http.Handler { return next })

	req := httptest.NewRequest(http.MethodPost, "/api/user/login", bytes.NewBufferString(`{"login":"pupa","password":"lupa"}`))
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestRegisterBadRequest(t *testing.T) {
	h := newTestRouter(fakeAuthService{}, func(next http.Handler) http.Handler { return next })

	req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewBufferString(`{`))
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestProtectedRouteRequiresAuth(t *testing.T) {
	tokenManager, err := auth.NewTokenManager([]byte("test-secret"))
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}

	h := newTestRouter(fakeAuthService{}, middleware.Auth(tokenManager))

	req := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestProtectedRouteWithAuth(t *testing.T) {
	tokenManager, err := auth.NewTokenManager([]byte("test-secret"))
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}

	token, err := tokenManager.Generate(1, time.Now().UTC())
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	h := newTestRouter(fakeAuthService{}, middleware.Auth(tokenManager))

	req := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)
	req.Header.Set(auth.AuthorizationHeader, auth.BearerPrefix+token)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotImplemented)
	}
}

func newTestRouter(authSvc fakeAuthService, authMiddleware func(http.Handler) http.Handler) http.Handler {
	r := chi.NewRouter()
	h := New(authSvc, authMiddleware)
	h.RegisterRoutes(r)
	return r
}
