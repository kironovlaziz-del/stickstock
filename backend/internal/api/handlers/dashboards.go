package handlers

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"stickstock/backend/internal/api/middleware"
	"log"
)

var allowedChartTypes = map[string]bool{
	"line": true, "bar": true, "pie": true, "scatter": true,
	"table": true, "heatmap": true, "boxplot": true, "treemap": true,
	"kpi": true, "forecast": true,
	"combo": true, "radar": true, "funnel": true, "waterfall": true,
	"bubble": true, "stackedbar": true,
}

type DashboardHandler struct {
	DB *sql.DB
}

type dashboardSummary struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type widgetResponse struct {
	ID           string          `json:"id"`
	DashboardID  string          `json:"dashboard_id"`
	SavedQueryID string          `json:"saved_query_id,omitempty"`
	MetricID     string          `json:"metric_id,omitempty"`
	ChartType    string          `json:"chart_type"`
	Config       json.RawMessage `json:"config"`
}

type dashboardDetailResponse struct {
	ID           string           `json:"id"`
	Name         string           `json:"name"`
	Layout       json.RawMessage  `json:"layout"`
	Role         string           `json:"role"`
	ShareEnabled bool             `json:"share_enabled"`
	CreatedAt    time.Time        `json:"created_at"`
	Widgets      []widgetResponse `json:"widgets"`
}

func dashboardRole(ctx context.Context, db *sql.DB, dashboardID, userID string) (string, error) {
	var ownerID string
	if err := db.QueryRowContext(ctx,
		`SELECT owner_id FROM dashboards WHERE id = $1`, dashboardID,
	).Scan(&ownerID); err != nil {
		return "", err
	}
	if ownerID == userID {
		return "owner", nil
	}

	var role string
	if err := db.QueryRowContext(ctx,
		`SELECT role FROM dashboard_collaborators WHERE dashboard_id = $1 AND user_id = $2`,
		dashboardID, userID,
	).Scan(&role); err != nil {
		return "", fmt.Errorf("no access")
	}
	return role, nil
}

func canEdit(role string) bool { return role == "owner" || role == "editor" }

type createDashboardRequest struct {
	Name   string          `json:"name"`
	Layout json.RawMessage `json:"layout"`
}

func (h *DashboardHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())

	var req createDashboardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeJSONError(w, http.StatusBadRequest, "name is required")
		return
	}
	layout := req.Layout
	if len(layout) == 0 {
		layout = json.RawMessage("[]")
	}

	var resp dashboardDetailResponse
	var layoutBytes []byte
	err := h.DB.QueryRowContext(r.Context(),
		`INSERT INTO dashboards (owner_id, name, layout) VALUES ($1, $2, $3)
		 RETURNING id, name, layout, created_at`,
		userID, req.Name, string(layout),
	).Scan(&resp.ID, &resp.Name, &layoutBytes, &resp.CreatedAt)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not create dashboard")
		return
	}
	resp.Layout = layoutBytes
	resp.Role = "owner"
	resp.Widgets = []widgetResponse{}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (h *DashboardHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())

	rows, err := h.DB.QueryContext(r.Context(), `
		SELECT id, name, role, created_at FROM (
			SELECT id, name, 'owner' AS role, created_at FROM dashboards WHERE owner_id = $1
			UNION
			SELECT d.id, d.name, dc.role, d.created_at
			FROM dashboards d
			JOIN dashboard_collaborators dc ON dc.dashboard_id = d.id
			WHERE dc.user_id = $1
		) combined
		ORDER BY created_at DESC`, userID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not list dashboards")
		return
	}
	defer rows.Close()

	out := []dashboardSummary{}
	for rows.Next() {
		var d dashboardSummary
		if err := rows.Scan(&d.ID, &d.Name, &d.Role, &d.CreatedAt); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "could not read dashboards")
			return
		}
		out = append(out, d)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (h *DashboardHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	id := r.PathValue("id")
	log.Printf("GetDashboard: userID=%s, id=%s", userID, id)

	role, err := dashboardRole(r.Context(), h.DB, id, userID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "dashboard not found")
		return
	}

	var resp dashboardDetailResponse
	var layoutBytes []byte
	var shareToken sql.NullString
	if err := h.DB.QueryRowContext(r.Context(),
		`SELECT id, name, layout, share_token, created_at FROM dashboards WHERE id = $1`, id,
	).Scan(&resp.ID, &resp.Name, &layoutBytes, &shareToken, &resp.CreatedAt); err != nil {
		writeJSONError(w, http.StatusNotFound, "dashboard not found")
		return
	}
	resp.Layout = layoutBytes
	resp.Role = role
	resp.ShareEnabled = shareToken.Valid

	rows, err := h.DB.QueryContext(r.Context(),
		`SELECT id, dashboard_id, saved_query_id, metric_id, chart_type, config
		 FROM dashboard_widgets WHERE dashboard_id = $1`, id)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not load widgets")
		return
	}
	defer rows.Close()

	widgets := []widgetResponse{}
	for rows.Next() {
		var wdg widgetResponse
		var configBytes []byte
		var sqID, mID sql.NullString
		if err := rows.Scan(&wdg.ID, &wdg.DashboardID, &sqID, &mID, &wdg.ChartType, &configBytes); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "could not read widgets")
			return
		}
		if sqID.Valid {
			wdg.SavedQueryID = sqID.String
		}
		if mID.Valid {
			wdg.MetricID = mID.String
		}
		wdg.Config = configBytes
		widgets = append(widgets, wdg)
	}
	resp.Widgets = widgets

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

type updateDashboardRequest struct {
	Name   string          `json:"name"`
	Layout json.RawMessage `json:"layout"`
}

func (h *DashboardHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	id := r.PathValue("id")

	role, err := dashboardRole(r.Context(), h.DB, id, userID)
	if err != nil || !canEdit(role) {
		writeJSONError(w, http.StatusNotFound, "dashboard not found")
		return
	}

	var req updateDashboardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" && len(req.Layout) == 0 {
		writeJSONError(w, http.StatusBadRequest, "name or layout is required")
		return
	}

	setClauses := []string{}
	args := []interface{}{}
	argN := 1
	if req.Name != "" {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argN))
		args = append(args, req.Name)
		argN++
	}
	if len(req.Layout) > 0 {
		setClauses = append(setClauses, fmt.Sprintf("layout = $%d", argN))
		args = append(args, string(req.Layout))
		argN++
	}
	args = append(args, id)

	stmt := fmt.Sprintf("UPDATE dashboards SET %s WHERE id = $%d", strings.Join(setClauses, ", "), argN)
	if _, err := h.DB.ExecContext(r.Context(), stmt, args...); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not update dashboard")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *DashboardHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	id := r.PathValue("id")

	res, err := h.DB.ExecContext(r.Context(),
		`DELETE FROM dashboards WHERE id = $1 AND owner_id = $2`, id, userID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not delete dashboard")
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSONError(w, http.StatusNotFound, "dashboard not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type addWidgetRequest struct {
	SavedQueryID string          `json:"saved_query_id"`
	MetricID     string          `json:"metric_id"`
	ChartType    string          `json:"chart_type"`
	Config       json.RawMessage `json:"config"`
}

func (h *DashboardHandler) AddWidget(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	dashboardID := r.PathValue("id")

	role, err := dashboardRole(r.Context(), h.DB, dashboardID, userID)
	if err != nil || !canEdit(role) {
		writeJSONError(w, http.StatusNotFound, "dashboard not found")
		return
	}

	var req addWidgetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.ChartType == "" {
		writeJSONError(w, http.StatusBadRequest, "chart_type is required")
		return
	}
	if req.SavedQueryID == "" && req.MetricID == "" {
		writeJSONError(w, http.StatusBadRequest, "either saved_query_id or metric_id is required")
		return
	}
	if req.SavedQueryID != "" && req.MetricID != "" {
		writeJSONError(w, http.StatusBadRequest, "cannot provide both saved_query_id and metric_id")
		return
	}
	if !allowedChartTypes[req.ChartType] {
		writeJSONError(w, http.StatusBadRequest, "unsupported chart_type: "+req.ChartType)
		return
	}

	var dashboardOwner string
	if err := h.DB.QueryRowContext(r.Context(), `SELECT owner_id FROM dashboards WHERE id = $1`, dashboardID).Scan(&dashboardOwner); err != nil {
		writeJSONError(w, http.StatusNotFound, "dashboard not found")
		return
	}
	if req.SavedQueryID != "" {
		var queryOwner string
		if err := h.DB.QueryRowContext(r.Context(), `SELECT owner_id FROM saved_queries WHERE id = $1`, req.SavedQueryID).Scan(&queryOwner); err != nil || queryOwner != dashboardOwner {
			writeJSONError(w, http.StatusNotFound, "saved query not found")
			return
		}
	}

	config := req.Config
	if len(config) == 0 {
		config = json.RawMessage("{}")
	}

	var widgetID string
	var insertErr error
	if req.SavedQueryID != "" {
		insertErr = h.DB.QueryRowContext(r.Context(),
			`INSERT INTO dashboard_widgets (dashboard_id, saved_query_id, chart_type, config)
			 VALUES ($1, $2, $3, $4) RETURNING id`,
			dashboardID, req.SavedQueryID, req.ChartType, string(config),
		).Scan(&widgetID)
	} else {
		insertErr = h.DB.QueryRowContext(r.Context(),
			`INSERT INTO dashboard_widgets (dashboard_id, metric_id, chart_type, config)
			 VALUES ($1, $2, $3, $4) RETURNING id`,
			dashboardID, req.MetricID, req.ChartType, string(config),
		).Scan(&widgetID)
	}
	if insertErr != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not add widget: "+insertErr.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(widgetResponse{
		ID:           widgetID,
		DashboardID:  dashboardID,
		SavedQueryID: req.SavedQueryID,
		MetricID:     req.MetricID,
		ChartType:    req.ChartType,
		Config:       config,
	})
}

type updateWidgetRequest struct {
	SavedQueryID string          `json:"saved_query_id"`
	MetricID     string          `json:"metric_id"`
	ChartType    string          `json:"chart_type"`
	Config       json.RawMessage `json:"config"`
}

func (h *DashboardHandler) UpdateWidget(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	dashboardID := r.PathValue("id")
	widgetID := r.PathValue("widget_id")

	role, err := dashboardRole(r.Context(), h.DB, dashboardID, userID)
	if err != nil || !canEdit(role) {
		writeJSONError(w, http.StatusNotFound, "dashboard not found")
		return
	}

	var req updateWidgetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ChartType != "" && !allowedChartTypes[req.ChartType] {
		writeJSONError(w, http.StatusBadRequest, "unsupported chart_type")
		return
	}
	if req.SavedQueryID != "" && req.MetricID != "" {
		writeJSONError(w, http.StatusBadRequest, "cannot provide both saved_query_id and metric_id")
		return
	}
	if req.SavedQueryID == "" && req.MetricID == "" && req.ChartType == "" && len(req.Config) == 0 {
		writeJSONError(w, http.StatusBadRequest, "nothing to update")
		return
	}

	if req.SavedQueryID != "" {
		var dashboardOwner string
		if err := h.DB.QueryRowContext(r.Context(), `SELECT owner_id FROM dashboards WHERE id = $1`, dashboardID).Scan(&dashboardOwner); err != nil {
			writeJSONError(w, http.StatusNotFound, "dashboard not found")
			return
		}
		var queryOwner string
		if err := h.DB.QueryRowContext(r.Context(), `SELECT owner_id FROM saved_queries WHERE id = $1`, req.SavedQueryID).Scan(&queryOwner); err != nil || queryOwner != dashboardOwner {
			writeJSONError(w, http.StatusNotFound, "saved query not found")
			return
		}
	}

	setClauses := []string{}
	args := []interface{}{}
	argN := 1
	if req.ChartType != "" {
		setClauses = append(setClauses, fmt.Sprintf("chart_type = $%d", argN))
		args = append(args, req.ChartType)
		argN++
	}
	if len(req.Config) > 0 {
		setClauses = append(setClauses, fmt.Sprintf("config = $%d", argN))
		args = append(args, string(req.Config))
		argN++
	}
	if req.SavedQueryID != "" {
		setClauses = append(setClauses, fmt.Sprintf("saved_query_id = $%d, metric_id = NULL", argN))
		args = append(args, req.SavedQueryID)
		argN++
	}
	if req.MetricID != "" {
		setClauses = append(setClauses, fmt.Sprintf("metric_id = $%d, saved_query_id = NULL", argN))
		args = append(args, req.MetricID)
		argN++
	}
	if len(setClauses) == 0 {
		writeJSONError(w, http.StatusBadRequest, "nothing to update")
		return
	}
	args = append(args, widgetID, dashboardID)

	stmt := fmt.Sprintf(
		"UPDATE dashboard_widgets SET %s WHERE id = $%d AND dashboard_id = $%d",
		strings.Join(setClauses, ", "), argN, argN+1,
	)
	res, err := h.DB.ExecContext(r.Context(), stmt, args...)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not update widget")
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSONError(w, http.StatusNotFound, "widget not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *DashboardHandler) DeleteWidget(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	dashboardID := r.PathValue("id")
	widgetID := r.PathValue("widget_id")

	role, err := dashboardRole(r.Context(), h.DB, dashboardID, userID)
	if err != nil || !canEdit(role) {
		writeJSONError(w, http.StatusNotFound, "dashboard not found")
		return
	}

	res, err := h.DB.ExecContext(r.Context(),
		`DELETE FROM dashboard_widgets WHERE id = $1 AND dashboard_id = $2`, widgetID, dashboardID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not delete widget")
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSONError(w, http.StatusNotFound, "widget not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type collaboratorResponse struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
}

func (h *DashboardHandler) ListCollaborators(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	dashboardID := r.PathValue("id")

	if _, err := dashboardRole(r.Context(), h.DB, dashboardID, userID); err != nil {
		writeJSONError(w, http.StatusNotFound, "dashboard not found")
		return
	}

	rows, err := h.DB.QueryContext(r.Context(), `
		SELECT dc.user_id, u.email, dc.role
		FROM dashboard_collaborators dc
		JOIN auth.users u ON u.id = dc.user_id
		WHERE dc.dashboard_id = $1`, dashboardID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not list collaborators")
		return
	}
	defer rows.Close()

	out := []collaboratorResponse{}
	for rows.Next() {
		var c collaboratorResponse
		if err := rows.Scan(&c.UserID, &c.Email, &c.Role); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "could not read collaborators")
			return
		}
		out = append(out, c)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

type addCollaboratorRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

func (h *DashboardHandler) AddCollaborator(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	dashboardID := r.PathValue("id")

	role, err := dashboardRole(r.Context(), h.DB, dashboardID, userID)
	if err != nil || role != "owner" {
		writeJSONError(w, http.StatusForbidden, "only the owner can manage collaborators")
		return
	}

	var req addCollaboratorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" {
		writeJSONError(w, http.StatusBadRequest, "email is required")
		return
	}
	if req.Role != "editor" && req.Role != "viewer" {
		writeJSONError(w, http.StatusBadRequest, `role must be "editor" or "viewer"`)
		return
	}

	var collaboratorID string
	if err := h.DB.QueryRowContext(r.Context(),
		`SELECT id FROM auth.users WHERE email = $1`, req.Email,
	).Scan(&collaboratorID); err != nil {
		writeJSONError(w, http.StatusNotFound, "no user found with that email — they need to have signed up first")
		return
	}
	if collaboratorID == userID {
		writeJSONError(w, http.StatusBadRequest, "you already own this dashboard")
		return
	}

	if _, err := h.DB.ExecContext(r.Context(),
		`INSERT INTO dashboard_collaborators (dashboard_id, user_id, role) VALUES ($1, $2, $3)
		 ON CONFLICT (dashboard_id, user_id) DO UPDATE SET role = EXCLUDED.role`,
		dashboardID, collaboratorID, req.Role,
	); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not add collaborator")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *DashboardHandler) RemoveCollaborator(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	dashboardID := r.PathValue("id")
	collaboratorUserID := r.PathValue("user_id")

	role, err := dashboardRole(r.Context(), h.DB, dashboardID, userID)
	if err != nil || role != "owner" {
		writeJSONError(w, http.StatusForbidden, "only the owner can manage collaborators")
		return
	}

	if _, err := h.DB.ExecContext(r.Context(),
		`DELETE FROM dashboard_collaborators WHERE dashboard_id = $1 AND user_id = $2`,
		dashboardID, collaboratorUserID,
	); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not remove collaborator")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func randomShareToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (h *DashboardHandler) CreateShareLink(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	dashboardID := r.PathValue("id")

	role, err := dashboardRole(r.Context(), h.DB, dashboardID, userID)
	if err != nil || role != "owner" {
		writeJSONError(w, http.StatusForbidden, "only the owner can share this dashboard")
		return
	}

	token, err := randomShareToken()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not generate share token")
		return
	}

	if _, err := h.DB.ExecContext(r.Context(),
		`UPDATE dashboards SET share_token = $1 WHERE id = $2`, token, dashboardID,
	); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not enable sharing")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"share_token": token})
}

func (h *DashboardHandler) RevokeShareLink(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	dashboardID := r.PathValue("id")

	role, err := dashboardRole(r.Context(), h.DB, dashboardID, userID)
	if err != nil || role != "owner" {
		writeJSONError(w, http.StatusForbidden, "only the owner can manage sharing")
		return
	}

	if _, err := h.DB.ExecContext(r.Context(),
		`UPDATE dashboards SET share_token = NULL WHERE id = $1`, dashboardID,
	); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not revoke sharing")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
