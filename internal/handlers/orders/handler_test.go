package orders

//go:generate mockgen -source=service.go -package=orders -destination=mock_service_test.go -mock_names Service=MockService

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	coreauth "github.com/kayumovtd/gophermart/internal/auth"
	coreorders "github.com/kayumovtd/gophermart/internal/orders"
	"go.uber.org/mock/gomock"
)

func TestUploadOrderAccepted(t *testing.T) {
	ctrl := gomock.NewController(t)
	orderSvc := NewMockService(ctrl)

	orderSvc.EXPECT().
		Upload(gomock.Any(), int64(42), "79927398713").
		Return(coreorders.UploadResultAccepted, nil)

	handler := NewUploadHandler(orderSvc)

	req := httptest.NewRequest(http.MethodPost, "/api/user/orders", strings.NewReader("79927398713"))
	req = req.WithContext(coreauth.WithUserID(req.Context(), 42))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusAccepted)
	}
}

func TestUploadOrderUnauthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	orderSvc := NewMockService(ctrl)

	handler := NewUploadHandler(orderSvc)

	req := httptest.NewRequest(http.MethodPost, "/api/user/orders", strings.NewReader("79927398713"))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestUploadOrderConflict(t *testing.T) {
	ctrl := gomock.NewController(t)
	orderSvc := NewMockService(ctrl)

	orderSvc.EXPECT().
		Upload(gomock.Any(), int64(1), "79927398713").
		Return(coreorders.UploadResultAccepted, coreorders.ErrOrderUploadedByAnotherUser)

	handler := NewUploadHandler(orderSvc)

	req := httptest.NewRequest(http.MethodPost, "/api/user/orders", strings.NewReader("79927398713"))
	req = req.WithContext(coreauth.WithUserID(req.Context(), 1))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusConflict)
	}
}

func TestUploadOrderInvalidNumber(t *testing.T) {
	ctrl := gomock.NewController(t)
	orderSvc := NewMockService(ctrl)

	orderSvc.EXPECT().
		Upload(gomock.Any(), int64(1), "bad-order").
		Return(coreorders.UploadResultAccepted, coreorders.ErrInvalidOrderNumber)

	handler := NewUploadHandler(orderSvc)

	req := httptest.NewRequest(http.MethodPost, "/api/user/orders", strings.NewReader("bad-order"))
	req = req.WithContext(coreauth.WithUserID(req.Context(), 1))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnprocessableEntity)
	}
}

func TestListOrdersNoContent(t *testing.T) {
	ctrl := gomock.NewController(t)
	orderSvc := NewMockService(ctrl)

	orderSvc.EXPECT().
		List(gomock.Any(), int64(7)).
		Return(nil, nil)

	handler := NewListHandler(orderSvc)

	req := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)
	req = req.WithContext(coreauth.WithUserID(req.Context(), 7))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNoContent)
	}
}

func TestListOrdersUnauthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	orderSvc := NewMockService(ctrl)

	handler := NewListHandler(orderSvc)

	req := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestListOrdersSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	orderSvc := NewMockService(ctrl)

	orderSvc.EXPECT().
		List(gomock.Any(), int64(1)).
		Return([]coreorders.Order{
			{
				Number:     "79927398713",
				Status:     coreorders.StatusNew,
				UploadedAt: time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC),
			},
		}, nil)

	handler := NewListHandler(orderSvc)

	req := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)
	req = req.WithContext(coreauth.WithUserID(req.Context(), 1))
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

	if got[0]["number"] != "79927398713" {
		t.Fatalf("number = %v, want %q", got[0]["number"], "79927398713")
	}

	if got[0]["status"] != coreorders.StatusNew {
		t.Fatalf("status = %v, want %q", got[0]["status"], coreorders.StatusNew)
	}

	if got[0]["uploaded_at"] != "2026-02-25T10:00:00Z" {
		t.Fatalf("uploaded_at = %v, want %q", got[0]["uploaded_at"], "2026-02-25T10:00:00Z")
	}
}
