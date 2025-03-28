FROM golang:latest

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

RUN go build -o email-service .

# Установка kafkacat
RUN apt-get update && apt-get install -y kafkacat # попробуйте сначала это
# если не работает, попробуйте это
# RUN apk update && apk add kafkacat

CMD ["./email-service"]
#  Удалите эти строки (или закомментируйте):
# ENV MAILOPOST_API_KEY=42b8da8ecf68106419c6208db9cf5353
# ENV MAILOPOST_SENDER=olegnemlii93@gmail.com


# docker run -d -e KAFKA_BROKER_ADDRESS=192.168.0.112:9092 -e KAFKA_TOPIC=email-notifications -e MAILOPOST_API_KEY=42b8da8ecf68106419c6208db9cf5353 -e MAILOPOST_SENDER=olegnemlii93@gmail.com my-email-service:latest

#docker run -d -e KAFKA_BROKER_ADDRESS=192.168.0.112:9092 -e KAFKA_TOPIC=email-notifications -e MAILOPOST_API_KEY=<your_mailopost_api_key> -e MAILOPOST_SENDER=<your_mailopost_sender_email> my-email-service:latest