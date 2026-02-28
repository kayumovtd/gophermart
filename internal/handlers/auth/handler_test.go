package auth

//go:generate mockgen -source=service.go -package=auth -destination=mock_service_test.go -mock_names Service=MockService

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	coreauth "github.com/kayumovtd/gophermart/internal/auth"
	"go.uber.org/mock/gomock"
)

func TestRegisterSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	authSvc := NewMockService(ctrl)

	authSvc.EXPECT().
		Register(gomock.Any(), "pupa", "lupa").
		Return("token", nil)

	handler := NewRegisterHandler(authSvc)

	req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewBufferString(`{"login":"pupa","password":"lupa"}`))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	res := rr.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusOK)
	}

	authHeader := res.Header.Get(coreauth.AuthorizationHeader)
	expected := coreauth.BearerPrefix + "token"
	if authHeader != expected {
		t.Fatalf("Authorization header = %q, want %q", authHeader, expected)
	}
}

func TestRegisterConflict(t *testing.T) {
	ctrl := gomock.NewController(t)
	authSvc := NewMockService(ctrl)

	authSvc.EXPECT().
		Register(gomock.Any(), "pupa", "lupa").
		Return("", coreauth.ErrLoginTaken)

	handler := NewRegisterHandler(authSvc)

	req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewBufferString(`{"login":"pupa","password":"lupa"}`))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusConflict)
	}
}

func TestLoginUnauthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	authSvc := NewMockService(ctrl)

	authSvc.EXPECT().
		Login(gomock.Any(), "pupa", "lupa").
		Return("", coreauth.ErrInvalidCredentials)

	handler := NewLoginHandler(authSvc)

	req := httptest.NewRequest(http.MethodPost, "/api/user/login", bytes.NewBufferString(`{"login":"pupa","password":"lupa"}`))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestRegisterBadRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	authSvc := NewMockService(ctrl)

	handler := NewRegisterHandler(authSvc)

	req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewBufferString(`{`))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}
