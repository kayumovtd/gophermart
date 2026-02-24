package logger

import (
	"fmt"

	"go.uber.org/zap"
)

// Logger обёртка над zap.Logger
type Logger struct {
	// Logger внутренний zap-логгер
	*zap.Logger
}

func New(level string) (*Logger, error) {
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return nil, fmt.Errorf("parse log level %q: %w", level, err)
	}

	cfg := zap.NewProductionConfig()
	cfg.Level = lvl

	zl, err := cfg.Build()
	if err != nil {
		return nil, fmt.Errorf("build zap logger: %w", err)
	}

	return &Logger{zl}, nil
}

func NewNoOp() *Logger {
	return &Logger{zap.NewNop()}
}

func (l *Logger) Sync() error {
	return l.Logger.Sync()
}
