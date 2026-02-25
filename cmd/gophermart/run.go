package main

import (
	"context"
	"fmt"
	"time"

	"github.com/kayumovtd/gophermart/internal/app"
	"github.com/kayumovtd/gophermart/internal/config"
)

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	initCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	application, err := app.New(initCtx, cfg)
	if err != nil {
		return err
	}
	defer func() {
		_ = application.Close()
	}()

	return application.Run()
}
