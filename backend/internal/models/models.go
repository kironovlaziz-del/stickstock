package models

import "time"

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Locale       string    `json:"locale"`
	CreatedAt    time.Time `json:"created_at"`
}

// DataSource is a saved connection to an external system: a database,
// REST API, or an uploaded file treated as a temporary table.
type DataSource struct {
	ID        string    `json:"id"`
	OwnerID   string    `json:"owner_id"`
	Name      string    `json:"name"`
	Kind      string    `json:"kind"` // "postgres" | "mysql" | "mongodb" | "rest" | "file"
	DSN       string    `json:"-"`    // encrypted at rest; never serialized to clients
	CreatedAt time.Time `json:"created_at"`
}

// SavedQuery is a named, optionally parameterized query ("view") that can
// be re-run, versioned, and used as a data source for visualizations.
type SavedQuery struct {
	ID           string    `json:"id"`
	OwnerID      string    `json:"owner_id"`
	DataSourceID string    `json:"data_source_id"`
	Name         string    `json:"name"`
	SQLText      string    `json:"sql_text"`
	Params       []byte    `json:"params"` // JSON schema of parameters
	Version      int       `json:"version"`
	CreatedAt    time.Time `json:"created_at"`
}

type Dashboard struct {
	ID        string    `json:"id"`
	OwnerID   string    `json:"owner_id"`
	Name      string    `json:"name"`
	Layout    []byte    `json:"layout"` // JSON grid layout (react-grid-layout format)
	CreatedAt time.Time `json:"created_at"`
}

type DashboardWidget struct {
	ID          string `json:"id"`
	DashboardID string `json:"dashboard_id"`
	SavedQueryID string `json:"saved_query_id"`
	ChartType   string `json:"chart_type"` // "line" | "bar" | "pie" | "heatmap" | "table" ...
	Config      []byte `json:"config"`     // JSON chart config (axes, colors, etc.)
}

type QueryResult struct {
	Columns []string        `json:"columns"`
	Rows    [][]interface{} `json:"rows"`
}
