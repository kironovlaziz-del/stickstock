package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

type AdminHandler struct {
	DB *sql.DB
}

type adminUserResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Locale    string    `json:"locale"`
	IsAdmin   bool      `json:"is_admin"`
	IsBlocked bool      `json:"is_blocked"`
	CreatedAt time.Time `json:"created_at"`
}

// ListUsers: GET /api/admin/users — joins into Supabase's own auth.users
// for email, since profiles deliberately doesn't duplicate that (see
// migrations/0001_init.sql). Works because the Go backend connects with
// the service-role connection string, which can read the auth schema.
func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.QueryContext(r.Context(), `
		SELECT p.id, u.email, p.locale, p.is_admin, p.is_blocked, p.created_at
		FROM profiles p
		JOIN auth.users u ON u.id = p.id
		ORDER BY p.created_at DESC`)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not list users: "+err.Error())
		return
	}
	defer rows.Close()

	out := []adminUserResponse{}
	for rows.Next() {
		var u adminUserResponse
		if err := rows.Scan(&u.ID, &u.Email, &u.Locale, &u.IsAdmin, &u.IsBlocked, &u.CreatedAt); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "could not read users")
			return
		}
		out = append(out, u)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

type updateUserRequest struct {
	IsAdmin   *bool `json:"is_admin"`
	IsBlocked *bool `json:"is_blocked"`
}

// UpdateUser: PUT /api/admin/users/{id} — block/unblock, grant/revoke admin.
func (h *AdminHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.IsAdmin == nil && req.IsBlocked == nil {
		writeJSONError(w, http.StatusBadRequest, "is_admin or is_blocked is required")
		return
	}

	if req.IsAdmin != nil {
		if _, err := h.DB.ExecContext(r.Context(),
			`UPDATE profiles SET is_admin = $1 WHERE id = $2`, *req.IsAdmin, id,
		); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "could not update user")
			return
		}
	}
	if req.IsBlocked != nil {
		if _, err := h.DB.ExecContext(r.Context(),
			`UPDATE profiles SET is_blocked = $1 WHERE id = $2`, *req.IsBlocked, id,
		); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "could not update user")
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

type systemStats struct {
	Users        int `json:"users"`
	DataSources  int `json:"data_sources"`
	SavedQueries int `json:"saved_queries"`
	Dashboards   int `json:"dashboards"`
}

// Stats: GET /api/admin/stats — coarse counts across the whole system,
// not scoped to the admin's own data (unlike every other list endpoint).
// Table names are hardcoded from a fixed local list below, never from
// request input, so the string-built query is not an injection risk.
func (h *AdminHandler) Stats(w http.ResponseWriter, r *http.Request) {
	var stats systemStats
	targets := []struct {
		table string
		dest  *int
	}{
		{"profiles", &stats.Users},
		{"data_sources", &stats.DataSources},
		{"saved_queries", &stats.SavedQueries},
		{"dashboards", &stats.Dashboards},
	}

	for _, t := range targets {
		if err := h.DB.QueryRowContext(r.Context(), "SELECT count(*) FROM "+t.table).Scan(t.dest); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "could not compute stats")
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
