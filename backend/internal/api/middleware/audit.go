package middleware

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// AuditMiddleware logs each authenticated request to the audit_log table.
func AuditMiddleware(db *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := &responseWriterWrapper{ResponseWriter: w, statusCode: http.StatusOK}
			start := time.Now()

			next.ServeHTTP(ww, r)

			userID, ok := UserIDFromContext(r.Context())
			if !ok || userID == "" {
				return
			}

			entry := AuditLogEntry{
				UserID:     userID,
				Action:     r.Method + " " + r.URL.Path,
				Resource:   extractResource(r.URL.Path),
				ResourceID: extractResourceID(r.URL.Path),
				IP:         r.RemoteAddr,
				UserAgent:  r.UserAgent(),
				CreatedAt:  time.Now(),
			}

			details := map[string]interface{}{
				"status_code": ww.statusCode,
				"duration_ms": time.Since(start).Milliseconds(),
				"method":      r.Method,
				"path":        r.URL.Path,
			}
			if ww.statusCode >= 400 {
				details["error"] = http.StatusText(ww.statusCode)
			}
			detailsJSON, _ := json.Marshal(details)
			entry.Details = detailsJSON

			go func(e AuditLogEntry) {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_, err := db.ExecContext(ctx,
					`INSERT INTO audit_log (user_id, action, resource, resource_id, ip_address, user_agent, details)
					 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
					e.UserID, e.Action, e.Resource, e.ResourceID, e.IP, e.UserAgent, e.Details,
				)
				if err != nil {
					log.Printf("Audit log insert error: %v", err)
				}
			}(entry)
		})
	}
}

type AuditLogEntry struct {
	UserID     string          `json:"user_id"`
	Action     string          `json:"action"`
	Resource   string          `json:"resource"`
	ResourceID string          `json:"resource_id"`
	IP         string          `json:"ip_address"`
	UserAgent  string          `json:"user_agent"`
	Details    json.RawMessage `json:"details"`
	CreatedAt  time.Time       `json:"created_at"`
}

type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (ww *responseWriterWrapper) WriteHeader(code int) {
	ww.statusCode = code
	ww.ResponseWriter.WriteHeader(code)
}

func (ww *responseWriterWrapper) Write(b []byte) (int, error) {
	if ww.statusCode == 0 {
		ww.statusCode = http.StatusOK
	}
	return ww.ResponseWriter.Write(b)
}

func extractResource(path string) string {
	parts := splitPath(path)
	if len(parts) >= 3 && parts[1] == "api" {
		return parts[2]
	}
	return "unknown"
}

func extractResourceID(path string) string {
	parts := splitPath(path)
	if len(parts) >= 4 {
		return parts[3]
	}
	return ""
}

func splitPath(path string) []string {
	var parts []string
	start := 0
	for i, c := range path {
		if c == '/' {
			if i > start {
				parts = append(parts, path[start:i])
			}
			start = i + 1
		}
	}
	if start < len(path) {
		parts = append(parts, path[start:])
	}
	return parts
}
