package balance

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	coreauth "github.com/kayumovtd/gophermart/internal/auth"
	corebalance "github.com/kayumovtd/gophermart/internal/balance"
)

type getBalanceResponse struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

func NewGetHandler(balanceSvc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := coreauth.UserID(r.Context())
		if !ok || userID <= 0 {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		balance, err := balanceSvc.Get(r.Context(), userID)
		if err != nil {
			switch {
			case errors.Is(err, corebalance.ErrInvalidInput):
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			default:
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(getBalanceResponse{
			Current:   balance.Current,
			Withdrawn: balance.Withdrawn,
		}); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}
}

type withdrawRequest struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

func NewWithdrawHandler(balanceSvc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			_ = r.Body.Close()
		}()

		userID, ok := coreauth.UserID(r.Context())
		if !ok || userID <= 0 {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		var req withdrawRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		if err := balanceSvc.Withdraw(r.Context(), userID, req.Order, req.Sum); err != nil {
			switch {
			case errors.Is(err, corebalance.ErrInvalidOrderNumber):
				http.Error(w, http.StatusText(http.StatusUnprocessableEntity), http.StatusUnprocessableEntity)
			case errors.Is(err, corebalance.ErrInsufficientFunds):
				http.Error(w, http.StatusText(http.StatusPaymentRequired), http.StatusPaymentRequired)
			case errors.Is(err, corebalance.ErrInvalidInput):
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			default:
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

type withdrawalsResponse struct {
	Order       string  `json:"order"`
	Sum         float64 `json:"sum"`
	ProcessedAt string  `json:"processed_at"`
}

func NewListWithdrawalsHandler(balanceSvc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := coreauth.UserID(r.Context())
		if !ok || userID <= 0 {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		items, err := balanceSvc.ListWithdrawals(r.Context(), userID)
		if err != nil {
			switch {
			case errors.Is(err, corebalance.ErrInvalidInput):
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

		resp := make([]withdrawalsResponse, 0, len(items))
		for _, item := range items {
			resp = append(resp, withdrawalsResponse{
				Order:       item.Order,
				Sum:         item.Sum,
				ProcessedAt: item.ProcessedAt.Format(time.RFC3339),
			})
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}
}
