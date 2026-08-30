package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"stickstock/backend/internal/api/middleware"
	"stickstock/backend/internal/export"
)

const MAX_EXPORT_ROWS = 50000

type ExportHandler struct {
	DB *sql.DB
}

var exportContentTypes = map[string]string{
	"csv":  "text/csv",
	"xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	"pdf":  "application/pdf",
}

func sanitizeFilename(name string) string {
	replacer := strings.NewReplacer("/", "-", "\\", "-", `"`, "", "\n", " ", "\r", " ")
	s := strings.TrimSpace(replacer.Replace(name))
	if s == "" {
		return "export"
	}
	return s
}

func streamExport(w http.ResponseWriter, format, filenameBase, title string, result *runResponse) {
	if len(result.Rows) > MAX_EXPORT_ROWS {
		result.Rows = result.Rows[:MAX_EXPORT_ROWS]
	}
	contentType, ok := exportContentTypes[format]
	if !ok {
		writeJSONError(w, http.StatusBadRequest, `format must be "csv", "xlsx", or "pdf"`)
		return
	}

	filename := fmt.Sprintf("%s.%s", sanitizeFilename(filenameBase), format)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	switch format {
	case "csv":
		_ = export.ToCSV(w, result.Columns, result.Rows)
	case "xlsx":
		_ = export.ToXLSX(w, result.Columns, result.Rows)
	case "pdf":
		_ = export.ToPDF(w, title, result.Columns, result.Rows)
	}
}

func (h *ExportHandler) ExportSaved(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	id := r.PathValue("id")
	format := r.URL.Query().Get("format")

	var name, dataSourceID, sqlText string
	if err := h.DB.QueryRowContext(r.Context(),
		`SELECT name, data_source_id, sql_text FROM saved_queries WHERE id = $1 AND owner_id = $2`, id, userID,
	).Scan(&name, &dataSourceID, &sqlText); err != nil {
		writeJSONError(w, http.StatusNotFound, "query not found")
		return
	}

	kind, dsn, _, err := dataSourceForOwner(r.Context(), h.DB, dataSourceID, userID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "data source not found")
		return
	}

	result, err := execute(r.Context(), kind, dsn, sqlText, nil)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	streamExport(w, format, name, name, result)
}

type exportAdHocRequest struct {
	DataSourceID string `json:"data_source_id"`
	SQL          string `json:"sql"`
	Format       string `json:"format"`
	Title        string `json:"title"`
}

func (h *ExportHandler) ExportAdHoc(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())

	var req exportAdHocRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.DataSourceID == "" || req.SQL == "" {
		writeJSONError(w, http.StatusBadRequest, "data_source_id and sql are required")
		return
	}

	kind, dsn, fileTable, err := dataSourceForOwner(r.Context(), h.DB, req.DataSourceID, userID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "data source not found")
		return
	}
	if err := validateQueryText(kind, fileTable, req.SQL); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := execute(r.Context(), kind, dsn, req.SQL, nil)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	title := req.Title
	if title == "" {
		title = "Query export"
	}
	streamExport(w, req.Format, title, title, result)
}
