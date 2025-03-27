FROM golang:1.24-alpine AS builder

WORKDIR /app

ENV GOPROXY=https://proxy.golang.org,direct  
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

COPY check_postgres.go .
RUN go build -o check_postgres check_postgres.go
RUN go build -o auth-service  

# --- stage-1 ---
FROM alpine:latest

RUN apk add --no-cache ca-certificates bash  

WORKDIR /app

COPY --from=builder /app/auth-service .  
COPY --from=builder /app/check_postgres . 

CMD ["./check_postgres"] 
# CMD ["./auth-service"]  # Потом вернем эту строку