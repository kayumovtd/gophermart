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

	"github.com/kayumovtd/gophermart/internal/auth"
	"github.com/kayumovtd/gophermart/internal/config"
	"github.com/kayumovtd/gophermart/internal/logger"
	"github.com/kayumovtd/gophermart/internal/server"
	"github.com/kayumovtd/gophermart/internal/storage/postgres"
)

type App struct {
	server *http.Server
	log    *logger.Logger
	db     *pgxpool.Pool
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

	userRepo := postgres.NewUserRepository(pool)
	authService := auth.NewService(userRepo, tokenManager)

	app := &App{
		server: server.New(cfg, logg, authService, tokenManager),
		log:    logg,
		db:     pool,
	}

	ok = true
	return app, nil
}

func (a *App) Run() error {
	a.log.Info("starting server", zap.String("addr", a.server.Addr))

	// Запуск сервиса в отдельной горутине
	serverErr := make(chan error, 1)
	go func() {
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// Ожидаем сигнал завершения от ОС
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stop)

	select {
	case err := <-serverErr:
		return fmt.Errorf("server error: %w", err)
	case <-stop:
	}

	// Даём сервису 10 секунд на завершение
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Перестаём принимать новые запросы
	a.log.Info("shutting down")
	if err := a.server.Shutdown(ctx); err != nil {
		a.log.Error("shutdown error", zap.Error(err))
	}

	return nil
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
