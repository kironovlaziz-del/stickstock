package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"stickstock/backend/internal/api/middleware"
	"stickstock/backend/internal/connectors"
)

type MetricHandler struct {
	DB             *sql.DB
	SemanticURL    string
}

// RunMetric executes the metric expression and returns the result.
// GET /api/semantic/metrics/{id}/run
func (h *MetricHandler) RunMetric(w http.ResponseWriter, r *http.Request) {
	metricID := r.PathValue("id")
	userID, _ := middleware.UserIDFromContext(r.Context())

	// 1. Получаем метрику из semantic-layer
	url := h.SemanticURL + "/metrics/" + metricID
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "semantic service unreachable")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		writeJSONError(w, http.StatusNotFound, "metric not found")
		return
	}

	var metric struct {
		ID           string `json:"id"`
		Expression   string `json:"expression"`
		TableName    string `json:"table_name"`
		DataSourceID string `json:"data_source_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&metric); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "invalid metric data")
		return
	}

	// 2. Проверяем, что источник принадлежит пользователю
	kind, dsn, _, err := dataSourceForOwner(r.Context(), h.DB, metric.DataSourceID, userID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "data source not found")
		return
	}

	// 3. Строим запрос
	query := fmt.Sprintf("SELECT %s FROM %s", metric.Expression, metric.TableName)
	log.Printf("Executing metric query: %s", query)

	// 4. Выполняем запрос
	conn, err := connectors.New(kind, dsn)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer conn.Close()

	result, err := conn.Query(r.Context(), query)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "query failed: "+err.Error())
		return
	}

	// 5. Извлекаем первое значение (для KPI)
	if len(result.Rows) == 0 || len(result.Rows[0]) == 0 {
		writeJSONError(w, http.StatusNotFound, "no data returned")
		return
	}
	value := result.Rows[0][0]

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"value": value,
	})
}