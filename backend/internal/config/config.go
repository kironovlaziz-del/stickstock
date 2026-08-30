package config

import (
	"fmt"
	"os"
)

type Config struct {
	Env                 string
	Port                string
	DatabaseURL         string
	JWTSecret           string
	SupabaseURL         string
	AnalyticsServiceURL string
	LocalesDir          string
	DefaultLocale       string
	AllowedOrigins      string
	DeveloperUserID     string

	SMTPHost         string
	SMTPPort         string
	SMTPUsername     string
	SMTPPassword     string
	SMTPFrom         string
	TelegramBotToken string
}

func Load() (*Config, error) {
	cfg := &Config{
		Env:                 getEnv("APP_ENV", "development"),
		Port:                getEnv("PORT", "8080"),
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		JWTSecret:           os.Getenv("JWT_SECRET"),
		SupabaseURL:         os.Getenv("SUPABASE_URL"),
		AnalyticsServiceURL: getEnv("ANALYTICS_SERVICE_URL", "http://analytics-service:8000"),
		LocalesDir:          getEnv("LOCALES_DIR", "/app/locales"),
		DefaultLocale:       getEnv("DEFAULT_LOCALE", "en"),
		AllowedOrigins:      getEnv("ALLOWED_ORIGINS", "*"),
		DeveloperUserID:     os.Getenv("DEVELOPER_USER_ID"),
		SMTPHost:            os.Getenv("SMTP_HOST"),
		SMTPPort:            getEnv("SMTP_PORT", "587"),
		SMTPUsername:        os.Getenv("SMTP_USERNAME"),
		SMTPPassword:        os.Getenv("SMTP_PASSWORD"),
		SMTPFrom:            os.Getenv("SMTP_FROM"),
		TelegramBotToken:    os.Getenv("TELEGRAM_BOT_TOKEN"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
