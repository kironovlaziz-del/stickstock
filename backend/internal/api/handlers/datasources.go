package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"stickstock/backend/internal/api/middleware"
	"stickstock/backend/internal/connectors"
	"stickstock/backend/internal/crypto"
)

type DataSourceHandler struct {
	DB             *sql.DB
	EncryptionKey  string
}

type createDataSourceRequest struct {
	Name string `json:"name"`
	Kind string `json:"kind"` // postgres | mysql | mongodb | rest | file
	DSN  string `json:"dsn"`
}

type dataSourceResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"`
}

// List returns the caller's saved connections (never the DSN itself).
func (h *DataSourceHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())

	rows, err := h.DB.QueryContext(r.Context(),
		`SELECT id, name, kind FROM data_sources WHERE owner_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not list data sources")
		return
	}
	defer rows.Close()

	out := []dataSourceResponse{}
	for rows.Next() {
		var d dataSourceResponse
		if err := rows.Scan(&d.ID, &d.Name, &d.Kind); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "could not read data sources")
			return
		}
		out = append(out, d)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

// Create saves a new connection after test-pinging it via whichever
// connector matches Kind. DSN is encrypted at rest using AES-256-GCM.
func (h *DataSourceHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())

	var req createDataSourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" || req.DSN == "" {
		writeJSONError(w, http.StatusBadRequest, "name, kind and dsn are required")
		return
	}
	if req.Kind == string(connectors.KindFile) {
		writeJSONError(w, http.StatusBadRequest, "file data sources are created via the upload endpoint, not this one")
		return
	}

	if h.EncryptionKey == "" {
		writeJSONError(w, http.StatusInternalServerError, "encryption key not configured")
		return
	}

	encryptedDSN, err := crypto.Encrypt(h.EncryptionKey, req.DSN)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to encrypt credentials")
		return
	}

	conn, err := connectors.New(req.Kind, req.DSN)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer conn.Close()
	if err := conn.TestConnection(r.Context()); err != nil {
		writeJSONError(w, http.StatusBadRequest, "could not connect: "+err.Error())
		return
	}

	var id string
	err = h.DB.QueryRowContext(r.Context(),
		`INSERT INTO data_sources (owner_id, name, kind, dsn) VALUES ($1, $2, $3, $4) RETURNING id`,
		userID, req.Name, req.Kind, encryptedDSN,
	).Scan(&id)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not save data source")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(dataSourceResponse{ID: id, Name: req.Name, Kind: req.Kind})
}

// Delete removes a data source and, if it is a file upload, drops the
// underlying uploads-schema table to prevent accumulation of orphaned
// tables. DELETE /api/datasources/{id}
func (h *DataSourceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	id := r.PathValue("id")

	var kind, fileTable string
	err := h.DB.QueryRowContext(r.Context(),
		`SELECT kind, file_table FROM data_sources WHERE id = $1 AND owner_id = $2`,
		id, userID,
	).Scan(&kind, &fileTable)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "data source not found")
		return
	}

	tx, err := h.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not start transaction")
		return
	}
	defer tx.Rollback()

	if kind == "file" && fileTable != "" {
		if err := DeleteUploadTable(r.Context(), tx, fileTable); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "could not drop upload table: "+err.Error())
			return
		}
	}

	res, err := tx.ExecContext(r.Context(),
		`DELETE FROM data_sources WHERE id = $1 AND owner_id = $2`, id, userID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not delete data source")
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSONError(w, http.StatusNotFound, "data source not found")
		return
	}

	if err := tx.Commit(); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not commit transaction")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
