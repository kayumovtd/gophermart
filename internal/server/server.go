package server

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/kayumovtd/gophermart/internal/auth"
	"github.com/kayumovtd/gophermart/internal/config"
	"github.com/kayumovtd/gophermart/internal/handlers"
	"github.com/kayumovtd/gophermart/internal/logger"
	"github.com/kayumovtd/gophermart/internal/middleware"
	"github.com/kayumovtd/gophermart/internal/orders"
)

type Dependencies struct {
	Logger        *logger.Logger
	AuthService   *auth.Service
	OrdersService *orders.Service
	TokenManager  *auth.TokenManager
}

func New(cfg config.Config, deps Dependencies) *http.Server {
	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.Recoverer)
	r.Use(middleware.RequestLogger(deps.Logger))

	h := handlers.New(deps.AuthService, deps.OrdersService, middleware.Auth(deps.TokenManager))
	h.RegisterRoutes(r)

	return &http.Server{
		Addr:              cfg.RunAddress,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}
}
