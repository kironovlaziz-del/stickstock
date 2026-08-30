package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"stickstock/backend/internal/api/middleware"
	"stickstock/backend/internal/connectors"
)

type SchemaHandler struct {
	DB             *sql.DB
}

// Get returns the tables/collections and columns a data source exposes,
// for the visual query builder's field picker: GET /api/datasources/{id}/schema
func (h *SchemaHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	id := r.PathValue("id")

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

	tables, err := conn.ListSchemas(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "could not list schema: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"tables": tables})
}