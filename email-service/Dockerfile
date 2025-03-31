FROM golang:latest

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

RUN go build -o email-service .
RUN apt-get update && apt-get install -y kafkacat
RUN apt-get update -y  
RUN apt-get install iputils-ping telnet -y 

CMD ["/bin/sh", "-c", "sleep 30 && /app/email-service"]

