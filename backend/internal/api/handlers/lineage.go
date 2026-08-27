package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"stickstock/backend/internal/api/middleware"
)

type LineageHandler struct {
	DB *sql.DB
}

type LineageNode struct {
	ID   string `json:"id"`
	Type string `json:"type"` // data_source, saved_query, dashboard
	Name string `json:"name"`
}

type LineageEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type LineageGraph struct {
	Nodes []LineageNode `json:"nodes"`
	Edges []LineageEdge `json:"edges"`
}

// Get returns the full lineage graph for the current user.
func (h *LineageHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	graph := LineageGraph{Nodes: []LineageNode{}, Edges: []LineageEdge{}}

	// 1. Data sources
	dsRows, err := h.DB.QueryContext(r.Context(),
		`SELECT id, name FROM data_sources WHERE owner_id = $1`, userID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not load data sources")
		return
	}
	for dsRows.Next() {
		var id, name string
		if err := dsRows.Scan(&id, &name); err != nil {
			dsRows.Close()
			writeJSONError(w, http.StatusInternalServerError, "could not read data source")
			return
		}
		graph.Nodes = append(graph.Nodes, LineageNode{ID: "ds:" + id, Type: "data_source", Name: name})
	}
	dsRows.Close()

	// 2. Saved queries
	qRows, err := h.DB.QueryContext(r.Context(),
		`SELECT id, name, data_source_id FROM saved_queries WHERE owner_id = $1`, userID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not load saved queries")
		return
	}
	for qRows.Next() {
		var id, name, dsID string
		if err := qRows.Scan(&id, &name, &dsID); err != nil {
			qRows.Close()
			writeJSONError(w, http.StatusInternalServerError, "could not read saved query")
			return
		}
		graph.Nodes = append(graph.Nodes, LineageNode{ID: "q:" + id, Type: "saved_query", Name: name})
		graph.Edges = append(graph.Edges, LineageEdge{From: "ds:" + dsID, To: "q:" + id})
	}
	qRows.Close()

	// 3. Dashboards
	dRows, err := h.DB.QueryContext(r.Context(),
		`SELECT id, name FROM dashboards WHERE owner_id = $1`, userID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not load dashboards")
		return
	}
	for dRows.Next() {
		var id, name string
		if err := dRows.Scan(&id, &name); err != nil {
			dRows.Close()
			writeJSONError(w, http.StatusInternalServerError, "could not read dashboard")
			return
		}
		graph.Nodes = append(graph.Nodes, LineageNode{ID: "d:" + id, Type: "dashboard", Name: name})
	}
	dRows.Close()

	// 4. Widgets -> connect queries to dashboards
	wRows, err := h.DB.QueryContext(r.Context(), `
		SELECT DISTINCT dw.saved_query_id, dw.dashboard_id
		FROM dashboard_widgets dw
		JOIN dashboards d ON d.id = dw.dashboard_id
		WHERE d.owner_id = $1`, userID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not load widget links")
		return
	}
	for wRows.Next() {
		var qID, dID string
		if err := wRows.Scan(&qID, &dID); err != nil {
			wRows.Close()
			writeJSONError(w, http.StatusInternalServerError, "could not read widget link")
			return
		}
		graph.Edges = append(graph.Edges, LineageEdge{From: "q:" + qID, To: "d:" + dID})
	}
	wRows.Close()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(graph)
}
