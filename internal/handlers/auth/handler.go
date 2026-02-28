package auth

import (
	"encoding/json"
	"errors"
	"net/http"

	coreauth "github.com/kayumovtd/gophermart/internal/auth"
)

type request struct {
	Login    string `json:"login"`
	Password string `json:"password"` //nolint:gosec
}

func NewRegisterHandler(authSvc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, ok := decodeRequest(w, r)
		if !ok {
			return
		}

		token, err := authSvc.Register(r.Context(), req.Login, req.Password)
		if err != nil {
			switch {
			case errors.Is(err, coreauth.ErrInvalidInput):
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			case errors.Is(err, coreauth.ErrLoginTaken):
				http.Error(w, "login already taken", http.StatusConflict)
			default:
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}

		setAuthorization(w, token)
		w.WriteHeader(http.StatusOK)
	}
}

func NewLoginHandler(authSvc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, ok := decodeRequest(w, r)
		if !ok {
			return
		}

		token, err := authSvc.Login(r.Context(), req.Login, req.Password)
		if err != nil {
			switch {
			case errors.Is(err, coreauth.ErrInvalidInput):
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			case errors.Is(err, coreauth.ErrInvalidCredentials):
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			default:
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}

		setAuthorization(w, token)
		w.WriteHeader(http.StatusOK)
	}
}

func decodeRequest(w http.ResponseWriter, r *http.Request) (request, bool) {
	defer func() {
		_ = r.Body.Close()
	}()

	var req request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return request{}, false
	}

	return req, true
}

func setAuthorization(w http.ResponseWriter, token string) {
	w.Header().Set(coreauth.AuthorizationHeader, coreauth.BearerPrefix+token)
}
