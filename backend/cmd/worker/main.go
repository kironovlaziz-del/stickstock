package main

import (
	"context"
	"log"
	"time"

	"stickstock/backend/internal/cache"
	"stickstock/backend/internal/config"
	"stickstock/backend/internal/db"
	"stickstock/backend/internal/delivery"
	"stickstock/backend/internal/scheduler"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	database, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database error: %v", err)
	}
	defer database.Close()

	// Generous but bounded timeout for the whole batch — a stuck upstream
	// query source shouldn't be able to wedge the worker forever if it's
	// invoked frequently by cron.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	schedCfg := scheduler.Config{
		SMTP: delivery.SMTPConfig{
			Host:     cfg.SMTPHost,
			Port:     cfg.SMTPPort,
			Username: cfg.SMTPUsername,
			Password: cfg.SMTPPassword,
			From:     cfg.SMTPFrom,
		},
		Telegram: delivery.TelegramConfig{BotToken: cfg.TelegramBotToken},
	}

	if err := scheduler.RunDue(ctx, database, schedCfg); err != nil {
		log.Fatalf("scheduler run failed: %v", err)
	}

	// Piggybacks on the same cron trigger rather than needing its own
	// schedule — sweeping expired cache rows is cheap and doesn't need to
	// run more or less often than the report scheduler does.
	if removed, err := cache.New(database).Cleanup(ctx); err != nil {
		log.Printf("cache cleanup failed: %v", err)
	} else if removed > 0 {
		log.Printf("cache cleanup: removed %d expired entries", removed)
	}
}
