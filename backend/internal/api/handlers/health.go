package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

func Health(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := "ok"
		code := http.StatusOK

		if err := db.PingContext(r.Context()); err != nil {
			status = "degraded"
			code = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  status,
			"service": "stickstock-api",
		})
	}
}
