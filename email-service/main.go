package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"email-service/mailpost"

	"github.com/segmentio/kafka-go"
)

type EmailMessage struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
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

	for {
		r := kafka.NewReader(kafka.ReaderConfig{
			Brokers:     []string{kafkaBrokerAddress},
			Topic:       kafkaTopic,
			GroupID:     consumerGroupName,
			StartOffset: kafka.LastOffset,
		})
		//defer r.Close() // Не закрываем reader через defer

		fmt.Println("Listening for messages on Kafka...")

		for {
			log.Println("Attempting to read message from Kafka...") // Лог перед чтением

			m, err := r.ReadMessage(context.Background())
			if err != nil {
				log.Printf("Failed to read message: %v", err)

				if err == io.EOF {
					log.Println("Connection closed by Kafka. Reconnecting...")
					break // Выходим из внутреннего цикла для переподключения
				} else if err.Error() == "[15] Group Coordinator Not Available" {
					log.Println("Group Coordinator Not Available. Retrying in 5 seconds...")
					time.Sleep(5 * time.Second)
					break // Выходим из внутреннего цикла, чтобы пересоздать reader
				} else {
					log.Printf("Received message from Kafka: %s", string(m.Value))
					log.Printf("Unexpected error: %v", err)
					time.Sleep(5 * time.Second)
					continue // Переходим к следующей итерации цикла
				}
			}

			log.Printf("Message Key: %s\n", string(m.Key)) // Лог после получения сообщения
			log.Printf("Message Value: %s\n", string(m.Value))

			var messageData EmailMessage
			err = json.Unmarshal(m.Value, &messageData)
			if err != nil {
				log.Printf("Failed to unmarshal message: %v", err)
				continue // Перейти к следующему сообщению
			}

			err = mailClient.SendMessage(context.Background(), messageData.To, messageData.Subject, messageData.Body)
			if err != nil {
				log.Printf("Failed to send email via Mailopost: %v", err)
				continue // Перейти к следующему сообщению
			}
			log.Println("Email sent successfully via Mailopost!")
			fmt.Println("Email sent successfully via Mailopost!")
		}

		err := r.Close() // Закрываем reader явно и обрабатываем ошибку
		if err != nil {
			log.Printf("Error closing Kafka reader: %v", err)
		}
		time.Sleep(5 * time.Second)
		log.Println("Reconnecting to Kafka...")
	}
}
