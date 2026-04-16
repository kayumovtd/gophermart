package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/kayumovtd/gophermart/internal/accrual"
	"github.com/kayumovtd/gophermart/internal/auth"
	"github.com/kayumovtd/gophermart/internal/balance"
	"github.com/kayumovtd/gophermart/internal/config"
	"github.com/kayumovtd/gophermart/internal/logger"
	"github.com/kayumovtd/gophermart/internal/orders"
	"github.com/kayumovtd/gophermart/internal/server"
	"github.com/kayumovtd/gophermart/internal/storage/postgres"
)

type App struct {
	server    *http.Server
	log       *logger.Logger
	db        *pgxpool.Pool
	processor *orders.Processor
}

func New(ctx context.Context, cfg config.Config) (*App, error) {
	logg, err := logger.New(cfg.LogLevel)
	if err != nil {
		return nil, fmt.Errorf("logger: %w", err)
	}

	var pool *pgxpool.Pool
	ok := false
	defer func() {
		if ok {
			return
		}
		if pool != nil {
			pool.Close()
		}
		_ = logg.Sync()
	}()

	pool, err = postgres.Open(ctx, cfg.DatabaseURI)
	if err != nil {
		return nil, fmt.Errorf("db init: %w", err)
	}

	tokenManager, err := auth.NewTokenManager([]byte(cfg.AuthSecret))
	if err != nil {
		return nil, fmt.Errorf("token manager: %w", err)
	}

	userStorage := postgres.NewUserStorage(pool)
	authService := auth.NewService(userStorage, tokenManager)
	orderStorage := postgres.NewOrderStorage(pool)
	ordersService := orders.NewService(orderStorage)
	balanceStorage := postgres.NewBalanceStorage(pool)
	balanceService := balance.NewService(balanceStorage)
	accrualClient := accrual.NewClient(cfg.AccrualAddress, &http.Client{Timeout: 5 * time.Second})
	processor := orders.NewProcessor(orderStorage, accrualClient, logg, 4)

	app := &App{
		server: server.New(cfg, server.Dependencies{
			Logger:         logg,
			AuthService:    authService,
			OrdersService:  ordersService,
			BalanceService: balanceService,
			TokenManager:   tokenManager,
		}),
		log:       logg,
		db:        pool,
		processor: processor,
	}

	ok = true
	return app, nil
}

func (a *App) Run() error {
	a.log.Info("starting server", zap.String("addr", a.server.Addr))

	processorCtx, cancelProcessor := context.WithCancel(context.Background())
	defer cancelProcessor()

	processorDone := make(chan struct{})
	go func() {
		defer close(processorDone)
		a.processor.Run(processorCtx)
	}()

	serverErr := make(chan error, 1)
	go func() {
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stop)

	var runErr error

	select {
	case err := <-serverErr:
		runErr = fmt.Errorf("server error: %w", err)
	case <-stop:
	}

	a.log.Info("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := a.server.Shutdown(ctx); err != nil {
		a.log.Error("shutdown error", zap.Error(err))
	}

	cancelProcessor()

	select {
	case <-processorDone:
	case <-ctx.Done():
		a.log.Warn("order processor shutdown timed out")
	}

	return runErr
}

func (a *App) Close() error {
	if a.db != nil {
		a.db.Close()
	}
	if a.log != nil {
		return a.log.Sync()
	}
	return nil
}
