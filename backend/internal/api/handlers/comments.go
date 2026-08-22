package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
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

// validateWidgetOwnership проверяет, что widget_id существует и принадлежит dashboard_id.
func validateWidgetOwnership(ctx context.Context, db *sql.DB, dashboardID, widgetID string) error {
	var exists bool
	err := db.QueryRowContext(ctx,
		`SELECT EXISTS (
			SELECT 1 FROM dashboard_widgets
			WHERE id = $1 AND dashboard_id = $2
		)`, widgetID, dashboardID,
	).Scan(&exists)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	if !exists {
		return fmt.Errorf("widget not found or does not belong to this dashboard")
	}
	return nil
}

// List: GET /api/dashboards/{id}/widgets/{widget_id}/comments
func (h *CommentHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	dashboardID := r.PathValue("id")
	widgetID := r.PathValue("widget_id")

	if _, err := dashboardRole(r.Context(), h.DB, dashboardID, userID); err != nil {
		writeJSONError(w, http.StatusNotFound, "dashboard not found")
		return
	}

	if err := validateWidgetOwnership(r.Context(), h.DB, dashboardID, widgetID); err != nil {
		writeJSONError(w, http.StatusNotFound, "widget not found")
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

// Create: POST /api/dashboards/{id}/widgets/{widget_id}/comments
func (h *CommentHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	dashboardID := r.PathValue("id")
	widgetID := r.PathValue("widget_id")

	if _, err := dashboardRole(r.Context(), h.DB, dashboardID, userID); err != nil {
		writeJSONError(w, http.StatusNotFound, "dashboard not found")
		return
	}

	if err := validateWidgetOwnership(r.Context(), h.DB, dashboardID, widgetID); err != nil {
		writeJSONError(w, http.StatusNotFound, "widget not found")
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

// Delete: DELETE /api/comments/{comment_id}
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