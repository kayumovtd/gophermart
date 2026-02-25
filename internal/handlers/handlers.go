package handlers

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// authService описывает операции аутентификации, используемые HTTP-слоем
type authService interface {
	Register(ctx context.Context, login, password string) (string, error)
	Login(ctx context.Context, login, password string) (string, error)
}

// Handler содержит HTTP‑хендлеры сервиса
type Handler struct {
	// authMiddleware проверка аутентификации
	authMiddleware func(http.Handler) http.Handler
	// registerHandler регистрация пользователя
	registerHandler http.HandlerFunc
	// loginHandler вход пользователя
	loginHandler http.HandlerFunc
}

func New(authSvc authService, authMiddleware func(http.Handler) http.Handler) *Handler {
	return &Handler{
		authMiddleware:  authMiddleware,
		registerHandler: newRegisterHandler(authSvc),
		loginHandler:    newLoginHandler(authSvc),
	}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/api/user", func(r chi.Router) {
		r.Post("/register", h.registerHandler)
		r.Post("/login", h.loginHandler)

		r.Group(func(r chi.Router) {
			r.Use(h.authMiddleware)
			r.Post("/orders", h.notImplemented)
			r.Get("/orders", h.notImplemented)
			r.Get("/balance", h.notImplemented)
			r.Post("/balance/withdraw", h.notImplemented)
			r.Get("/withdrawals", h.notImplemented)
		})
	})
}

func (h *Handler) notImplemented(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}
