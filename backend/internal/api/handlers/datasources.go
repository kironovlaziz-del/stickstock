package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"stickstock/backend/internal/api/middleware"
	"stickstock/backend/internal/connectors"
)

type DataSourceHandler struct {
	DB *sql.DB
}

type createDataSourceRequest struct {
	Name          string `json:"name"`
	Kind          string `json:"kind"`
	DSN           string `json:"dsn"`
	SSHHost       string `json:"ssh_host,omitempty"`
	SSHPort       int    `json:"ssh_port,omitempty"`
	SSHUser       string `json:"ssh_user,omitempty"`
	SSHPassword   string `json:"ssh_password,omitempty"`
	SSHPrivateKey string `json:"ssh_private_key,omitempty"`
}

type dataSourceResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	SSHHost string `json:"ssh_host,omitempty"`
	SSHPort int    `json:"ssh_port,omitempty"`
	SSHUser string `json:"ssh_user,omitempty"`
}

func (h *DataSourceHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	log.Printf("List data sources for user: %s", userID)

	rows, err := h.DB.QueryContext(r.Context(),
		`SELECT id, name, kind, COALESCE(ssh_host, ''), COALESCE(ssh_port, 0), COALESCE(ssh_user, '') FROM data_sources WHERE owner_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		log.Printf("Query error: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "could not read data sources: "+err.Error())
		return
	}
	defer rows.Close()

	out := []dataSourceResponse{}
	for rows.Next() {
		var d dataSourceResponse
		if err := rows.Scan(&d.ID, &d.Name, &d.Kind, &d.SSHHost, &d.SSHPort, &d.SSHUser); err != nil {
			log.Printf("Scan error: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "could not read data sources: "+err.Error())
			return
		}
		out = append(out, d)
	}

	log.Printf("Found %d data sources", len(out))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (h *DataSourceHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok || userID == "" {
		writeJSONError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}
	log.Printf("Create data source: userID=%s", userID)

	var req createDataSourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" || req.DSN == "" {
		writeJSONError(w, http.StatusBadRequest, "name, kind and dsn are required")
		return
	}
	if req.Kind == string(connectors.KindFile) {
		writeJSONError(w, http.StatusBadRequest, "file data sources are created via the upload endpoint, not this one")
		return
	}

	conn, err := connectors.NewWithSSH(req.Kind, req.DSN, req.SSHHost, req.SSHPort, req.SSHUser, req.SSHPassword, req.SSHPrivateKey)
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
		userID, req.Name, req.Kind, req.DSN,
	).Scan(&id)
	if err != nil {
		log.Printf("Insert error: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "could not save data source: "+err.Error())
		return
	}

	log.Printf("Data source created: id=%s, name=%s", id, req.Name)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(dataSourceResponse{ID: id, Name: req.Name, Kind: req.Kind})
}

func (h *DataSourceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	log.Printf("Deleting data source: id=%s", id)
	_, err := h.DB.ExecContext(r.Context(), `DELETE FROM data_sources WHERE id = $1`, id)
	if err != nil {
		log.Printf("Delete error: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "could not delete data source")
		return
	}
	log.Printf("Data source deleted: id=%s", id)
	w.WriteHeader(http.StatusNoContent)
}
