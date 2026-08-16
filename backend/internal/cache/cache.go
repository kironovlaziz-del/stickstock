// Package cache provides a small key-value cache backed by a Postgres
// table (see migrations/0003_cache.sql) instead of Redis — one less
// service to run on a resource-constrained VPS. It's slower than Redis
// for high-frequency access (every op is a round trip to Postgres), which
// is a real trade-off; fine for things like "cache a data source's schema
// listing for a few minutes" and not the right tool for a hot counter.
package cache

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

type Store struct {
	DB *sql.DB
}

func New(db *sql.DB) *Store {
	return &Store{DB: db}
}

// Set stores value (JSON-marshaled) under key. ttl <= 0 means "never
// expires" — use that deliberately, not by accident.
func (s *Store) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	var expiresAt sql.NullTime
	if ttl > 0 {
		expiresAt = sql.NullTime{Time: time.Now().Add(ttl), Valid: true}
	}

	_, err = s.DB.ExecContext(ctx,
		`INSERT INTO cache (key, value, expires_at) VALUES ($1, $2, $3)
		 ON CONFLICT (key) DO UPDATE
		 SET value = EXCLUDED.value, expires_at = EXCLUDED.expires_at, created_at = now()`,
		key, string(data), expiresAt,
	)
	return err
}

// Get unmarshals the cached value into dest and reports whether it was
// found (and not expired). An expired row is deleted on read rather than
// needing a separate sweeper process.
func (s *Store) Get(ctx context.Context, key string, dest interface{}) (bool, error) {
	var raw []byte
	var expiresAt sql.NullTime

	err := s.DB.QueryRowContext(ctx,
		`SELECT value, expires_at FROM cache WHERE key = $1`, key,
	).Scan(&raw, &expiresAt)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	if expiresAt.Valid && time.Now().After(expiresAt.Time) {
		_, _ = s.DB.ExecContext(ctx, `DELETE FROM cache WHERE key = $1`, key)
		return false, nil
	}

	if err := json.Unmarshal(raw, dest); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Store) Delete(ctx context.Context, key string) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM cache WHERE key = $1`, key)
	return err
}

// Cleanup removes all expired rows — call this from the scheduled worker
// (see README "Воркер") rather than a background goroutine, to match the
// "don't keep processes running, invoke on a schedule" approach the
// recommended architecture uses for the worker.
func (s *Store) Cleanup(ctx context.Context) (int64, error) {
	res, err := s.DB.ExecContext(ctx,
		`DELETE FROM cache WHERE expires_at IS NOT NULL AND expires_at < now()`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
