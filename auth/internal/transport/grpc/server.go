package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"math/rand"
	"os"
	"time"

	"auth/config"
	"auth/internal/service"
	"auth/pkg/pb"
	authpb "auth/pkg/pb"

	"github.com/segmentio/kafka-go"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	authpb.UnimplementedAuthServer
	service *service.Service
}

func NewServer(service *service.Service) *Server {
	return &Server{service: service}
}

func generateVerificationCode() string {
	rand.Seed(time.Now().UnixNano())
	code := rand.Intn(9000) + 1000
	return fmt.Sprintf("%d", code)
}

func (s *Server) Register(ctx context.Context, req *authpb.RegisterRequest) (*authpb.RegisterResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, status.Errorf(codes.InvalidArgument, "email and password are required")
	}

	signature, err := s.service.RegisterUser(ctx, req.Email, req.Password)
	if err != nil {
		slog.Error("failed to register", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to register: %v", err)
	}

	kafkaBrokerAddress := os.Getenv("KAFKA_BROKER_ADDRESS")
	if kafkaBrokerAddress == "" {
		log.Println("KAFKA_BROKER_ADDRESS is not set")
		return nil, fmt.Errorf("KAFKA_BROKER_ADDRESS is not set")
	}
	kafkaTopic := os.Getenv("KAFKA_TOPIC")
	if kafkaTopic == "" {
		log.Println("KAFKA_TOPIC is not set")
		return nil, fmt.Errorf("KAFKA_TOPIC is not set")
	}

	kafkaWriter := &kafka.Writer{
		Addr:         kafka.TCP(kafkaBrokerAddress),
		Topic:        kafkaTopic,
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
	}
	defer func() {
		if err := kafkaWriter.Close(); err != nil {
			log.Printf("failed to close kafka writer: %v", err)
		}
	}()

	verificationCode := generateVerificationCode()

	err = s.SendEmailNotification(ctx, kafkaWriter, req.Email, "Welcome!", "", verificationCode)
	if err != nil {
		log.Printf("failed to send email notification: %v", err)
		return nil, err
	}

	return &authpb.RegisterResponse{Signature: signature}, nil
}

func (s *Server) SendEmailNotification(ctx context.Context, kafkaWriter *kafka.Writer, to, subject, body string, verificationCode string) error {
	messageBytes, err := json.Marshal(map[string]string{
		"to":               to,
		"subject":          subject,
		"body":             body,
		"verificationCode": verificationCode,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	msg := kafka.Message{

		Value: messageBytes,
	}

	err = kafkaWriter.WriteMessages(ctx, msg)
	if err != nil {
		return fmt.Errorf("failed to write messages: %w", err)
	}

	log.Printf("sent message to topic %s: %s\n", os.Getenv("KAFKA_TOPIC"), string(messageBytes))
	return nil
}

func (s *Server) VerifyCode(ctx context.Context, req *authpb.VerifyCodeRequest) (*authpb.VerifyCodeResponse, error) {
	log.Println("VerifyCode вызван")

	if req.GetCode() == "" || req.GetSignature() == "" {
		log.Println("error: code or signature is empty")
		return nil, status.Errorf(codes.InvalidArgument, "code and signature are required")
	}

	accessToken, refreshToken, user, err := s.service.VerifyCode(ctx, req.GetCode(), req.GetSignature())
	if err != nil {
		log.Printf("failed to verifycation: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to verify code")
	}

	accessTokenTTL, err := time.ParseDuration(config.Cfg.AccessTokenTTL)
	if err != nil {
		log.Printf("invalid AccessTokenTTL: %v", err)
		return nil, status.Errorf(codes.Internal, "invalid AccessTokenTTL")
	}

	refreshTokenTTL, err := time.ParseDuration(config.Cfg.RefreshTokenTTL)
	if err != nil {
		log.Printf("invalid RefreshTokenTTL: %v", err)
		return nil, status.Errorf(codes.Internal, "invalid RefreshTokenTTL")
	}

	return &authpb.VerifyCodeResponse{
		AccessToken: &authpb.Token{
			Data:      accessToken,
			ExpiresAt: time.Now().Add(accessTokenTTL).Unix(),
		},
		RefreshToken: &authpb.Token{
			Data:      refreshToken,
			ExpiresAt: time.Now().Add(refreshTokenTTL).Unix(),
		},
		User: &authpb.User{
			Id:    user.ID.String(),
			Email: user.Email,
		},
	}, nil
}

func (s *Server) GetMe(ctx context.Context, req *authpb.GetMeRequest) (*authpb.GetMeResponse, error) {
	log.Println("GetMe вызван")

	accessTokenString := req.GetAccessToken().GetData()
	if accessTokenString == "" {
		log.Println("error: access token is empty")
		return nil, status.Errorf(codes.InvalidArgument, "access token is required")
	}

	email, err := s.service.VerifyJWT(accessTokenString)
	if err != nil {
		log.Printf("failed to verify access token: %v, token: %s", err, accessTokenString)
		return nil, status.Errorf(codes.Unauthenticated, "invalid access token")
	}

	user, err := s.service.GetMe(ctx, email)
	if err != nil {
		log.Printf("failed to get user by email: %v, email: %s", err, email)
		return nil, status.Errorf(codes.Internal, "failed to get user")
	}

	return &authpb.GetMeResponse{
		User: &authpb.User{
			Id:    user.Id,
			Email: user.Email,
		},
	}, nil
}

func (s *Server) VerifyAndRefreshTokens(ctx context.Context, req *authpb.VerifyAndRefreshTokensRequest) (*authpb.VerifyAndRefreshTokensResponse, error) {
	log.Println("refreshTokens is called")

	if req.GetRefreshToken().GetData() == "" || req.GetAccessToken().GetData() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "refresh token and access token are required")
	}

	newAccessToken, newRefreshToken, user, err := s.service.VerifyAndRefreshTokens(ctx, req.GetAccessToken().GetData(), req.GetRefreshToken().GetData())
	if err != nil {
		log.Printf("failed to refresh tokens: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to refresh tokens")
	}

	accessTokenTTL, err := time.ParseDuration(config.Cfg.AccessTokenTTL)
	if err != nil {
		log.Printf("invalid AccessTokenTTL: %v", err)
		return nil, status.Errorf(codes.Internal, "invalid AccessTokenTTL")
	}

	refreshTokenTTL, err := time.ParseDuration(config.Cfg.RefreshTokenTTL)
	if err != nil {
		log.Printf("invalid RefreshTokenTTL: %v", err)
		return nil, status.Errorf(codes.Internal, "invalid RefreshTokenTTL")
	}

	return &authpb.VerifyAndRefreshTokensResponse{
		AccessToken: &authpb.Token{
			Data:      newAccessToken,
			ExpiresAt: time.Now().Add(accessTokenTTL).Unix(),
		},
		RefreshToken: &authpb.Token{
			Data:      newRefreshToken,
			ExpiresAt: time.Now().Add(refreshTokenTTL).Unix(),
		},
		User: &authpb.User{
			Id:    user.ID.String(),
			Email: user.Email,
		},
	}, nil
}

func (s *Server) Login(ctx context.Context, req *authpb.LoginRequest) (*authpb.LoginResponse, error) {
	log.Println("login is called")

	if req.GetEmail() == "" || req.GetPassword() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "email and password are required")
	}

	accessTokenString, refreshTokenString, user, err := s.service.Login(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		log.Printf("Login error: %v", err)
		return nil, err
	}

	accessTokenTTL, err := time.ParseDuration(config.Cfg.AccessTokenTTL)
	if err != nil {
		log.Printf("invalid AccessTokenTTL: %v", err)
		return nil, status.Errorf(codes.Internal, "invalid AccessTokenTTL")
	}

	refreshTokenTTL, err := time.ParseDuration(config.Cfg.RefreshTokenTTL)
	if err != nil {
		log.Printf("invalid RefreshTokenTTL: %v", err)
		return nil, status.Errorf(codes.Internal, "invalid RefreshTokenTTL")
	}

	return &authpb.LoginResponse{
		AccessToken: &authpb.Token{
			Data:      accessTokenString,
			ExpiresAt: time.Now().Add(accessTokenTTL).Unix(),
		},
		RefreshToken: &authpb.Token{
			Data:      refreshTokenString,
			ExpiresAt: time.Now().Add(refreshTokenTTL).Unix(),
		},
		User: &authpb.User{
			Id:    user.ID.String(),
			Email: user.Email,
		},
	}, nil
}

func (s *Server) LogOut(ctx context.Context, req *pb.LogOutRequest) (*pb.LogOutResponse, error) {
	log.Println("logOut is called")

	if req.GetAccessToken().GetData() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "access token is required")
	}

	err := s.service.LogOut(ctx, req.GetAccessToken().GetData())
	if err != nil {
		log.Printf("exit error: %v", err)
		return nil, err
	}

	return &authpb.LogOutResponse{Success: true}, nil
}
