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

## Обработка заказов

- Загрузка заказа через `POST /api/user/orders` только сохраняет заказ в `orders` и ставит задачу в очередь
- Очередь реализована в отдельной таблице PostgreSQL `order_jobs`, а не в таблице `orders`
- Фоновый процессор с пулом воркеров периодически выбирает задачи из `order_jobs`, блокирует их (`FOR UPDATE SKIP LOCKED`) и выполняет запросы в accrual-сервис
- По результату обработки воркеры обновляют статус заказа в `orders` (`NEW/PROCESSING/INVALID/PROCESSED`) и метаданные ретраев в `order_jobs`
- При ответе `429 Too Many Requests` применяется global pause для всех воркеров с учетом `Retry-After`

## Тестирование

```bash
go test ./... -cover
```

## Линтинг

```bash
golangci-lint run
```
