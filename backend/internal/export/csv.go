// Package export renders a query result ({columns, rows}, the same shape
// internal/api/handlers already uses) into CSV, XLSX, or PDF, streamed
// directly to an io.Writer (normally an http.ResponseWriter) rather than
// buffered in memory first.
package export

import (
	"encoding/csv"
	"fmt"
	"io"
)

func ToCSV(w io.Writer, columns []string, rows [][]interface{}) error {
	cw := csv.NewWriter(w)

	if err := cw.Write(columns); err != nil {
		return err
	}
	for _, row := range rows {
		record := make([]string, len(row))
		for i, v := range row {
			record[i] = fmt.Sprintf("%v", v)
		}
		if err := cw.Write(record); err != nil {
			return err
		}
	}

	cw.Flush()
	return cw.Error()
}
