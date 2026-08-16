package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"stickstock/backend/internal/api/middleware"
	"stickstock/backend/internal/scheduler"
)

type ReportHandler struct {
	DB *sql.DB
}

var allowedDeliveryKinds = map[string]bool{"email": true, "telegram": true}

type scheduledReportResponse struct {
	ID             string     `json:"id"`
	SavedQueryID   string     `json:"saved_query_id"`
	CronExpr       string     `json:"cron_expr"`
	DeliveryKind   string     `json:"delivery_kind"`
	DeliveryTarget string     `json:"delivery_target"`
	IsActive       bool       `json:"is_active"`
	LastRunAt      *time.Time `json:"last_run_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

type createReportRequest struct {
	SavedQueryID   string `json:"saved_query_id"`
	CronExpr       string `json:"cron_expr"`
	DeliveryKind   string `json:"delivery_kind"`
	DeliveryTarget string `json:"delivery_target"`
}

func (h *ReportHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())

	var req createReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil ||
		req.SavedQueryID == "" || req.CronExpr == "" || req.DeliveryTarget == "" {
		writeJSONError(w, http.StatusBadRequest, "saved_query_id, cron_expr and delivery_target are required")
		return
	}
	if !allowedDeliveryKinds[req.DeliveryKind] {
		writeJSONError(w, http.StatusBadRequest, `delivery_kind must be "email" or "telegram"`)
		return
	}
	if _, err := scheduler.ParseCron(req.CronExpr); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid cron_expr: "+err.Error())
		return
	}

	// Confirm the saved query belongs to this user before scheduling it.
	var queryOwner string
	if err := h.DB.QueryRowContext(r.Context(),
		`SELECT owner_id FROM saved_queries WHERE id = $1`, req.SavedQueryID,
	).Scan(&queryOwner); err != nil || queryOwner != userID {
		writeJSONError(w, http.StatusNotFound, "saved query not found")
		return
	}

	var resp scheduledReportResponse
	err := h.DB.QueryRowContext(r.Context(),
		`INSERT INTO scheduled_reports (owner_id, saved_query_id, cron_expr, delivery_kind, delivery_target)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, saved_query_id, cron_expr, delivery_kind, delivery_target, is_active, created_at`,
		userID, req.SavedQueryID, req.CronExpr, req.DeliveryKind, req.DeliveryTarget,
	).Scan(&resp.ID, &resp.SavedQueryID, &resp.CronExpr, &resp.DeliveryKind, &resp.DeliveryTarget, &resp.IsActive, &resp.CreatedAt)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not create scheduled report")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (h *ReportHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())

	rows, err := h.DB.QueryContext(r.Context(),
		`SELECT id, saved_query_id, cron_expr, delivery_kind, delivery_target, is_active, last_run_at, created_at
		 FROM scheduled_reports WHERE owner_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not list scheduled reports")
		return
	}
	defer rows.Close()

	out := []scheduledReportResponse{}
	for rows.Next() {
		var rep scheduledReportResponse
		var lastRun sql.NullTime
		if err := rows.Scan(&rep.ID, &rep.SavedQueryID, &rep.CronExpr, &rep.DeliveryKind, &rep.DeliveryTarget, &rep.IsActive, &lastRun, &rep.CreatedAt); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "could not read scheduled reports")
			return
		}
		if lastRun.Valid {
			rep.LastRunAt = &lastRun.Time
		}
		out = append(out, rep)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

type updateReportRequest struct {
	CronExpr       string `json:"cron_expr"`
	DeliveryKind   string `json:"delivery_kind"`
	DeliveryTarget string `json:"delivery_target"`
	IsActive       *bool  `json:"is_active"`
}

func (h *ReportHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	id := r.PathValue("id")

	var req updateReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.CronExpr != "" {
		if _, err := scheduler.ParseCron(req.CronExpr); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid cron_expr: "+err.Error())
			return
		}
	}
	if req.DeliveryKind != "" && !allowedDeliveryKinds[req.DeliveryKind] {
		writeJSONError(w, http.StatusBadRequest, `delivery_kind must be "email" or "telegram"`)
		return
	}

	setClauses := []string{}
	args := []interface{}{}
	argN := 1
	if req.CronExpr != "" {
		setClauses = append(setClauses, fmt.Sprintf("cron_expr = $%d", argN))
		args = append(args, req.CronExpr)
		argN++
	}
	if req.DeliveryKind != "" {
		setClauses = append(setClauses, fmt.Sprintf("delivery_kind = $%d", argN))
		args = append(args, req.DeliveryKind)
		argN++
	}
	if req.DeliveryTarget != "" {
		setClauses = append(setClauses, fmt.Sprintf("delivery_target = $%d", argN))
		args = append(args, req.DeliveryTarget)
		argN++
	}
	if req.IsActive != nil {
		setClauses = append(setClauses, fmt.Sprintf("is_active = $%d", argN))
		args = append(args, *req.IsActive)
		argN++
	}
	if len(setClauses) == 0 {
		writeJSONError(w, http.StatusBadRequest, "no fields to update")
		return
	}
	args = append(args, id, userID)

	stmt := fmt.Sprintf(
		"UPDATE scheduled_reports SET %s WHERE id = $%d AND owner_id = $%d",
		strings.Join(setClauses, ", "), argN, argN+1,
	)
	res, err := h.DB.ExecContext(r.Context(), stmt, args...)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not update scheduled report")
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSONError(w, http.StatusNotFound, "scheduled report not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ReportHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	id := r.PathValue("id")

	res, err := h.DB.ExecContext(r.Context(),
		`DELETE FROM scheduled_reports WHERE id = $1 AND owner_id = $2`, id, userID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not delete scheduled report")
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSONError(w, http.StatusNotFound, "scheduled report not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
