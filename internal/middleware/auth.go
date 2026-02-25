package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/kayumovtd/gophermart/internal/auth"
)

// tokenParser валидирует токен и возвращает идентификатор пользователя
type tokenParser interface {
	Parse(token string, now time.Time) (int64, error)
}

func Auth(tokens tokenParser) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := extractBearerToken(r.Header.Get(auth.AuthorizationHeader))
			if !ok {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			userID, err := tokens.Parse(token, time.Now().UTC())
			if err != nil {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			ctx := auth.WithUserID(r.Context(), userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func extractBearerToken(headerValue string) (string, bool) {
	if !strings.HasPrefix(headerValue, auth.BearerPrefix) {
		return "", false
	}

	token := strings.TrimSpace(strings.TrimPrefix(headerValue, auth.BearerPrefix))
	if token == "" {
		return "", false
	}

	return token, true
}
