FROM golang:1.24-alpine AS builder
WORKDIR /app
# COPY .env .
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o email-service

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/email-service /app/email-service
CMD ["/app/email-service"]
#  Удалите эти строки (или закомментируйте):
# ENV MAILOPOST_API_KEY=42b8da8ecf68106419c6208db9cf5353
# ENV MAILOPOST_SENDER=olegnemlii93@gmail.com


# docker run -d -e KAFKA_BROKER_ADDRESS=192.168.0.112:9092 -e KAFKA_TOPIC=email-notifications -e MAILOPOST_API_KEY=42b8da8ecf68106419c6208db9cf5353 -e MAILOPOST_SENDER=olegnemlii93@gmail.com my-email-service:latest

#docker run -d -e KAFKA_BROKER_ADDRESS=192.168.0.112:9092 -e KAFKA_TOPIC=email-notifications -e MAILOPOST_API_KEY=<your_mailopost_api_key> -e MAILOPOST_SENDER=<your_mailopost_sender_email> my-email-service:latest