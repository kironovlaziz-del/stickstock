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

type lineageNode struct {
	ID   string `json:"id"`
	Type string `json:"type"` // "data_source" | "saved_query" | "dashboard"
	Name string `json:"name"`
}

type lineageEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type lineageGraph struct {
	Nodes []lineageNode `json:"nodes"`
	Edges []lineageEdge `json:"edges"`
}

// Get builds the caller's full lineage graph: which data source feeds
// which saved query, and which dashboards use that query via a widget —
// "where did this number come from" (see spec section on data lineage).
// This deliberately doesn't need its own tracking tables: the answer is
// already fully captured by existing foreign keys, so it's just three
// queries and some ID-prefixing to keep node IDs unique across types.
// GET /api/lineage
func (h *LineageHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	graph := lineageGraph{Nodes: []lineageNode{}, Edges: []lineageEdge{}}

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
			writeJSONError(w, http.StatusInternalServerError, "could not read data sources")
			return
		}
		graph.Nodes = append(graph.Nodes, lineageNode{ID: "ds:" + id, Type: "data_source", Name: name})
	}
	dsRows.Close()

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
			writeJSONError(w, http.StatusInternalServerError, "could not read saved queries")
			return
		}
		graph.Nodes = append(graph.Nodes, lineageNode{ID: "q:" + id, Type: "saved_query", Name: name})
		graph.Edges = append(graph.Edges, lineageEdge{From: "ds:" + dsID, To: "q:" + id})
	}
	qRows.Close()

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
			writeJSONError(w, http.StatusInternalServerError, "could not read dashboards")
			return
		}
		graph.Nodes = append(graph.Nodes, lineageNode{ID: "d:" + id, Type: "dashboard", Name: name})
	}
	dRows.Close()

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
			writeJSONError(w, http.StatusInternalServerError, "could not read widget links")
			return
		}
		graph.Edges = append(graph.Edges, lineageEdge{From: "q:" + qID, To: "d:" + dID})
	}
	wRows.Close()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(graph)
}
