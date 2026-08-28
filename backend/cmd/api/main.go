package main

import (
	"path/filepath"
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"stickstock/backend/internal/api"
	"stickstock/backend/internal/config"
	"stickstock/backend/internal/db"
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

	// ── Apply migrations ──────────────────────────────────────────
	migrationFiles, err := filepath.Glob("./migrations/*.sql")
	if err != nil {
		log.Fatalf("could not list migration files: %v", err)
	}
	var statements []string
	for _, f := range migrationFiles {
		content, err := os.ReadFile(f)
		if err != nil {
			log.Fatalf("could not read migration file %s: %v", f, err)
		}
		statements = append(statements, string(content))
	}
	if len(statements) > 0 {
		log.Printf("Applying %d migration(s)...", len(statements))
		if err := db.RunMigrations(database, statements); err != nil {
			log.Fatalf("migrations failed: %v", err)
		}
		log.Println("Migrations applied successfully")
	}

	router := api.NewRouter(cfg, database)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("stickstock api listening on :%s (env=%s)", cfg.Port, cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Graceful shutdown on SIGINT/SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}