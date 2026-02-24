package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/kayumovtd/gophermart/internal/logger"
)

// Handler содержит HTTP‑хендлеры сервиса
type Handler struct {
	// log логгер приложения
	log *logger.Logger
}

func New(log *logger.Logger) *Handler {
	return &Handler{log: log}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/api/user", func(r chi.Router) {
		r.Post("/register", h.notImplemented)
		r.Post("/login", h.notImplemented)
		r.Post("/orders", h.notImplemented)
		r.Get("/orders", h.notImplemented)
		r.Get("/balance", h.notImplemented)
		r.Post("/balance/withdraw", h.notImplemented)
		r.Get("/withdrawals", h.notImplemented)
	})
}

func (h *Handler) notImplemented(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}
