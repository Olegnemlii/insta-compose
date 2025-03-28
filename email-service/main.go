package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

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
	time.Sleep(10 * time.Second)

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

	consumerGroupName := "email-service-group" // Важно: укажите GroupID

	// Бесконечный цикл для переподключения к Kafka
	for {
		// Создание Kafka reader
		r := kafka.NewReader(kafka.ReaderConfig{
			Brokers:     []string{kafkaBrokerAddress},
			Topic:       kafkaTopic,
			GroupID:     consumerGroupName, // Важно: укажите GroupID
			StartOffset: kafka.LastOffset,  // Читать новые сообщения
		})
		defer r.Close() // defer для закрытия reader

		fmt.Println("Listening for messages on Kafka...")

		// Цикл чтения сообщений
		for {
			m, err := r.ReadMessage(context.Background())
			if err != nil {
				log.Printf("Failed to read message: %v", err)
				if err.Error() == "[15] Group Coordinator Not Available" {
					log.Println("Group Coordinator Not Available. Retrying in 5 seconds...")
					time.Sleep(5 * time.Second)
					break // Выходим из внутреннего цикла, чтобы пересоздать reader
				} else {
					log.Printf("Received message from Kafka: %s", string(m.Value)) // Добавьте эту строку
					log.Printf("Unexpected error: %v", err)
					time.Sleep(5 * time.Second) // Пауза перед повторной попыткой чтения
					continue                    //  Переходим к следующей итерации цикла
				}

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
			log.Println("Email sent successfully via Mailopost!") // Добавьте эту строку
			fmt.Println("Email sent successfully via Mailopost!")
		}
		r.Close()                               // Закрываем reader после ошибки
		time.Sleep(5 * time.Second)             // Ждем перед пересозданием reader
		log.Println("Reconnecting to Kafka...") // Сообщение о переподключении
	}
}
