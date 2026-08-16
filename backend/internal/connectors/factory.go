package connectors

import "fmt"

// New instantiates the Connector for a given data source kind.
// "file" kind reuses the Postgres connector: uploaded CSVs are
// materialized as real tables in the app's own "uploads" schema (see
// migrations/0002_uploads.sql), so querying one IS just Postgres.
func New(kind string, dsn string) (Connector, error) {
	switch Kind(kind) {
	case KindPostgres, KindFile:
		return NewPostgresConnector(dsn)
	case KindMySQL:
		return NewMySQLConnector(dsn)
	case KindMongoDB:
		return NewMongoConnector(dsn)
	case KindREST:
		return NewRESTConnector(dsn)
	default:
		return nil, fmt.Errorf("connector kind %q is not implemented yet", kind)
	}
}

// IsSQLKind reports whether kind is queried with SQL text (subject to
// queryengine's SELECT-only guard and :param binding), as opposed to a
// JSON DSL like MongoDB/REST use. Shared by the API handlers and the
// scheduler worker so both treat kinds consistently.
func IsSQLKind(kind string) bool {
	switch Kind(kind) {
	case KindPostgres, KindMySQL, KindFile:
		return true
	default:
		return false
	}
}
