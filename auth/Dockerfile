FROM golang:latest AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

RUN go build -o /app/auth .

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/auth /app/auth
COPY .env .

EXPOSE 50051

CMD ["/app/auth"] # Явно указываем путь к исполняемому файлу