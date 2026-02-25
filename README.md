# gophermart

![lint](https://github.com/kayumovtd/gophermart/actions/workflows/lint.yml/badge.svg)
![coverage](https://codecov.io/gh/kayumovtd/gophermart/branch/main/graph/badge.svg)

HTTP API сервис накопительной системы лояльности «Гофермарт».

## Требования

- Go 1.24+
- PostgreSQL 17+

## Конфигурация

Переменные окружения:

- `RUN_ADDRESS` — адрес запуска сервиса (по умолчанию `localhost:8080`)
- `DATABASE_URI` — строка подключения к PostgreSQL
- `ACCRUAL_SYSTEM_ADDRESS` — адрес системы начислений
- `LOG_LEVEL` — уровень логирования (по умолчанию `info`)
- `AUTH_SECRET` — секрет подписи JWT (если не задан, генерируется при старте)

Флаги:

- `-a` — адрес запуска сервиса
- `-d` — строка подключения к PostgreSQL
- `-r` — адрес системы начислений
- `-l` — уровень логирования
- `--auth-secret` — секрет подписи JWT

Флаги имеют более высокий приоритет, чем переменные окружения.

## Локальный запуск (Go)

```bash
export DATABASE_URI="postgres://user:pass@localhost:5432/gophermart?sslmode=disable"
export ACCRUAL_SYSTEM_ADDRESS="http://localhost:8081"
export AUTH_SECRET="local-dev-secret"

go run ./cmd/gophermart -a 0.0.0.0:8080
```

При старте сервиса автоматически применяются миграции из директории `internal/migrations/`.

## Локальный запуск (Docker Compose)

```bash
docker-compose up --build
```

Сервис начислений является внешним. Для локальных проверок задайте `ACCRUAL_SYSTEM_ADDRESS`.
После `register/login` токен возвращается в заголовке `Authorization: Bearer <jwt>`.

## Тестирование

```bash
go test ./... -cover
```

## Линтинг

```bash
golangci-lint run
```
