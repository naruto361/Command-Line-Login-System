package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	DatabasePath             string
	MaxFailedLoginAttempts   int
	MaxTOTPAttempts          int
	SessionTimeout           time.Duration
	SessionWarningBefore     time.Duration
	PasswordResetTokenExpiry time.Duration
	DevMode                  bool

	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string
	AppName      string
}

func Load() (*Config, error) {
	cfg := &Config{
		DatabasePath:             getEnv("DATABASE_PATH", "/app/data/app.db"),
		MaxFailedLoginAttempts:   getEnvInt("MAX_FAILED_LOGIN_ATTEMPTS", 5),
		MaxTOTPAttempts:          getEnvInt("MAX_TOTP_ATTEMPTS", 3),
		SessionTimeout:           time.Duration(getEnvInt("SESSION_TIMEOUT_MINUTES", 30)) * time.Minute,
		SessionWarningBefore:     time.Duration(getEnvInt("SESSION_WARNING_MINUTES", 5)) * time.Minute,
		PasswordResetTokenExpiry: time.Duration(getEnvInt("PASSWORD_RESET_TOKEN_EXPIRY_MINUTES", 60)) * time.Minute,
		DevMode:                  getEnvBool("DEV_MODE", true),
		SMTPHost:                 getEnv("SMTP_HOST", ""),
		SMTPPort:                 getEnvInt("SMTP_PORT", 587),
		SMTPUser:                 getEnv("SMTP_USER", ""),
		SMTPPassword:             getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:                 getEnv("SMTP_FROM", "noreply@osto.local"),
		AppName:                  getEnv("APP_NAME", "OSTO"),
	}

	if cfg.MaxFailedLoginAttempts < 1 {
		return nil, fmt.Errorf("MAX_FAILED_LOGIN_ATTEMPTS must be at least 1")
	}
	if cfg.MaxTOTPAttempts < 1 {
		return nil, fmt.Errorf("MAX_TOTP_ATTEMPTS must be at least 1")
	}
	if cfg.SessionWarningBefore >= cfg.SessionTimeout {
		return nil, fmt.Errorf("SESSION_WARNING_MINUTES must be less than SESSION_TIMEOUT_MINUTES")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getEnvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}
