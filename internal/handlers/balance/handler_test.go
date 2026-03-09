package balance

//go:generate mockgen -source=service.go -package=balance -destination=mock_service_test.go -mock_names Service=MockService

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	coreauth "github.com/kayumovtd/gophermart/internal/auth"
	corebalance "github.com/kayumovtd/gophermart/internal/balance"
	"go.uber.org/mock/gomock"
)

func TestGetBalanceSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	balanceSvc := NewMockService(ctrl)

	balanceSvc.EXPECT().
		Get(gomock.Any(), int64(7)).
		Return(corebalance.Balance{Current: 100.5, Withdrawn: 20}, nil)

	handler := NewGetHandler(balanceSvc)
	req := httptest.NewRequest(http.MethodGet, "/api/user/balance", nil)
	req = req.WithContext(coreauth.WithUserID(req.Context(), 7))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var got map[string]float64
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if got["current"] != 100.5 {
		t.Fatalf("current = %v, want 100.5", got["current"])
	}
	if got["withdrawn"] != 20 {
		t.Fatalf("withdrawn = %v, want 20", got["withdrawn"])
	}
}

func TestWithdrawInsufficientFunds(t *testing.T) {
	ctrl := gomock.NewController(t)
	balanceSvc := NewMockService(ctrl)

	balanceSvc.EXPECT().
		Withdraw(gomock.Any(), int64(1), "79927398713", 100.0).
		Return(corebalance.ErrInsufficientFunds)

	handler := NewWithdrawHandler(balanceSvc)
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/user/balance/withdraw",
		strings.NewReader(`{"order":"79927398713","sum":100}`),
	)
	req = req.WithContext(coreauth.WithUserID(req.Context(), 1))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusPaymentRequired {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusPaymentRequired)
	}
}

func TestWithdrawInvalidOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	balanceSvc := NewMockService(ctrl)

	balanceSvc.EXPECT().
		Withdraw(gomock.Any(), int64(1), "123", 100.0).
		Return(corebalance.ErrInvalidOrderNumber)

	handler := NewWithdrawHandler(balanceSvc)
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/user/balance/withdraw",
		strings.NewReader(`{"order":"123","sum":100}`),
	)
	req = req.WithContext(coreauth.WithUserID(req.Context(), 1))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnprocessableEntity)
	}
}

func TestListWithdrawalsNoContent(t *testing.T) {
	ctrl := gomock.NewController(t)
	balanceSvc := NewMockService(ctrl)

	balanceSvc.EXPECT().
		ListWithdrawals(gomock.Any(), int64(2)).
		Return(nil, nil)

	handler := NewListWithdrawalsHandler(balanceSvc)
	req := httptest.NewRequest(http.MethodGet, "/api/user/withdrawals", nil)
	req = req.WithContext(coreauth.WithUserID(req.Context(), 2))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNoContent)
	}
}

func TestListWithdrawalsSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	balanceSvc := NewMockService(ctrl)
	ts := time.Date(2026, 3, 1, 12, 30, 0, 0, time.UTC)

	balanceSvc.EXPECT().
		ListWithdrawals(gomock.Any(), int64(2)).
		Return([]corebalance.Withdrawal{
			{
				Order:       "79927398713",
				Sum:         12.5,
				ProcessedAt: ts,
			},
		}, nil)

	handler := NewListWithdrawalsHandler(balanceSvc)
	req := httptest.NewRequest(http.MethodGet, "/api/user/withdrawals", nil)
	req = req.WithContext(coreauth.WithUserID(req.Context(), 2))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var got []map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("len(body) = %d, want %d", len(got), 1)
	}

	if got[0]["order"] != "79927398713" {
		t.Fatalf("order = %v, want %q", got[0]["order"], "79927398713")
	}

	if got[0]["sum"] != 12.5 {
		t.Fatalf("sum = %v, want 12.5", got[0]["sum"])
	}

	if got[0]["processed_at"] != "2026-03-01T12:30:00Z" {
		t.Fatalf("processed_at = %v, want %q", got[0]["processed_at"], "2026-03-01T12:30:00Z")
	}
}

func TestGetBalanceUnauthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	handler := NewGetHandler(NewMockService(ctrl))

	req := httptest.NewRequest(http.MethodGet, "/api/user/balance", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}
