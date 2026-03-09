package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	authHandler "github.com/kayumovtd/gophermart/internal/handlers/auth"
	balanceHandler "github.com/kayumovtd/gophermart/internal/handlers/balance"
	ordersHandler "github.com/kayumovtd/gophermart/internal/handlers/orders"
)

// Handler содержит HTTP‑хендлеры сервиса
type Handler struct {
	// authMiddleware проверка аутентификации
	authMiddleware func(http.Handler) http.Handler
	// registerHandler регистрация пользователя
	registerHandler http.HandlerFunc
	// loginHandler вход пользователя
	loginHandler http.HandlerFunc
	// uploadOrderHandler загрузка номера заказа
	uploadOrderHandler http.HandlerFunc
	// listOrdersHandler список заказов пользователя
	listOrdersHandler http.HandlerFunc
	// getBalanceHandler получение баланса пользователя
	getBalanceHandler http.HandlerFunc
	// withdrawBalanceHandler списание баллов
	withdrawBalanceHandler http.HandlerFunc
	// listWithdrawalsHandler список списаний
	listWithdrawalsHandler http.HandlerFunc
}

func New(
	authSvc authHandler.Service,
	orderSvc ordersHandler.Service,
	balanceSvc balanceHandler.Service,
	authMiddleware func(http.Handler) http.Handler,
) *Handler {
	return &Handler{
		authMiddleware:         authMiddleware,
		registerHandler:        authHandler.NewRegisterHandler(authSvc),
		loginHandler:           authHandler.NewLoginHandler(authSvc),
		uploadOrderHandler:     ordersHandler.NewUploadHandler(orderSvc),
		listOrdersHandler:      ordersHandler.NewListHandler(orderSvc),
		getBalanceHandler:      balanceHandler.NewGetHandler(balanceSvc),
		withdrawBalanceHandler: balanceHandler.NewWithdrawHandler(balanceSvc),
		listWithdrawalsHandler: balanceHandler.NewListWithdrawalsHandler(balanceSvc),
	}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/api/user", func(r chi.Router) {
		r.Post("/register", h.registerHandler)
		r.Post("/login", h.loginHandler)

		r.Group(func(r chi.Router) {
			r.Use(h.authMiddleware)
			r.Post("/orders", h.uploadOrderHandler)
			r.Get("/orders", h.listOrdersHandler)
			r.Get("/balance", h.getBalanceHandler)
			r.Post("/balance/withdraw", h.withdrawBalanceHandler)
			r.Get("/withdrawals", h.listWithdrawalsHandler)
		})
	})
}
