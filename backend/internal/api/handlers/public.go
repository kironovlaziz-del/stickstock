package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

// PublicHandler serves dashboards via their share_token, with no auth —
// see DashboardHandler.CreateShareLink. The token itself is the
// credential (24 random bytes, hex-encoded), so anyone holding a valid
// link can view; nothing here checks a Supabase session.
type PublicHandler struct {
	DB *sql.DB
}

type publicDashboardResponse struct {
	Name    string           `json:"name"`
	Layout  json.RawMessage  `json:"layout"`
	Widgets []widgetResponse `json:"widgets"`
}

// GetDashboard: GET /api/public/dashboards/{token}
func (h *PublicHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")

	var resp publicDashboardResponse
	var dashboardID string
	var layoutBytes []byte
	if err := h.DB.QueryRowContext(r.Context(),
		`SELECT id, name, layout FROM dashboards WHERE share_token = $1`, token,
	).Scan(&dashboardID, &resp.Name, &layoutBytes); err != nil {
		writeJSONError(w, http.StatusNotFound, "shared dashboard not found")
		return
	}
	resp.Layout = layoutBytes

	rows, err := h.DB.QueryContext(r.Context(),
		`SELECT id, dashboard_id, saved_query_id, chart_type, config FROM dashboard_widgets WHERE dashboard_id = $1`,
		dashboardID,
	)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not load widgets")
		return
	}
	defer rows.Close()

	resp.Widgets = []widgetResponse{}
	for rows.Next() {
		var wdg widgetResponse
		var configBytes []byte
		if err := rows.Scan(&wdg.ID, &wdg.DashboardID, &wdg.SavedQueryID, &wdg.ChartType, &configBytes); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "could not read widgets")
			return
		}
		wdg.Config = configBytes
		resp.Widgets = append(resp.Widgets, wdg)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// RunWidget executes a shared dashboard's widget query server-side and
// returns just the rows — the underlying data source's DSN/credentials
// are never sent to the (anonymous) caller.
// POST /api/public/dashboards/{token}/widgets/{widget_id}/run
func (h *PublicHandler) RunWidget(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	widgetID := r.PathValue("widget_id")

	var savedQueryID string
	if err := h.DB.QueryRowContext(r.Context(),
		`SELECT dw.saved_query_id FROM dashboard_widgets dw
		 JOIN dashboards d ON d.id = dw.dashboard_id
		 WHERE dw.id = $1 AND d.share_token = $2`, widgetID, token,
	).Scan(&savedQueryID); err != nil {
		writeJSONError(w, http.StatusNotFound, "widget not found")
		return
	}

	var sqlText, dataSourceID string
	if err := h.DB.QueryRowContext(r.Context(),
		`SELECT sql_text, data_source_id FROM saved_queries WHERE id = $1`, savedQueryID,
	).Scan(&sqlText, &dataSourceID); err != nil {
		writeJSONError(w, http.StatusNotFound, "saved query not found")
		return
	}

	var kind, dsn string
	if err := h.DB.QueryRowContext(r.Context(),
		`SELECT kind, dsn FROM data_sources WHERE id = $1`, dataSourceID,
	).Scan(&kind, &dsn); err != nil {
		writeJSONError(w, http.StatusNotFound, "data source not found")
		return
	}

	result, err := execute(r.Context(), kind, dsn, sqlText, nil)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
