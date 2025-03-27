package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"auth/internal/adapter/mailpost"
	"auth/internal/domain"
	"auth/internal/repository"
	"auth/pkg/pb"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserRepository interface {
	CreateUser(ctx context.Context, email, hashedPassword string) (uuid.UUID, error)
	CreateVerificationCodeData(ctx context.Context, code string, userID uuid.UUID, signature string) error
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
	GetEmailBySignature(ctx context.Context, signature string) (string, error)
	ConfirmUser(ctx context.Context, userID uuid.UUID) error
	StoreRefreshToken(ctx context.Context, email string, refreshToken string) error
}

type MailSender interface {
	SendMessage(ctx context.Context, to, subject, body string) error
}

type TokenRepository interface {
	DeleteToken(ctx context.Context, accessToken string) error
	GetRefreshToken(ctx context.Context, accessToken string) (string, error)
}

type PasswordHasher interface {
	CompareHashAndPassword(hashedPassword, password string) bool
}

type JWTService interface {
	GenerateJWT(email string) (string, error)
	VerifyJWT(token string) (string, error)
}

type Service struct {
	userRepo       *repository.UserRepository
	tokenRepo      *repository.TokenRepository
	mailSender     *mailpost.Client
	passwordHasher PasswordHasher
	jwtService     JWTService
	// kafkaWriter    *kafka.Writer // Remove kafkaWriter
}

func NewService(userRepo *repository.UserRepository, tokenRepo *repository.TokenRepository, mailSender *mailpost.Client, passwordHasher PasswordHasher, jwtService JWTService) *Service {
	return &Service{userRepo: userRepo, tokenRepo: tokenRepo, mailSender: mailSender, passwordHasher: passwordHasher, jwtService: jwtService}
}

type VerificodeResponse struct {
	Access bool
}

func (s *Service) RegisterUser(ctx context.Context, email, password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	userID, err := s.userRepo.CreateUser(ctx, email, string(hashedPassword))
	if err != nil {
		return "", fmt.Errorf("failed to create user: %w", err)
	}

	verificationCode := s.generateRandomCode()

	signature, err := s.generateSignature()
	if err != nil {
		return "", fmt.Errorf("failed to generate signature: %w", err)
	}
	err = s.userRepo.CreateVerificationCodeData(ctx, verificationCode, userID, signature)
	if err != nil {
		return "", fmt.Errorf("failed to store verification code: %w", err)
	}

	// err = s.sendVerificationEmail(ctx, email, verificationCode)
	// if err != nil {
	// 	return "", fmt.Errorf("failed to send verification email: %w", err)
	// }
	subject := "Welcome!"
	body := "Welcome to our service!"
	err = s.SendEmailNotification(ctx, email, subject, body)
	if err != nil {
		log.Printf("failed to send email notification to kafka: %v", err)
		// Handle the error appropriately (e.g., log it)
	}

	return signature, nil
}

func (s *Service) generateSignature() (string, error) {
	id := uuid.New()
	return id.String(), nil
}

func (s *Service) generateRandomCode() string {
	var code string

	for range 4 {
		code += fmt.Sprintf("%d", rand.Intn(10))
	}

	return code
}

func (s *Service) ConfirmUser(ctx context.Context, userID uuid.UUID) error {
	if err := s.userRepo.ConfirmUser(ctx, userID); err != nil {
		return fmt.Errorf("failed to confirm user: %w", err)
	}
	return nil
}

func (s *Service) VerifyCode(ctx context.Context, code, signature string) (string, string, *domain.User, error) {
	email, err := s.userRepo.GetEmailBySignature(ctx, signature)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to get email by signature: %w", err)
	}

	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to get user: %w", err)
	}

	if err := s.userRepo.ConfirmUser(ctx, user.ID); err != nil {
		return "", "", nil, fmt.Errorf("failed to confirm user: %w", err)
	}

	accessToken, err := s.jwtService.GenerateJWT(user.Email)
	if err != nil {
		return "", "", nil, status.Errorf(codes.Internal, "failed to generate access token")
	}

	refreshToken, err := GenerateRefreshToken()
	if err != nil {
		return "", "", nil, status.Errorf(codes.Internal, "failed to generate refresh token")
	}

	if err := s.tokenRepo.StoreRefreshToken(ctx, user.Email, refreshToken); err != nil {
		return "", "", nil, status.Errorf(codes.Internal, "failed to store refresh token")
	}

	return accessToken, refreshToken, user, nil
}
func (s *Service) GetMe(ctx context.Context, email string) (*pb.User, error) {
	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &pb.User{
		Id:    user.ID.String(),
		Email: user.Email,
	}, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (string, string, *domain.User, error) {
	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return "", "", nil, status.Errorf(codes.NotFound, "user not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", "", nil, status.Errorf(codes.Unauthenticated, "invalid credentials")
	}

	accessToken, err := s.jwtService.GenerateJWT(user.Email)
	if err != nil {
		return "", "", nil, status.Errorf(codes.Internal, "failed to generate access token")
	}

	refreshToken, err := GenerateRefreshToken()
	if err != nil {
		return "", "", nil, status.Errorf(codes.Internal, "failed to generate refresh token")
	}

	if err := s.tokenRepo.StoreRefreshToken(ctx, email, refreshToken); err != nil {
		return "", "", nil, status.Errorf(codes.Internal, "failed to store refresh token")
	}

	return accessToken, refreshToken, user, nil
}

func GenerateRefreshToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}
func (s *Service) VerifyAndRefreshTokens(ctx context.Context, accessToken, refreshToken string) (string, string, *domain.User, error) {
	email, err := s.jwtService.VerifyJWT(accessToken)
	if err != nil {
		log.Println("invalid access token")
		return "", "", nil, fmt.Errorf("invalid access token: %w", err)
	}

	storedRefreshToken, err := s.tokenRepo.GetRefreshToken(ctx, email)
	if err != nil {
		log.Println("failed to get stored refresh token")
		return "", "", nil, fmt.Errorf("failed to get stored refresh token: %w", err)
	}

	if refreshToken != storedRefreshToken {
		log.Println("invalid refresh token")
		return "", "", nil, fmt.Errorf("invalid refresh token")
	}

	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		log.Println("failed to get user")
		return "", "", nil, fmt.Errorf("failed to get user: %w", err)
	}

	newAccessToken, err := s.jwtService.GenerateJWT(user.Email)
	if err != nil {
		log.Println("failed to generate access token")
		return "", "", nil, fmt.Errorf("failed to generate access token: %w", err)
	}
	log.Println("Generated new access token:", newAccessToken)
	newRefreshToken, err := GenerateRefreshToken()
	if err != nil {
		log.Println("failed to generate refresh token")
		return "", "", nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}
	log.Println("Generated new refresh token:", newRefreshToken)
	log.Println("Deleting refresh token in database from email:", email)

	if err := s.tokenRepo.DeleteToken(ctx, email); err != nil {
		log.Println("failed to delete old refresh token")
		return "", "", nil, fmt.Errorf("failed to delete old refresh token: %w", err)
	}
	log.Println("save refresh token in database from email:", email)

	if err := s.tokenRepo.StoreRefreshToken(ctx, email, newRefreshToken); err != nil {
		log.Println("failed to store new refresh token")
		return "", "", nil, fmt.Errorf("failed to store new refresh token: %w", err)
	}
	log.Println("refresh token successfully saved to the database!")

	return newAccessToken, newRefreshToken, user, nil
}
func (s *Service) LogOut(ctx context.Context, accessToken string) error {
	email, err := s.jwtService.VerifyJWT(accessToken)
	if err != nil {
		return status.Errorf(codes.Unauthenticated, "invalid access token")
	}

	if err := s.tokenRepo.DeleteToken(ctx, email); err != nil {
	}

	return nil
}

func (s *Service) VerifyJWT(token string) (string, error) {
	return s.jwtService.VerifyJWT(token)
}

// func (s *Service) sendVerificationEmail(ctx context.Context, email, code string) error {

// 	subject := "Код подтверждения"
// 	body := fmt.Sprintf("Ваш код подтверждения: %s", code)

// 	err := s.mailSender.SendMessage(ctx, email, subject, body)
// 	if err != nil {
// 		return fmt.Errorf("failed to send verification email: %w", err)
// 	}
// 	return nil
// }

func (s *Service) SendEmailNotification(ctx context.Context, email, subject, body string) error {
	kafkaBrokerAddress := os.Getenv("KAFKA_BROKER_ADDRESS")
	if kafkaBrokerAddress == "" {
		log.Println("KAFKA_BROKER_ADDRESS is not set")
		return fmt.Errorf("KAFKA_BROKER_ADDRESS is not set") // Don't use Fatal, return error
	}
	kafkaTopic := os.Getenv("KAFKA_TOPIC")
	if kafkaTopic == "" {
		log.Println("KAFKA_TOPIC is not set")
		return fmt.Errorf("KAFKA_TOPIC is not set") // Don't use Fatal, return error
	}

	// Create Kafka writer
	w := &kafka.Writer{
		Addr:         kafka.TCP(kafkaBrokerAddress),
		Topic:        kafkaTopic,
		Balancer:     &kafka.LeastBytes{}, // используем RoundRobin
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
	}
	defer w.Close()

	// Create Kafka message
	message := kafka.Message{
		Key:   []byte(email), // Use email as key
		Value: []byte(fmt.Sprintf(`{"to": "%s", "subject": "%s", "body": "%s"}`, email, subject, body)),
	}

	// Send message to Kafka
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := w.WriteMessages(ctx, message)
	if err != nil {
		return fmt.Errorf("failed to write messages: %w", err)
	}

	log.Println("Message send successfully to Kafka!")
	return nil
}
