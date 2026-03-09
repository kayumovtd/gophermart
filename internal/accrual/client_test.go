package accrual

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientGetOrderProcessed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/orders/123" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/api/orders/123")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"order":"123","status":"PROCESSED","accrual":42.5}`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, srv.Client())

	order, err := client.GetOrder(context.Background(), "123")
	if err != nil {
		t.Fatalf("GetOrder() error = %v", err)
	}

	if order.Status != StatusProcessed {
		t.Fatalf("status = %q, want %q", order.Status, StatusProcessed)
	}

	if order.Accrual == nil || *order.Accrual != 42.5 {
		t.Fatalf("accrual = %v, want %v", order.Accrual, 42.5)
	}
}

func TestClientGetOrderNotRegistered(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, srv.Client())

	_, err := client.GetOrder(context.Background(), "123")
	if err != ErrOrderNotRegistered {
		t.Fatalf("GetOrder() error = %v, want %v", err, ErrOrderNotRegistered)
	}
}

func TestClientGetOrderRateLimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, srv.Client())

	_, err := client.GetOrder(context.Background(), "123")
	if err == nil {
		t.Fatalf("GetOrder() error = nil")
	}

	var rateLimitErr *RateLimitError
	if ok := errors.As(err, &rateLimitErr); !ok {
		t.Fatalf("GetOrder() error = %v, want RateLimitError", err)
	}

	if rateLimitErr.RetryAfter != 60*time.Second {
		t.Fatalf("RetryAfter = %s, want %s", rateLimitErr.RetryAfter, 60*time.Second)
	}
}

func TestClientGetOrderInvalidBaseURL(t *testing.T) {
	client := NewClient("://bad-url", nil)

	_, err := client.GetOrder(context.Background(), "123")
	if err == nil {
		t.Fatalf("GetOrder() error = nil")
	}
	if err.Error() != "accrual base URL is invalid" {
		t.Fatalf("GetOrder() error = %v, want %q", err, "accrual base URL is invalid")
	}
}
