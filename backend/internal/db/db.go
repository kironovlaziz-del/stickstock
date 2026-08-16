package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" driver
)

// Connect opens a connection pool to Postgres and retries a few times,
// since in docker-compose the DB container may not be ready yet when
// the API container starts.
func Connect(databaseURL string) (*sql.DB, error) {
	var dbConn *sql.DB
	var err error

	for attempt := 1; attempt <= 10; attempt++ {
		dbConn, err = sql.Open("pgx", databaseURL)
		if err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			pingErr := dbConn.PingContext(ctx)
			cancel()
			if pingErr == nil {
				break
			}
			err = pingErr
		}
		log.Printf("db: connection attempt %d/10 failed: %v", attempt, err)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		return nil, fmt.Errorf("could not connect to database after retries: %w", err)
	}

	dbConn.SetMaxOpenConns(20)
	dbConn.SetMaxIdleConns(10)
	dbConn.SetConnMaxLifetime(30 * time.Minute)

	return dbConn, nil
}

// RunMigrations applies any .sql files in migrationsDir in lexical order.
// This is a minimal migration runner; swap for golang-migrate later if
// the schema grows complex enough to need down-migrations and locking.
func RunMigrations(dbConn *sql.DB, statements []string) error {
	for i, stmt := range statements {
		if _, err := dbConn.Exec(stmt); err != nil {
			return fmt.Errorf("migration statement %d failed: %w", i, err)
		}
	}
	return nil
}
