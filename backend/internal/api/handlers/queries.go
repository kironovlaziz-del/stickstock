package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"stickstock/backend/internal/api/middleware"
	"stickstock/backend/internal/connectors"
	"stickstock/backend/internal/crypto"
	"stickstock/backend/internal/queryengine"
)

type QueryHandler struct {
	DB             *sql.DB
	EncryptionKey  string // ключ для AES-256-GCM
}

const maxResultRows = 1000

type runResponse struct {
	Columns   []string        `json:"columns"`
	Rows      [][]interface{} `json:"rows"`
	Truncated bool            `json:"truncated"`
}

// dataSourceForOwner returns enough about a data source to run a query
// against it. fileTable is only set for kind == "file" (see
// migrations/0002_uploads.sql) and is empty otherwise. A free function
// (not a *QueryHandler method) so ExportHandler can share it.

func dataSourceForOwner(ctx context.Context, db *sql.DB, id, ownerID, encryptionKey string) (kind, dsn, fileTable string, err error) {
    var fileTableNull sql.NullString
    err = db.QueryRowContext(ctx,
        `SELECT kind, dsn, file_table FROM data_sources WHERE id = $1 AND owner_id = $2`, id, ownerID,
    ).Scan(&kind, &dsn, &fileTableNull)
    if err != nil {
        return "", "", "", err
    }
    fileTable = fileTableNull.String

    if kind != string(connectors.KindFile) && dsn != "" {
        decrypted, err := crypto.Decrypt(encryptionKey, dsn)
        if err != nil {
            // Если не удалось расшифровать, используем DSN как открытый текст (легаси-данные)
            log.Printf("WARNING: Decrypt error for id %s: %v — using plaintext DSN", id, err)
        } else {
            dsn = decrypted
        }
    }
    return
}

// validateQueryText applies the right safety check for the data source's
// kind. SQL sources (Postgres/MySQL/file-uploads-as-tables) get the
// SELECT-only guard; "file" sources get an additional check that the
// query can't reach outside its own uploads-schema table. MongoDB/REST's
// "query" is a JSON DSL, not SQL — queryengine's SQL guard doesn't apply
// to them, so they just need to actually be valid JSON.
func validateQueryText(kind, fileTable, text string) error {
	if connectors.IsSQLKind(kind) {
		if err := queryengine.ValidateReadOnly(text); err != nil {
			return err
		}
		if kind == string(connectors.KindFile) {
			return queryengine.ValidateFileScope(text, fileTable)
		}
		return nil
	}

	var js interface{}
	if err := json.Unmarshal([]byte(text), &js); err != nil {
		return fmt.Errorf("query must be valid JSON for a %s data source", kind)
	}
	return nil
}

// execute binds params (SQL sources only — see queryengine.BindParams)
// and runs the query through whichever Connector matches kind.
func execute(ctx context.Context, kind, dsn, queryText string, params map[string]interface{}) (*runResponse, error) {
	boundQuery := queryText
	var args []interface{}

	if connectors.IsSQLKind(kind) {
		bound, boundArgs, err := queryengine.BindParams(queryText, params)
		if err != nil {
			return nil, err
		}
		boundQuery, args = bound, boundArgs
	}
	// REST queries are a fixed JSON document — params aren't substituted
	// into them yet (a future iteration could add {{param}} templating to
	// the JSON DSL if a use case needs it).

	conn, err := connectors.New(kind, dsn)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	result, err := conn.Query(ctx, boundQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	resp := &runResponse{Columns: result.Columns, Rows: result.Rows}
	if len(resp.Rows) > maxResultRows {
		resp.Rows = resp.Rows[:maxResultRows]
		resp.Truncated = true
	}
	return resp, nil
}

// --- Ad-hoc execution (unsaved query, e.g. from the editor) ---

type runAdHocRequest struct {
	DataSourceID string                 `json:"data_source_id"`
	SQL          string                 `json:"sql"`
	Params       map[string]interface{} `json:"params"`
}

func (h *QueryHandler) RunAdHoc(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())

	var req runAdHocRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.DataSourceID == "" || req.SQL == "" {
		writeJSONError(w, http.StatusBadRequest, "data_source_id and sql are required")
		return
	}

	kind, dsn, fileTable, err := dataSourceForOwner(r.Context(), h.DB, req.DataSourceID, userID, h.EncryptionKey)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "data source not found")
		return
	}
	if err := validateQueryText(kind, fileTable, req.SQL); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := execute(r.Context(), kind, dsn, req.SQL, req.Params)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// --- Saved query CRUD + versioning ---

type savedQueryResponse struct {
	ID           string     `json:"id"`
	DataSourceID string     `json:"data_source_id"`
	Name         string     `json:"name"`
	SQLText      string     `json:"sql_text,omitempty"`
	Version      int        `json:"version"`
	LastRunAt    *time.Time `json:"last_run_at,omitempty"`
	LastRowCount *int       `json:"last_row_count,omitempty"`
}

type createSavedQueryRequest struct {
	DataSourceID string `json:"data_source_id"`
	Name         string `json:"name"`
	SQLText      string `json:"sql_text"`
}

func (h *QueryHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())

	var req createSavedQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" || req.SQLText == "" || req.DataSourceID == "" {
		writeJSONError(w, http.StatusBadRequest, "data_source_id, name and sql_text are required")
		return
	}

	kind, _, fileTable, err := dataSourceForOwner(r.Context(), h.DB, req.DataSourceID, userID, h.EncryptionKey)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "data source not found")
		return
	}
	if err := validateQueryText(kind, fileTable, req.SQLText); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	tx, err := h.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not start transaction")
		return
	}
	defer tx.Rollback()

	var id string
	err = tx.QueryRowContext(r.Context(),
		`INSERT INTO saved_queries (owner_id, data_source_id, name, sql_text, version)
		 VALUES ($1, $2, $3, $4, 1) RETURNING id`,
		userID, req.DataSourceID, req.Name, req.SQLText,
	).Scan(&id)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not save query")
		return
	}

	if _, err := tx.ExecContext(r.Context(),
		`INSERT INTO saved_query_versions (saved_query_id, version, sql_text) VALUES ($1, 1, $2)`,
		id, req.SQLText,
	); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not save query version")
		return
	}

	if err := tx.Commit(); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not commit transaction")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(savedQueryResponse{
		ID: id, DataSourceID: req.DataSourceID, Name: req.Name, SQLText: req.SQLText, Version: 1,
	})
}

func (h *QueryHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())

	rows, err := h.DB.QueryContext(r.Context(),
		`SELECT id, data_source_id, name, version, last_run_at, last_row_count
		 FROM saved_queries WHERE owner_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not list queries")
		return
	}
	defer rows.Close()

	out := []savedQueryResponse{}
	for rows.Next() {
		var q savedQueryResponse
		var lastRunAt sql.NullTime
		var lastRowCount sql.NullInt64
		if err := rows.Scan(&q.ID, &q.DataSourceID, &q.Name, &q.Version, &lastRunAt, &lastRowCount); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "could not read queries")
			return
		}
		if lastRunAt.Valid {
			q.LastRunAt = &lastRunAt.Time
		}
		if lastRowCount.Valid {
			n := int(lastRowCount.Int64)
			q.LastRowCount = &n
		}
		out = append(out, q)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (h *QueryHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	id := r.PathValue("id")

	var q savedQueryResponse
	var lastRunAt sql.NullTime
	var lastRowCount sql.NullInt64
	err := h.DB.QueryRowContext(r.Context(),
		`SELECT id, data_source_id, name, sql_text, version, last_run_at, last_row_count
		 FROM saved_queries WHERE id = $1 AND owner_id = $2`,
		id, userID,
	).Scan(&q.ID, &q.DataSourceID, &q.Name, &q.SQLText, &q.Version, &lastRunAt, &lastRowCount)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "query not found")
		return
	}
	if lastRunAt.Valid {
		q.LastRunAt = &lastRunAt.Time
	}
	if lastRowCount.Valid {
		n := int(lastRowCount.Int64)
		q.LastRowCount = &n
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(q)
}

type updateSavedQueryRequest struct {
	Name    string `json:"name"`
	SQLText string `json:"sql_text"`
}

// Update writes a new version rather than overwriting sql_text in place,
// preserving the prior text in saved_query_versions (the "Git-like
// history" requirement).
func (h *QueryHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	id := r.PathValue("id")

	var req updateSavedQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.SQLText == "" {
		writeJSONError(w, http.StatusBadRequest, "sql_text is required")
		return
	}

	var dataSourceID string
	if err := h.DB.QueryRowContext(r.Context(),
		`SELECT data_source_id FROM saved_queries WHERE id = $1 AND owner_id = $2`, id, userID,
	).Scan(&dataSourceID); err != nil {
		writeJSONError(w, http.StatusNotFound, "query not found")
		return
	}
	kind, _, fileTable, err := dataSourceForOwner(r.Context(), h.DB, dataSourceID, userID, h.EncryptionKey)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "data source not found")
		return
	}
	if err := validateQueryText(kind, fileTable, req.SQLText); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	tx, err := h.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not start transaction")
		return
	}
	defer tx.Rollback()

	var currentVersion int
	if err := tx.QueryRowContext(r.Context(),
		`SELECT version FROM saved_queries WHERE id = $1 AND owner_id = $2 FOR UPDATE`, id, userID,
	).Scan(&currentVersion); err != nil {
		writeJSONError(w, http.StatusNotFound, "query not found")
		return
	}
	newVersion := currentVersion + 1

	setClause := "sql_text = $1, version = $2"
	args := []interface{}{req.SQLText, newVersion}
	nextArg := 3
	if req.Name != "" {
		setClause += fmt.Sprintf(", name = $%d", nextArg)
		args = append(args, req.Name)
		nextArg++
	}
	args = append(args, id, userID)

	updateSQL := fmt.Sprintf(
		"UPDATE saved_queries SET %s WHERE id = $%d AND owner_id = $%d",
		setClause, nextArg, nextArg+1,
	)
	if _, err := tx.ExecContext(r.Context(), updateSQL, args...); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not update query")
		return
	}

	if _, err := tx.ExecContext(r.Context(),
		`INSERT INTO saved_query_versions (saved_query_id, version, sql_text) VALUES ($1, $2, $3)`,
		id, newVersion, req.SQLText,
	); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not save query version")
		return
	}

	if err := tx.Commit(); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not commit transaction")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"id": id, "version": newVersion})
}

func (h *QueryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	id := r.PathValue("id")

	res, err := h.DB.ExecContext(r.Context(),
		`DELETE FROM saved_queries WHERE id = $1 AND owner_id = $2`, id, userID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not delete query")
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSONError(w, http.StatusNotFound, "query not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type versionResponse struct {
	Version   int       `json:"version"`
	SQLText   string    `json:"sql_text"`
	CreatedAt time.Time `json:"created_at"`
}

func (h *QueryHandler) Versions(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	id := r.PathValue("id")

	var owner string
	if err := h.DB.QueryRowContext(r.Context(),
		`SELECT owner_id FROM saved_queries WHERE id = $1`, id,
	).Scan(&owner); err != nil || owner != userID {
		writeJSONError(w, http.StatusNotFound, "query not found")
		return
	}

	rows, err := h.DB.QueryContext(r.Context(),
		`SELECT version, sql_text, created_at FROM saved_query_versions WHERE saved_query_id = $1 ORDER BY version DESC`, id)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not list versions")
		return
	}
	defer rows.Close()

	out := []versionResponse{}
	for rows.Next() {
		var v versionResponse
		if err := rows.Scan(&v.Version, &v.SQLText, &v.CreatedAt); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "could not read versions")
			return
		}
		out = append(out, v)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

type runSavedRequest struct {
	Params map[string]interface{} `json:"params"`
}

// callerCanViewSavedQuery reports whether userID has at least viewer
// access to some dashboard that uses savedQueryID via a widget — this is
// what lets a shared dashboard's collaborators (who don't own the
// underlying saved query) still see the widget's live data when viewing
// that dashboard.
func callerCanViewSavedQuery(ctx context.Context, db *sql.DB, savedQueryID, userID string) (bool, error) {
	var exists bool
	err := db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM dashboard_widgets dw
			JOIN dashboards d ON d.id = dw.dashboard_id
			WHERE dw.saved_query_id = $1
			AND (
				d.owner_id = $2
				OR EXISTS (
					SELECT 1 FROM dashboard_collaborators dc
					WHERE dc.dashboard_id = d.id AND dc.user_id = $2
				)
			)
		)`, savedQueryID, userID,
	).Scan(&exists)
	return exists, err
}

func (h *QueryHandler) RunSaved(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	id := r.PathValue("id")

	var req runSavedRequest
	_ = json.NewDecoder(r.Body).Decode(&req) // params are optional if the query takes none

	var dataSourceID, sqlText, queryOwnerID string
	if err := h.DB.QueryRowContext(r.Context(),
		`SELECT data_source_id, sql_text, owner_id FROM saved_queries WHERE id = $1`, id,
	).Scan(&dataSourceID, &sqlText, &queryOwnerID); err != nil {
		writeJSONError(w, http.StatusNotFound, "query not found")
		return
	}

	if queryOwnerID != userID {
		canView, err := callerCanViewSavedQuery(r.Context(), h.DB, id, userID)
		if err != nil || !canView {
			writeJSONError(w, http.StatusNotFound, "query not found")
			return
		}
	}

	kind, dsn, _, err := dataSourceForOwner(r.Context(), h.DB, dataSourceID, queryOwnerID, h.EncryptionKey)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "data source not found")
		return
	}

	// Saved queries were already validated at creation/update time — no
	// need to re-run validateQueryText on every execution.
	resp, err := execute(r.Context(), kind, dsn, sqlText, req.Params)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Freshness tracking for data-health monitoring — best-effort; a
	// failure here shouldn't fail the request since the query itself
	// already succeeded.
	if _, err := h.DB.ExecContext(r.Context(),
		`UPDATE saved_queries SET last_run_at = now(), last_row_count = $1 WHERE id = $2`,
		len(resp.Rows), id,
	); err != nil {
		log.Printf("could not stamp freshness for query %s: %v", id, err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}