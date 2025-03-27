package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"email-service/mailpost" // Замените your_module на имя вашего Go модуля

	"github.com/segmentio/kafka-go"
)

// EmailMessage структура сообщения из Kafka
type EmailMessage struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

func main() {
	// Настройки Kafka
	kafkaBrokerAddress := os.Getenv("KAFKA_BROKER_ADDRESS")
	if kafkaBrokerAddress == "" {
		log.Fatal("KAFKA_BROKER_ADDRESS is not set")
	}
	kafkaTopic := os.Getenv("KAFKA_TOPIC")
	if kafkaTopic == "" {
		log.Fatal("KAFKA_TOPIC is not set")
	}

	// Настройки Mailopost
	mailopostAPIKey := os.Getenv("MAILOPOST_API_KEY")
	if mailopostAPIKey == "" {
		log.Fatal("MAILOPOST_API_KEY is not set")
	}
	mailopostSender := os.Getenv("MAILOPOST_SENDER")
	if mailopostSender == "" {
		log.Fatal("MAILOPOST_SENDER is not set")
	}

	// Создание Mailopost клиента
	mailClient := mailpost.NewClient(mailopostAPIKey, mailopostSender)

	// Создание Kafka reader
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{kafkaBrokerAddress},
		Topic:       kafkaTopic,
		GroupID:     "email-service-group", // Важно: укажите GroupID
		StartOffset: kafka.LastOffset,      // Читать новые сообщения
	})
	defer r.Close()

	fmt.Println("Listening for messages on Kafka...")

	for {
		// Чтение сообщения из Kafka
		m, err := r.ReadMessage(context.Background())
		if err != nil {
			log.Printf("Failed to read message: %v", err)
			break // Завершить цикл при ошибке чтения
		}

		fmt.Printf("Message Key: %s\n", string(m.Key))
		fmt.Printf("Message Value: %s\n", string(m.Value))

		// Обработка сообщения
		var messageData EmailMessage
		err = json.Unmarshal(m.Value, &messageData)
		if err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			continue // Перейти к следующему сообщению
		}

		// Отправка email через Mailopost
		err = mailClient.SendMessage(context.Background(), messageData.To, messageData.Subject, messageData.Body)
		if err != nil {
			log.Printf("Failed to send email via Mailopost: %v", err)
			continue // Перейти к следующему сообщению
		}

		fmt.Println("Email sent successfully via Mailopost!")
	}
}
