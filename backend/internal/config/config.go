package config

import (
	"fmt"
	"os"
)

// Config holds all runtime configuration for the API service.
// Values are populated from environment variables (see .env.example),
// which docker-compose injects via env_file.
type Config struct {
	Env                 string // "development" | "production"
	Port                string
	DatabaseURL         string // Supabase Postgres connection string (Settings → Database)
	JWTSecret           string // Supabase project's JWT secret (Settings → API) — verifies tokens Supabase Auth issues
	SupabaseURL         string // e.g. https://xxxx.supabase.co — used to build the JWKS discovery URL for ES256 tokens
	AnalyticsServiceURL string // internal URL of the Python analytics service, e.g. http://analytics-service:8000
	LocalesDir          string
	DefaultLocale       string
	AllowedOrigins      string
	EncryptionKey       string
	DeveloperUserID     string

	// Delivery channels for cmd/worker's scheduled reports. Both optional —
	// a deployment only using one of email/telegram doesn't need the other
	// configured; SendEmail/SendTelegramMessage error clearly at delivery
	// time if a report tries to use an unconfigured channel.
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
		EncryptionKey:       os.Getenv("ENCRYPTION_KEY"),
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
