package config

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	MailopostAPIKey     string `envconfig:"MAILOPOST_API_KEY" required:"true"`
	MailopostSender     string `envconfig:"MAILOPOST_SENDER" required:"true"`
	RedisAddr           string `envconfig:"REDIS_ADDR" required:"true"`
	RedisPassword       string `envconfig:"REDIS_PASSWORD"`
	RedisDB             string `envconfig:"REDIS_DB" default:"0"`
	JwtSecret           string `envconfig:"JWT_SECRET" required:"true"`
	VerificationCodeTTL string `envconfig:"VERIFICATION_CODE_TTL" default:"300s"`
	AccessTokenTTL      string `envconfig:"ACCESS_TOKEN_TTL" default:"3600s"`
	RefreshTokenTTL     string `envconfig:"REFRESH_TOKEN_TTL" default:"604800s"`
	DatabaseURL         string `envconfig:"DATABASE_URL" required:"true"`
	Verification        VerificationConfig
	TestEmail           string `envconfig:"TEST_EMAIL"`
	TestPassword        string `envconfig:"TEST_PASSWORD"`
}

type VerificationConfig struct {
	CodeExpireTime string `envconfig:"CODE_EXPIRE_TIME" default:"5m"`
}

var Cfg Config

func LoadConfig() (Config, error) {
	cfg := Config{}

	err := envconfig.Process("", &cfg)
	if err != nil {
		return Config{}, fmt.Errorf("failed to load config from env: %w", err)
	}

	Cfg = cfg

	return cfg, nil
}
