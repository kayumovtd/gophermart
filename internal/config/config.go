package config

import (
	"io"
	"os"

	"github.com/spf13/pflag"
)

const (
	defaultRunAddress     = "localhost:8080"
	defaultDatabaseURI    = "postgres://localhost:5432/gophermart?sslmode=disable"
	defaultAccrualAddress = "http://localhost:8081"
	defaultLogLevel       = "info"
)

// Config описывает параметры запуска сервиса
type Config struct {
	// RunAddress адрес, на котором слушает HTTP‑сервер
	RunAddress string
	// DatabaseURI строка подключения к PostgreSQL
	DatabaseURI string
	// AccrualAddress адрес системы начислений
	AccrualAddress string
	// LogLevel уровень логирования
	LogLevel string
}

func Load() (Config, error) {
	return Parse(os.Args[1:], os.LookupEnv)
}

func Parse(args []string, lookupEnv func(string) (string, bool)) (Config, error) {
	cfg := Config{
		RunAddress:     defaultRunAddress,
		DatabaseURI:    defaultDatabaseURI,
		AccrualAddress: defaultAccrualAddress,
		LogLevel:       defaultLogLevel,
	}
	if v, ok := lookupEnv("RUN_ADDRESS"); ok {
		cfg.RunAddress = v
	}
	if v, ok := lookupEnv("DATABASE_URI"); ok {
		cfg.DatabaseURI = v
	}
	if v, ok := lookupEnv("ACCRUAL_SYSTEM_ADDRESS"); ok {
		cfg.AccrualAddress = v
	}
	if v, ok := lookupEnv("LOG_LEVEL"); ok {
		cfg.LogLevel = v
	}

	fs := pflag.NewFlagSet("gophermart", pflag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVarP(&cfg.RunAddress, "address", "a", cfg.RunAddress, "server run address")
	fs.StringVarP(&cfg.DatabaseURI, "database", "d", cfg.DatabaseURI, "database uri")
	fs.StringVarP(&cfg.AccrualAddress, "accrual", "r", cfg.AccrualAddress, "accrual system address")
	fs.StringVarP(&cfg.LogLevel, "log-level", "l", cfg.LogLevel, "log level")
	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	return cfg, nil
}
