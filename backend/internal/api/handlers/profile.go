package handlers

import (
	"strings"
	"log"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"stickstock/backend/internal/api/middleware"
	"stickstock/backend/internal/connectors"
)

type ProfileHandler struct {
	DB             *sql.DB
	ProfilerURL    string
}

type ProfileRequest struct {
	Table     string `json:"table"`
	SampleSize int   `json:"sample_size,omitempty"`
}

type ProfilerResponse struct {
	Columns   []ColumnProfile `json:"columns"`
	TotalRows int             `json:"total_rows"`
}

type ColumnProfile struct {
	Name             string                 `json:"name"`
	DataType         string                 `json:"data_type"`
	NullCount        int                    `json:"null_count"`
	NullPercentage   float64                `json:"null_percentage"`
	UniqueCount      int                    `json:"unique_count"`
	UniquePercentage float64                `json:"unique_percentage"`
	Min              interface{}            `json:"min,omitempty"`
	Max              interface{}            `json:"max,omitempty"`
	Mean             float64                `json:"mean,omitempty"`
	Std              float64                `json:"std,omitempty"`
	Quantiles        map[string]float64     `json:"quantiles,omitempty"`
	TopValues        map[string]int         `json:"top_values,omitempty"`
}

func (h *ProfileHandler) ProfileDataSource(w http.ResponseWriter, r *http.Request) {
	log.Printf("ProfileDataSource called: id=%s", r.PathValue("id"))
	userID, _ := middleware.UserIDFromContext(r.Context())
	id := r.PathValue("id")

	var req ProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.Table == "" {
		writeJSONError(w, http.StatusBadRequest, "table is required")
		return
	}
	if req.SampleSize <= 0 || req.SampleSize > 10000 {
		req.SampleSize = 1000
	}

	kind, dsn, _, err := dataSourceForOwner(r.Context(), h.DB, id, userID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "data source not found")
		return
	}

	conn, err := connectors.New(kind, dsn)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer conn.Close()

	// Проверяем, что таблица существует
	tables, err := conn.ListSchemas(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "could not list schema: "+err.Error())
		return
	}
	var found bool
	for _, t := range tables {
		if t.Name == req.Table {
			found = true
			break
		}
	}
	if !found {
		writeJSONError(w, http.StatusNotFound, "table not found in this data source")
		return
	}

	// Выполняем запрос к таблице
		tableName := req.Table
	if !strings.Contains(tableName, ".") {
		tableName = "demo." + tableName
	}
	query := fmt.Sprintf("SELECT * FROM %s LIMIT %d", tableName, req.SampleSize)
	result, err := conn.Query(r.Context(), query)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "query failed: "+err.Error())
		return
	}

	// Подготавливаем данные для профилировщика
	profilerReq := struct {
		Columns []string        `json:"columns"`
		Rows    [][]interface{} `json:"rows"`
	}{
		Columns: result.Columns,
		Rows:    result.Rows,
	}
	body, _ := json.Marshal(profilerReq)

	// Вызываем data-profiler
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(h.ProfilerURL+"/profile", "application/json", bytes.NewReader(body))
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "profiler service unreachable: "+err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]string
		json.NewDecoder(resp.Body).Decode(&errResp)
		writeJSONError(w, http.StatusBadGateway, "profiler error: "+errResp["error"])
		return
	}

	var profilerResp ProfilerResponse
	if err := json.NewDecoder(resp.Body).Decode(&profilerResp); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "invalid response from profiler")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profilerResp)
}
