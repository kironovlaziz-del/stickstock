// Package connectors defines a common interface that every data source
// type (Postgres, MySQL, MongoDB, REST, uploaded files) implements, so
// the query engine and the visual query builder can treat them uniformly.
package connectors

import "context"

type TableInfo struct {
	Schema  string       `json:"schema"`
	Name    string       `json:"name"`
	Columns []ColumnInfo `json:"columns"`
}

type ColumnInfo struct {
	Name     string `json:"name"`
	DataType string `json:"data_type"`
	Nullable bool   `json:"nullable"`
}

type Row = []interface{}

type Result struct {
	Columns []string `json:"columns"`
	Rows    []Row    `json:"rows"`
}

// Connector is implemented by every supported data source type.
type Connector interface {
	// TestConnection verifies credentials/reachability without running a query.
	TestConnection(ctx context.Context) error

	// ListSchemas returns an overview of available tables/collections and
	// their columns, used for the "browse data" view (like DBeaver).
	ListSchemas(ctx context.Context) ([]TableInfo, error)

	// Query executes a read query and returns tabular results. For SQL
	// sources this is raw SQL; non-SQL connectors (Mongo, REST) translate
	// their native query language into the same Result shape.
	Query(ctx context.Context, query string, args ...interface{}) (*Result, error)

	// Close releases underlying resources (connections, clients).
	Close() error
}

// Kind identifies which concrete connector to instantiate for a DataSource.
type Kind string

const (
	KindPostgres Kind = "postgres"
	KindMySQL    Kind = "mysql"
	KindMongoDB  Kind = "mongodb"
	KindREST     Kind = "rest"
	KindFile     Kind = "file"
)
