package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/kayumovtd/gophermart/internal/auth"
)

type authRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"` //nolint:gosec
}

func newRegisterHandler(authSvc authService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, ok := decodeAuthRequest(w, r)
		if !ok {
			return
		}

		token, err := authSvc.Register(r.Context(), req.Login, req.Password)
		if err != nil {
			switch {
			case errors.Is(err, auth.ErrInvalidInput):
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			case errors.Is(err, auth.ErrLoginTaken):
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

func newLoginHandler(authSvc authService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, ok := decodeAuthRequest(w, r)
		if !ok {
			return
		}

		token, err := authSvc.Login(r.Context(), req.Login, req.Password)
		if err != nil {
			switch {
			case errors.Is(err, auth.ErrInvalidInput):
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			case errors.Is(err, auth.ErrInvalidCredentials):
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

func decodeAuthRequest(w http.ResponseWriter, r *http.Request) (authRequest, bool) {
	defer func() {
		_ = r.Body.Close()
	}()

	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return authRequest{}, false
	}

	return req, true
}

func setAuthorization(w http.ResponseWriter, token string) {
	w.Header().Set(auth.AuthorizationHeader, auth.BearerPrefix+token)
}
