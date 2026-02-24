package server

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/kayumovtd/gophermart/internal/config"
	"github.com/kayumovtd/gophermart/internal/handlers"
	"github.com/kayumovtd/gophermart/internal/logger"
	"github.com/kayumovtd/gophermart/internal/middleware"
)

func New(cfg config.Config, log *logger.Logger) *http.Server {
	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.Recoverer)
	r.Use(middleware.RequestLogger(log))

	h := handlers.New(log)
	h.RegisterRoutes(r)

	return &http.Server{
		Addr:              cfg.RunAddress,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}
}
