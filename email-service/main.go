package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"email-service/mailpost"

	"github.com/segmentio/kafka-go"
)

type EmailMessage struct {
	To               string `json:"to"`
	Subject          string `json:"subject"`
	Body             string `json:"body"`
	VerificationCode string `json:"verificationCode"`
}

func main() {
	time.Sleep(10 * time.Second)

	kafkaBrokerAddress := os.Getenv("KAFKA_BROKER_ADDRESS")
	if kafkaBrokerAddress == "" {
		log.Fatal("KAFKA_BROKER_ADDRESS is not set")
	}
	kafkaTopic := os.Getenv("KAFKA_TOPIC")
	if kafkaTopic == "" {
		log.Fatal("KAFKA_TOPIC is not set")
	}

	mailopostAPIKey := os.Getenv("MAILOPOST_API_KEY")
	if mailopostAPIKey == "" {
		log.Fatal("MAILOPOST_API_KEY is not set")
	}
	mailopostSender := os.Getenv("MAILOPOST_SENDER")
	if mailopostSender == "" {
		log.Fatal("MAILOPOST_SENDER is not set")
	}

	mailClient := mailpost.NewClient(mailopostAPIKey, mailopostSender)

	consumerGroupName := "email-service-group"

	config := kafka.ReaderConfig{
		Brokers:  []string{kafkaBrokerAddress},
		GroupID:  consumerGroupName,
		Topic:    kafkaTopic,
		MinBytes: 10e3,
		MaxBytes: 10e6,
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		<-signals
		fmt.Println("interrupt is detected")
		cancel()
	}()

	reader := kafka.NewReader(config)
	defer reader.Close()

	fmt.Println("starting a new kafka reader")
	for {
		m, err := reader.ReadMessage(ctx)
		if err != nil {
			log.Printf("could not read message %s", err.Error())
			break
		}
		fmt.Printf("message at topic:%v partition:%v offset:%v    %s = %s\n", m.Topic, m.Partition, m.Offset, string(m.Key), string(m.Value))

		var messageData EmailMessage
		err = json.Unmarshal(m.Value, &messageData)
		if err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			continue
		}

		emailBody := fmt.Sprintf("%s\n\nYour verification code is: %s", messageData.Body, messageData.VerificationCode)

		err = mailClient.SendMessage(ctx, messageData.To, messageData.Subject, emailBody)
		if err != nil {
			log.Printf("Failed to send email via Mailopost: %v", err)
			continue
		}
		log.Println("Email sent successfully via Mailopost!")
		fmt.Println("Email sent successfully via Mailopost!")

	}
}
