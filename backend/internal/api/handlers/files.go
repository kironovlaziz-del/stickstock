package handlers

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"stickstock/backend/internal/api/middleware"
)

type FileHandler struct {
	DB          *sql.DB
	DatabaseURL string // reused as the DSN so uploaded tables are queryable through the normal Postgres connector
}

const maxUploadBytes = 25 << 20 // 25MB, matches nginx client_max_body_size

var identifierSanitizer = regexp.MustCompile(`[^a-z0-9_]+`)

// sanitizeIdentifier turns arbitrary CSV header text into a safe Postgres
// column identifier. Callers rely on the output containing only
// [a-z0-9_] — createUploadTable and insertUploadRows use %q formatting to
// quote identifiers, which is only safe here *because* nothing but those
// characters can come out of this function.
func sanitizeIdentifier(name string, fallback string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = identifierSanitizer.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	if s == "" {
		s = fallback
	}
	if s[0] >= '0' && s[0] <= '9' {
		s = "c_" + s
	}
	return s
}

func randomSuffix() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// inferColumnType guesses a Postgres column type from a sample of string
// values from a CSV column: bigint if every non-empty value parses as an
// integer, double precision if every value parses as a float, else text.
// Known limitation: this loses leading zeros (e.g. a "007" code becomes
// the integer 7) — fine for genuinely numeric columns, not for ID-like
// codes. No attempt is made to distinguish the two automatically.
func inferColumnType(samples []string) string {
	allInt, allFloat, seenAny := true, true, false

	for _, v := range samples {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		seenAny = true
		if _, err := strconv.ParseInt(v, 10, 64); err != nil {
			allInt = false
		}
		if _, err := strconv.ParseFloat(v, 64); err != nil {
			allFloat = false
		}
	}

	switch {
	case !seenAny:
		return "text"
	case allInt:
		return "bigint"
	case allFloat:
		return "double precision"
	default:
		return "text"
	}
}

type uploadResponse struct {
	DataSourceID string   `json:"data_source_id"`
	Table        string   `json:"table"`
	Columns      []string `json:"columns"`
	RowCount     int      `json:"row_count"`
}

// Upload parses a CSV file, materializes it as a new table in the
// "uploads" Postgres schema, and registers a "file" data source pointing
// at it. XLSX isn't supported yet — see README roadmap.
func (h *FileHandler) Upload(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		writeJSONError(w, http.StatusBadRequest, "could not parse upload (max 25MB)")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, `form field "file" is required`)
		return
	}
	defer file.Close()

	name := r.FormValue("name")
	if name == "" {
		name = header.Filename
	}

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1 // tolerate ragged rows; short rows get NULLs, extra fields are ignored
	records, err := reader.ReadAll()
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "could not parse CSV: "+err.Error())
		return
	}
	if len(records) < 2 {
		writeJSONError(w, http.StatusBadRequest, "file needs a header row plus at least one data row")
		return
	}

	rawHeader := records[0]
	dataRows := records[1:]

	colNames := make([]string, len(rawHeader))
	seen := map[string]int{}
	for i, rawName := range rawHeader {
		base := sanitizeIdentifier(rawName, fmt.Sprintf("col_%d", i+1))
		final := base
		if n, ok := seen[base]; ok {
			n++
			final = fmt.Sprintf("%s_%d", base, n)
			seen[base] = n
		} else {
			seen[base] = 0
		}
		colNames[i] = final
	}

	colTypes := make([]string, len(colNames))
	for i := range colNames {
		samples := make([]string, 0, len(dataRows))
		for _, row := range dataRows {
			if i < len(row) {
				samples = append(samples, row[i])
			}
		}
		colTypes[i] = inferColumnType(samples)
	}

	suffix, err := randomSuffix()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not generate table name")
		return
	}
	tableName := "t_" + suffix

	tx, err := h.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not start transaction")
		return
	}
	defer tx.Rollback()

	if err := createUploadTable(r.Context(), tx, tableName, colNames, colTypes); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not create table: "+err.Error())
		return
	}
	if err := insertUploadRows(r.Context(), tx, tableName, colNames, dataRows); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not insert rows: "+err.Error())
		return
	}

	qualifiedTable := "uploads." + tableName
	var dataSourceID string
	err = tx.QueryRowContext(r.Context(),
		`INSERT INTO data_sources (owner_id, name, kind, dsn, file_table) VALUES ($1, $2, 'file', $3, $4) RETURNING id`,
		userID, name, h.DatabaseURL, qualifiedTable,
	).Scan(&dataSourceID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not register data source: "+err.Error())
		return
	}

	if err := tx.Commit(); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not commit transaction")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(uploadResponse{
		DataSourceID: dataSourceID,
		Table:        qualifiedTable,
		Columns:      colNames,
		RowCount:     len(dataRows),
	})
}

// createUploadTable and insertUploadRows use %q to quote identifiers,
// which is safe here specifically because tableName and cols are always
// either machine-generated (tableName: "t_" + hex) or passed through
// sanitizeIdentifier, so neither can contain a character %q would need to
// escape.
func createUploadTable(ctx context.Context, tx *sql.Tx, table string, cols, types []string) error {
	defs := make([]string, len(cols))
	for i, c := range cols {
		defs[i] = fmt.Sprintf("%q %s", c, types[i])
	}
	stmt := fmt.Sprintf("CREATE TABLE uploads.%q (%s)", table, strings.Join(defs, ", "))
	_, err := tx.ExecContext(ctx, stmt)
	return err
}

const insertBatchSize = 500

func insertUploadRows(ctx context.Context, tx *sql.Tx, table string, cols []string, rows [][]string) error {
	quotedCols := make([]string, len(cols))
	for i, c := range cols {
		quotedCols[i] = fmt.Sprintf("%q", c)
	}
	colList := strings.Join(quotedCols, ", ")

	for start := 0; start < len(rows); start += insertBatchSize {
		end := start + insertBatchSize
		if end > len(rows) {
			end = len(rows)
		}
		batch := rows[start:end]

		placeholders := make([]string, len(batch))
		args := make([]interface{}, 0, len(batch)*len(cols))
		argN := 1
		for i, row := range batch {
			ph := make([]string, len(cols))
			for j := range cols {
				ph[j] = fmt.Sprintf("$%d", argN)
				argN++
				if j < len(row) {
					args = append(args, row[j])
				} else {
					args = append(args, nil)
				}
			}
			placeholders[i] = "(" + strings.Join(ph, ", ") + ")"
		}

		stmt := fmt.Sprintf("INSERT INTO uploads.%q (%s) VALUES %s", table, colList, strings.Join(placeholders, ", "))
		if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
			return err
		}
	}
	return nil
}
