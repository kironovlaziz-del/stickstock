package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"stickstock/backend/internal/api/middleware"
)

type CommentHandler struct {
	DB *sql.DB
}

type commentResponse struct {
	ID        string    `json:"id"`
	WidgetID  string    `json:"widget_id"`
	AuthorID  string    `json:"author_id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

// List: GET /api/dashboards/{id}/widgets/{widget_id}/comments — any role
// (owner/editor/viewer) with access to the dashboard can read.
func (h *CommentHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	dashboardID := r.PathValue("id")
	widgetID := r.PathValue("widget_id")

	if _, err := dashboardRole(r.Context(), h.DB, dashboardID, userID); err != nil {
		writeJSONError(w, http.StatusNotFound, "dashboard not found")
		return
	}

	rows, err := h.DB.QueryContext(r.Context(),
		`SELECT id, widget_id, author_id, body, created_at FROM widget_comments
		 WHERE widget_id = $1 ORDER BY created_at ASC`, widgetID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not list comments")
		return
	}
	defer rows.Close()

	out := []commentResponse{}
	for rows.Next() {
		var c commentResponse
		if err := rows.Scan(&c.ID, &c.WidgetID, &c.AuthorID, &c.Body, &c.CreatedAt); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "could not read comments")
			return
		}
		out = append(out, c)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

type createCommentRequest struct {
	Body string `json:"body"`
}

// Create: POST /api/dashboards/{id}/widgets/{widget_id}/comments — any
// role can comment, including viewers (discussion shouldn't require edit
// access).
func (h *CommentHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	dashboardID := r.PathValue("id")
	widgetID := r.PathValue("widget_id")

	if _, err := dashboardRole(r.Context(), h.DB, dashboardID, userID); err != nil {
		writeJSONError(w, http.StatusNotFound, "dashboard not found")
		return
	}

	var req createCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Body == "" {
		writeJSONError(w, http.StatusBadRequest, "body is required")
		return
	}

	var resp commentResponse
	err := h.DB.QueryRowContext(r.Context(),
		`INSERT INTO widget_comments (widget_id, author_id, body) VALUES ($1, $2, $3)
		 RETURNING id, widget_id, author_id, body, created_at`,
		widgetID, userID, req.Body,
	).Scan(&resp.ID, &resp.WidgetID, &resp.AuthorID, &resp.Body, &resp.CreatedAt)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not create comment")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// Delete: DELETE /api/comments/{comment_id} — only the comment's own
// author, for now. Letting the dashboard owner moderate others' comments
// too is a reasonable follow-up (needs comment -> widget -> dashboard ->
// owner lookup) but nobody's asked for moderation yet.
func (h *CommentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	commentID := r.PathValue("comment_id")

	res, err := h.DB.ExecContext(r.Context(),
		`DELETE FROM widget_comments WHERE id = $1 AND author_id = $2`, commentID, userID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not delete comment")
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSONError(w, http.StatusNotFound, "comment not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
