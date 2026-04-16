package orders

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	coreauth "github.com/kayumovtd/gophermart/internal/auth"
	coreorders "github.com/kayumovtd/gophermart/internal/orders"
)

func NewUploadHandler(orderSvc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			_ = r.Body.Close()
		}()

		userID, ok := coreauth.UserID(r.Context())
		if !ok || userID <= 0 {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		body, err := io.ReadAll(io.LimitReader(r.Body, 1024))
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		number := strings.TrimSpace(string(body))
		result, err := orderSvc.Upload(r.Context(), userID, number)
		if err != nil {
			switch {
			case errors.Is(err, coreorders.ErrInvalidOrderNumber):
				http.Error(w, http.StatusText(http.StatusUnprocessableEntity), http.StatusUnprocessableEntity)
			case errors.Is(err, coreorders.ErrOrderUploadedByAnotherUser):
				http.Error(w, http.StatusText(http.StatusConflict), http.StatusConflict)
			case errors.Is(err, coreorders.ErrInvalidInput):
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			default:
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}

		switch result {
		case coreorders.UploadResultAlreadyUploadedByUser:
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusAccepted)
		}
	}
}

type response struct {
	Number     string   `json:"number"`
	Status     string   `json:"status"`
	Accrual    *float64 `json:"accrual,omitempty"`
	UploadedAt string   `json:"uploaded_at"`
}

func NewListHandler(orderSvc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := coreauth.UserID(r.Context())
		if !ok || userID <= 0 {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		items, err := orderSvc.List(r.Context(), userID)
		if err != nil {
			switch {
			case errors.Is(err, coreorders.ErrInvalidInput):
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			default:
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}

		if len(items) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		resp := make([]response, 0, len(items))
		for _, item := range items {
			resp = append(resp, response{
				Number:     item.Number,
				Status:     item.Status,
				Accrual:    item.Accrual,
				UploadedAt: item.UploadedAt.Format(time.RFC3339),
			})
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}
}
