package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"stickstock/backend/internal/api/middleware"
	"stickstock/backend/internal/i18n"
)

// ProfileHandler serves the "public.profiles" row that migrations/0001_init.sql
// auto-creates for every Supabase Auth user (via the on_auth_user_created
// trigger). Credentials, email, and OAuth identities live in Supabase's own
// auth.users and are never touched here — this is just the app-specific
// bits (currently just locale).
type ProfileHandler struct {
	DB *sql.DB
}

type profileResponse struct {
	ID        string    `json:"id"`
	Locale    string    `json:"locale"`
	CreatedAt time.Time `json:"created_at"`
}

func (h *ProfileHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())

	var resp profileResponse
	err := h.DB.QueryRowContext(r.Context(),
		`SELECT id, locale, created_at FROM profiles WHERE id = $1`, userID,
	).Scan(&resp.ID, &resp.Locale, &resp.CreatedAt)
	if err != nil {
		// A signed-in user with no profile row means the on_auth_user_created
		// trigger didn't fire (e.g. the migration was applied after they
		// signed up) — surface that plainly rather than a generic 404.
		writeJSONError(w, http.StatusNotFound, "profile not found — was migrations/0001_init.sql applied after this user signed up?")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

type updateProfileRequest struct {
	Locale string `json:"locale"`
}

func (h *ProfileHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())

	var req updateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Locale == "" {
		writeJSONError(w, http.StatusBadRequest, "locale is required")
		return
	}
	if !i18n.IsSupported(req.Locale) {
		writeJSONError(w, http.StatusBadRequest, "unsupported locale")
		return
	}

	res, err := h.DB.ExecContext(r.Context(),
		`UPDATE profiles SET locale = $1 WHERE id = $2`, req.Locale, userID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not update profile")
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSONError(w, http.StatusNotFound, "profile not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
