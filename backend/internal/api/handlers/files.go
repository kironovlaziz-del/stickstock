package handlers

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
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

const (
	maxUploadBytes  = 25 << 20 // 25MB, matches nginx client_max_body_size
	maxColumns      = 100      //  CSV
	sampleSize      = 100      // tipes
	insertBatchSize = 500      // one INSERT
)

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
	//(Postgres max 63 )
	if len(s) > 63 {
		s = s[:63]
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

// inferColumnTypeFromSample
//  "bigint", "double precision" or "text".
func inferColumnTypeFromSample(samples []string) string {
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

// convertValue
// NULL
func convertValue(val string, colType string) interface{} {
	val = strings.TrimSpace(val)
	if val == "" {
		return nil
	}
	switch colType {
	case "bigint":
		if i, err := strconv.ParseInt(val, 10, 64); err == nil {
			return i
		}
		return nil
	case "double precision":
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
		return nil
	default:
		return val
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
	reader.FieldsPerRecord = -1 // tolerate ragged rows
	reader.ReuseRecord = true


	rawHeader, err := reader.Read()
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "could not read CSV header: "+err.Error())
		return
	}
	if len(rawHeader) == 0 {
		writeJSONError(w, http.StatusBadRequest, "CSV header is empty")
		return
	}
	if len(rawHeader) > maxColumns {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("CSV has %d columns, maximum allowed is %d", len(rawHeader), maxColumns))
		return
	}


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


	samples := make([][]string, 0, sampleSize)
	rowCount := 0
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "could not read CSV row: "+err.Error())
			return
		}
		if len(row) > len(colNames) {
			row = row[:len(colNames)]
		}
		samples = append(samples, row)
		rowCount++
		if len(samples) >= sampleSize {
			break
		}
	}

	if rowCount == 0 {
		writeJSONError(w, http.StatusBadRequest, "file needs a header row plus at least one data row")
		return
	}


	colTypes := make([]string, len(colNames))
	for i := range colNames {
		vals := make([]string, 0, len(samples))
		for _, row := range samples {
			if i < len(row) {
				vals = append(vals, row[i])
			}
		}
		colTypes[i] = inferColumnTypeFromSample(vals)
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

	file2, _, err := r.FormFile("file")
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not re-open file")
		return
	}
	defer file2.Close()

	reader2 := csv.NewReader(file2)
	reader2.ReuseRecord = true

	if _, err := reader2.Read(); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not re-read header")
		return
	}

	totalRows := 0
	batch := make([][]string, 0, insertBatchSize)

	for {
		row, err := reader2.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "could not read CSV row: "+err.Error())
			return
		}

		if len(row) > len(colNames) {
			row = row[:len(colNames)]
		}

		if len(row) < len(colNames) {
			missing := make([]string, len(colNames)-len(row))
			row = append(row, missing...)
		}
		batch = append(batch, row)
		totalRows++
		if len(batch) >= insertBatchSize {
			if err := insertUploadRowsTyped(r.Context(), tx, tableName, colNames, colTypes, batch); err != nil {
				writeJSONError(w, http.StatusInternalServerError, "could not insert rows: "+err.Error())
				return
			}
			batch = batch[:0]
		}
	}

	if len(batch) > 0 {
		if err := insertUploadRowsTyped(r.Context(), tx, tableName, colNames, colTypes, batch); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "could not insert rows: "+err.Error())
			return
		}
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
		RowCount:     totalRows,
	})
}

// createUploadTable  insertUploadRowsTyped  %q 
//  tableName  colNames  sanitizeIdentifier.
func createUploadTable(ctx context.Context, tx *sql.Tx, table string, cols, types []string) error {
	defs := make([]string, len(cols))
	for i, c := range cols {
		defs[i] = fmt.Sprintf("%q %s", c, types[i])
	}
	stmt := fmt.Sprintf("CREATE TABLE uploads.%q (%s)", table, strings.Join(defs, ", "))
	_, err := tx.ExecContext(ctx, stmt)
	return err
}

// insertUploadRowsTyped
func insertUploadRowsTyped(ctx context.Context, tx *sql.Tx, table string, cols, types []string, rows [][]string) error {
	if len(rows) == 0 {
		return nil
	}

	quotedCols := make([]string, len(cols))
	for i, c := range cols {
		quotedCols[i] = fmt.Sprintf("%q", c)
	}
	colList := strings.Join(quotedCols, ", ")

	placeholders := make([]string, len(rows))
	args := make([]interface{}, 0, len(rows)*len(cols))
	argN := 1
	for i, row := range rows {
		ph := make([]string, len(cols))
		for j := range cols {
			ph[j] = fmt.Sprintf("$%d", argN)
			argN++

			var val interface{}
			if j < len(row) {
				val = convertValue(row[j], types[j])
			} else {
				val = nil
			}
			args = append(args, val)
		}
		placeholders[i] = "(" + strings.Join(ph, ", ") + ")"
	}

	stmt := fmt.Sprintf("INSERT INTO uploads.%q (%s) VALUES %s", table, colList, strings.Join(placeholders, ", "))
	_, err := tx.ExecContext(ctx, stmt, args...)
	return err
}

// DeleteUploadTable delete upload
// delete data_sources.
func DeleteUploadTable(ctx context.Context, tx *sql.Tx, tableName string) error {
	if tableName == "" {
		return nil
	}
	// tableName  "uploads.t_xxxx"
	parts := strings.SplitN(tableName, ".", 2)
	if len(parts) != 2 || parts[0] != "uploads" {
		return fmt.Errorf("invalid table name: %s", tableName)
	}
	stmt := fmt.Sprintf("DROP TABLE IF EXISTS uploads.%q CASCADE", parts[1])
	_, err := tx.ExecContext(ctx, stmt)
	return err
}