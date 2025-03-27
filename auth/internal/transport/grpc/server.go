package grpc

import (
	"context"
	"log"
	"log/slog" // Добавляем импорт для strconv
	"time"

	"auth/config"
	"auth/internal/service"
	"auth/pkg/pb"
	authpb "auth/pkg/pb"

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

func (s *Server) Register(ctx context.Context, req *authpb.RegisterRequest) (*authpb.RegisterResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, status.Errorf(codes.InvalidArgument, "email and password are required")
	}

	signature, err := s.service.RegisterUser(ctx, req.Email, req.Password)
	if err != nil {
		slog.Error("failed to register", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to register: %v", err)
	}

	return &authpb.RegisterResponse{Signature: signature}, nil
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

	accessTokenTTL, err := time.ParseDuration(config.Cfg.AccessTokenTTL) // Преобразуем строку в time.Duration
	if err != nil {
		log.Printf("invalid AccessTokenTTL: %v", err)
		return nil, status.Errorf(codes.Internal, "invalid AccessTokenTTL")
	}

	refreshTokenTTL, err := time.ParseDuration(config.Cfg.RefreshTokenTTL) // Преобразуем строку в time.Duration
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

	accessTokenTTL, err := time.ParseDuration(config.Cfg.AccessTokenTTL) // Преобразуем строку в time.Duration
	if err != nil {
		log.Printf("invalid AccessTokenTTL: %v", err)
		return nil, status.Errorf(codes.Internal, "invalid AccessTokenTTL")
	}

	refreshTokenTTL, err := time.ParseDuration(config.Cfg.RefreshTokenTTL) // Преобразуем строку в time.Duration
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

	accessTokenTTL, err := time.ParseDuration(config.Cfg.AccessTokenTTL) // Преобразуем строку в time.Duration
	if err != nil {
		log.Printf("invalid AccessTokenTTL: %v", err)
		return nil, status.Errorf(codes.Internal, "invalid AccessTokenTTL")
	}

	refreshTokenTTL, err := time.ParseDuration(config.Cfg.RefreshTokenTTL) // Преобразуем строку в time.Duration
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
