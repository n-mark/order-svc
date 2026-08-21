# Order Service (Сервис заказов)

Микросервис заказов для интернет-сервиса по размещению объявлений (финальный проект). Оркестрирует жизненный цикл заказа: создание, оплату (через биллинг) и доставку. Написан на Go.

- **GitHub:** https://github.com/n-mark/order-svc
- **DockerHub:** [`mblkuta/ordersvc`](https://hub.docker.com/r/mblkuta/ordersvc)

## Возможности

- Создание и управление заказами (`/api/v1/order`)
- Публикация событий `order.created` / `order.updated` в Kafka (топик `order`)
- Подписка на события биллинга (`billing.payment.success`, `billing.payment.required`)
- Интеграция с сервисами объявлений, биллинга и доставки по HTTP

## Технологии

- Go, PostgreSQL, Kafka
- Docker / docker-compose

## Структура проекта

```text
main.go        # точка входа
internal/      # бизнес-логика, обработчики, хранилище
migrations/    # SQL-миграции
```

## Переменные окружения

| Переменная | Описание | Пример |
|---|---|---|
| `APP_PORT` | Порт HTTP-сервера | `8080` |
| `PG_HOST` / `PG_PORT` | Хост/порт PostgreSQL | `postgres-orders` / `5432` |
| `PG_DATABASE` | Имя БД | `orderdb` |
| `PG_USER` / `PG_PASSWORD` | Учётные данные БД | `order_user` |
| `PG_SSLMODE` | Режим SSL | `disable` |
| `BROKER_TYPE` | Тип брокера | `KAFKA` |
| `KAFKA_BROKERS` | Адреса брокеров Kafka | `kafka:9092` |
| `KAFKA_ORDER_TOPIC` | Топик событий заказов | `order` |
| `KAFKA_ORDER_CREATED_EVENT_TYPE` / `KAFKA_ORDER_UPDATED_EVENT_TYPE` | Типы событий | `order.created` / `order.updated` |
| `KAFKA_BILLING_TOPIC` / `KAFKA_BILLING_GROUP` | Топик/группа биллинга | `billing` / `ordersvc.billing` |
| `KAFKA_BILLING_EVENT_TYPES` | Типы событий биллинга | `billing.payment.success,billing.payment.required` |
| `ADVERT_CMD_URL` | URL сервиса объявлений | `http://advert-cmd-svc:8080` |
| `BILLING_URL` | URL сервиса биллинга | `http://billing-service:8080` |
| `DELIVERY_URL` | URL сервиса доставки | `http://delivery-service:8080` |

## Запуск

### Docker Compose

```bash
docker compose up -d
```

### Локально

```bash
go run ./main.go
```

## Эндпоинты

- `GET /health` — health-check
- `/api/v1/order/...` — операции с заказами

## Связанные репозитории

Инфраструктура всего проекта (k8s, Helm, docker-compose всего стека): https://github.com/n-mark/final-project