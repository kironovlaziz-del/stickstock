package connectors

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type PostgresConnector struct {
	dsn string
	db  *sql.DB
}

func NewPostgresConnector(dsn string) (*PostgresConnector, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	return &PostgresConnector{dsn: dsn, db: db}, nil
}

func (c *PostgresConnector) TestConnection(ctx context.Context) error {
	return c.db.PingContext(ctx)
}

func (c *PostgresConnector) ListSchemas(ctx context.Context) ([]TableInfo, error) {
	const q = `
		SELECT table_schema, table_name, column_name, data_type, is_nullable
		FROM information_schema.columns
		WHERE table_schema NOT IN ('pg_catalog', 'information_schema')
		ORDER BY table_schema, table_name, ordinal_position`

	rows, err := c.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tableIndex := map[string]int{}
	var tables []TableInfo

	for rows.Next() {
		var schema, table, col, dtype, nullable string
		if err := rows.Scan(&schema, &table, &col, &dtype, &nullable); err != nil {
			return nil, err
		}
		key := schema + "." + table
		idx, ok := tableIndex[key]
		if !ok {
			tables = append(tables, TableInfo{Schema: schema, Name: table})
			idx = len(tables) - 1
			tableIndex[key] = idx
		}
		tables[idx].Columns = append(tables[idx].Columns, ColumnInfo{
			Name:     col,
			DataType: dtype,
			Nullable: nullable == "YES",
		})
	}
	return tables, rows.Err()
}

// Query runs a read-only SQL statement. The API layer is responsible for
// enforcing that only SELECT statements reach here (see handlers/query.go
// in a later iteration) — this connector does not itself sanitize SQL.
func (c *PostgresConnector) Query(ctx context.Context, query string, args ...interface{}) (*Result, error) {
	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	const maxRows = 1000
	res := &Result{Columns: cols}
	rowCount := 0
	for rows.Next() {
		if rowCount >= maxRows {
			// Достигнут лимит — прекращаем чтение
			break
		}
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		res.Rows = append(res.Rows, vals)
		rowCount++
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return res, nil
}

func (c *PostgresConnector) Close() error {
	return c.db.Close()
}