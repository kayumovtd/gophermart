package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/kayumovtd/gophermart/internal/config"
	"github.com/kayumovtd/gophermart/internal/logger"
	"github.com/kayumovtd/gophermart/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	logg, err := logger.New(cfg.LogLevel)
	if err != nil {
		log.Fatalf("logger: %v", err)
	}
	defer func() {
		if err := logg.Sync(); err != nil {
			log.Printf("logger sync failed: %v", err)
		}
	}()

	srv := server.New(cfg, logg)

	logg.Info("starting server", zap.String("addr", cfg.RunAddress))

	// Запуск сервиса в отдельной горутине
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logg.Fatal("server error", zap.Error(err))
		}
	}()

	// Ожидаем сигнал завершения от ОС
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	// Даём сервису 10 секунд на завершение
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Перестаём принимать новые запросы
	logg.Info("shutting down")
	if err := srv.Shutdown(ctx); err != nil {
		logg.Error("shutdown error", zap.Error(err))
	}
}
