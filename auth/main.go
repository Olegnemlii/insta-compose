package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"os" // Import os package
	"time"

	"google.golang.org/grpc"

	"auth/config"
	"auth/internal/adapter/jwtservice"
	"auth/internal/adapter/mailpost"
	"auth/internal/adapter/passwordhasher"
	"auth/internal/repository"
	"auth/internal/service"
	server "auth/internal/transport/grpc"

	authpb "auth/pkg/pb"
	"auth/pkg/postgres"

	_ "github.com/lib/pq"
)

func Run() error {
	time.Sleep(5 * time.Second)
	rand.Seed(time.Now().UnixNano())
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	// Check Kafka broker address and topic
	kafkaBrokerAddress := os.Getenv("KAFKA_BROKER_ADDRESS")
	if kafkaBrokerAddress == "" {
		log.Fatal("KAFKA_BROKER_ADDRESS is not set")
	}
	kafkaTopic := os.Getenv("KAFKA_TOPIC")
	if kafkaTopic == "" {
		log.Fatal("KAFKA_TOPIC is not set")
	}

	postgresConn, err := postgres.NewPostgresConn(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer postgresConn.Close()

	repository := repository.NewUserRepository(postgresConn.DB, cfg.Verification)

	mailSender := mailpost.NewClient(cfg.MailopostAPIKey, cfg.MailopostSender)

	tokenRepo := repository.NewTokenRepository(postgresConn.DB)

	passwordHasher := passwordhasher.NewPasswordHasher()

	JWTService := jwtservice.NewJWTService(cfg.JwtSecret)

	service := service.NewService(repository, tokenRepo, mailSender, passwordHasher, JWTService)

	grpcServer := grpc.NewServer()

	authpb.RegisterAuthServer(grpcServer, server.NewServer(service))

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	return grpcServer.Serve(lis)
}

func main() {
	if err := Run(); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}
