FROM golang:latest AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

# Установите goose
RUN go install github.com/pressly/goose/v3/cmd/goose@latest

# Установите переменные окружения для миграции
ARG POSTGRES_USER
ARG POSTGRES_PASSWORD
ARG POSTGRES_DB
ENV DATABASE_URL="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}?sslmode=disable"

# Установите psql (PostgreSQL client) - НУЖЕН ТОЛЬКО ДЛЯ МИГРАЦИЙ
RUN apt-get update && apt-get install -y postgresql-client

RUN CGO_ENABLED=0 GOOS=linux go build -o auth .

FROM alpine/git AS migrations

WORKDIR /migrations

COPY --from=builder /app/migrations .

FROM golang:alpine

WORKDIR /app

COPY --from=builder /app/auth .
COPY --from=migrations /migrations ./migrations
COPY --from=builder /go/bin/goose /usr/local/bin/goose

RUN chmod +x auth
RUN chmod +x /usr/local/bin/goose

EXPOSE 50051

CMD ["./auth"]