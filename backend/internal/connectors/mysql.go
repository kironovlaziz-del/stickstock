package connectors

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
)

type MySQLConnector struct {
	dsn string
	db  *sql.DB
}

func NewMySQLConnector(dsn string) (*MySQLConnector, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	return &MySQLConnector{dsn: dsn, db: db}, nil
}

func (c *MySQLConnector) TestConnection(ctx context.Context) error {
	return c.db.PingContext(ctx)
}

func (c *MySQLConnector) ListSchemas(ctx context.Context) ([]TableInfo, error) {
	const q = `
		SELECT table_schema, table_name, column_name, data_type, is_nullable
		FROM information_schema.columns
		WHERE table_schema NOT IN ('information_schema', 'mysql', 'performance_schema', 'sys')
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

var pgPlaceholder = regexp.MustCompile(`\$(\d+)`)

// toMySQLArgs rewrites Postgres-style $1,$2,... placeholders (as produced
// by queryengine.BindParams, which all connectors receive regardless of
// dialect) into MySQL's positional "?" placeholders. It expands the args
// slice so a placeholder used more than once gets its value duplicated at
// each occurrence — MySQL's driver has no concept of a named or reusable
// positional parameter the way $N is reusable in Postgres.
func toMySQLArgs(query string, args []interface{}) (string, []interface{}) {
	var newArgs []interface{}
	rewritten := pgPlaceholder.ReplaceAllStringFunc(query, func(m string) string {
		n, err := strconv.Atoi(m[1:])
		if err == nil && n >= 1 && n <= len(args) {
			newArgs = append(newArgs, args[n-1])
		}
		return "?"
	})
	return rewritten, newArgs
}

func (c *MySQLConnector) Query(ctx context.Context, query string, args ...interface{}) (*Result, error) {
	mysqlQuery, mysqlArgs := toMySQLArgs(query, args)

	rows, err := c.db.QueryContext(ctx, mysqlQuery, mysqlArgs...)
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

func (c *MySQLConnector) Close() error {
	return c.db.Close()
}