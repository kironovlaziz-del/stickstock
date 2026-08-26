package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"stickstock/backend/internal/api/middleware"
)

type AnalyticsTaskHandler struct {
	DB                  *sql.DB
	AnalyticsServiceURL string
}

type TaskResponse struct {
	ID          string          `json:"id"`
	Type        string          `json:"type"`
	Status      string          `json:"status"` // pending, running, completed, failed
	CreatedAt   time.Time       `json:"created_at"`
	StartedAt   *time.Time      `json:"started_at,omitempty"`
	CompletedAt *time.Time      `json:"completed_at,omitempty"`
	Output      json.RawMessage `json:"output,omitempty"`
	Error       string          `json:"error,omitempty"`
}

type CreateTaskRequest struct {
	Type  string          `json:"type"`
	Input json.RawMessage `json:"input"`
}

func (h *AnalyticsTaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())

	var req CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Type == "" {
		writeJSONError(w, http.StatusBadRequest, "type and input are required")
		return
	}

	var id string
	err := h.DB.QueryRowContext(r.Context(),
		`INSERT INTO analytics_tasks (user_id, type, input, status)
		 VALUES ($1, $2, $3, 'pending') RETURNING id`,
		userID, req.Type, req.Input,
	).Scan(&id)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not create task")
		return
	}

	go h.processTask(id)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"task_id": id})
}

func (h *AnalyticsTaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	id := r.PathValue("id")

	var resp TaskResponse
	var output, inputBytes []byte
	var startedAt, completedAt sql.NullTime

	err := h.DB.QueryRowContext(r.Context(),
		`SELECT id, type, status, input, output, error, created_at, started_at, completed_at
		 FROM analytics_tasks WHERE id = $1 AND user_id = $2`,
		id, userID,
	).Scan(&resp.ID, &resp.Type, &resp.Status, &inputBytes, &output, &resp.Error,
		&resp.CreatedAt, &startedAt, &completedAt)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "task not found")
		return
	}
	if startedAt.Valid {
		resp.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		resp.CompletedAt = &completedAt.Time
	}
	resp.Output = output

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *AnalyticsTaskHandler) processTask(taskID string) {
	_, _ = h.DB.Exec(`UPDATE analytics_tasks SET status = 'running', started_at = now() WHERE id = $1`, taskID)

	var taskType string
	var input json.RawMessage
	err := h.DB.QueryRow(`SELECT type, input FROM analytics_tasks WHERE id = $1`, taskID).Scan(&taskType, &input)
	if err != nil {
		_, _ = h.DB.Exec(`UPDATE analytics_tasks SET status = 'failed', error = $1, completed_at = now() WHERE id = $2`, "task data not found", taskID)
		return
	}

	result, err := h.callAnalyticsService(taskType, input)
	if err != nil {
		_, _ = h.DB.Exec(`UPDATE analytics_tasks SET status = 'failed', error = $1, completed_at = now() WHERE id = $2`, err.Error(), taskID)
		return
	}

	_, _ = h.DB.Exec(`UPDATE analytics_tasks SET status = 'completed', output = $1, completed_at = now() WHERE id = $2`, result, taskID)
}

func (h *AnalyticsTaskHandler) callAnalyticsService(taskType string, input json.RawMessage) (json.RawMessage, error) {
	var path string
	switch taskType {
	case "regression":
		path = "/api/analytics/stats/regression"
	case "forecast":
		path = "/api/analytics/stats/forecast"
	case "anomalies":
		path = "/api/analytics/anomalies"
	case "ttest":
		path = "/api/analytics/stats/ttest"
	default:
		return nil, fmt.Errorf("unsupported task type: %s", taskType)
	}

	url := h.AnalyticsServiceURL + path
	if h.AnalyticsServiceURL == "" {
		url = "http://analytics-service:8000" + path
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(input))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("analytics service request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("analytics service returned %d: %s", resp.StatusCode, string(body))
	}

	return json.RawMessage(body), nil
}
