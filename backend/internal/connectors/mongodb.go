// MongoDB connector. Like REST, "query" here is a small JSON DSL, not
// SQL — Mongo has no SQL to pass through. The DSN is a standard mongodb://
// or mongodb+srv:// connection string; the database name is taken from
// its path if present, otherwise the query DSL must supply one via
// "database". queryengine's SELECT-only guard doesn't apply here (see
// isSQLKind in connectors.IsSQLKind) since there's no SQL text to check.
package connectors

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const defaultMongoLimit = 1000

type MongoConnector struct {
	client    *mongo.Client
	defaultDB string
}

func NewMongoConnector(dsn string) (*MongoConnector, error) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(dsn))
	if err != nil {
		return nil, fmt.Errorf("invalid mongodb dsn: %w", err)
	}

	return &MongoConnector{client: client, defaultDB: parseDefaultDatabase(dsn)}, nil
}

// parseDefaultDatabase pulls the database name out of the connection
// string's path (mongodb://host:port/dbname), the conventional place for
// it. Returns "" if there isn't one, which just means callers must supply
// "database" in their query DSL instead.
func parseDefaultDatabase(dsn string) string {
	u, err := url.Parse(dsn)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(u.Path, "/")
}

func (c *MongoConnector) TestConnection(ctx context.Context) error {
	return c.client.Ping(ctx, nil)
}

// ListSchemas samples a few documents per collection to infer field
// names, since MongoDB has no fixed schema to introspect the way SQL's
// information_schema does. DataType is always "mixed" — a real type per
// field would need scanning every document, not just a sample.
func (c *MongoConnector) ListSchemas(ctx context.Context) ([]TableInfo, error) {
	if c.defaultDB == "" {
		return nil, fmt.Errorf("mongodb dsn has no database in its path; add one, e.g. mongodb://host:27017/mydb")
	}
	db := c.client.Database(c.defaultDB)

	names, err := db.ListCollectionNames(ctx, bson.M{})
	if err != nil {
		return nil, err
	}

	tables := make([]TableInfo, 0, len(names))
	for _, name := range names {
		cur, err := db.Collection(name).Find(ctx, bson.M{}, options.Find().SetLimit(5))
		if err != nil {
			continue // best-effort schema sampling; skip a collection we can't read rather than failing the whole listing
		}

		var docs []bson.M
		_ = cur.All(ctx, &docs)
		cur.Close(ctx)

		fieldSet := map[string]struct{}{}
		for _, d := range docs {
			for k := range d {
				fieldSet[k] = struct{}{}
			}
		}
		cols := make([]ColumnInfo, 0, len(fieldSet))
		for k := range fieldSet {
			cols = append(cols, ColumnInfo{Name: k, DataType: "mixed", Nullable: true})
		}

		tables = append(tables, TableInfo{Schema: c.defaultDB, Name: name, Columns: cols})
	}
	return tables, nil
}

type mongoQuery struct {
	Database   string                 `json:"database"`
	Collection string                 `json:"collection"`
	Filter     map[string]interface{} `json:"filter"`
	Projection map[string]interface{} `json:"projection"`
	Sort       map[string]interface{} `json:"sort"`
	Limit      int64                  `json:"limit"`
}

func (c *MongoConnector) Query(ctx context.Context, query string, args ...interface{}) (*Result, error) {
	var q mongoQuery
	if err := json.Unmarshal([]byte(query), &q); err != nil {
		return nil, fmt.Errorf(`mongodb query must be JSON like {"collection":"orders","filter":{}}: %w`, err)
	}
	if q.Collection == "" {
		return nil, fmt.Errorf(`mongodb query requires a "collection" field`)
	}

	dbName := q.Database
	if dbName == "" {
		dbName = c.defaultDB
	}
	if dbName == "" {
		return nil, fmt.Errorf(`no database: set one in the connection string path or the query's "database" field`)
	}

	findOpts := options.Find()
	if q.Projection != nil {
		findOpts.SetProjection(q.Projection)
	}
	if q.Sort != nil {
		findOpts.SetSort(q.Sort)
	}
	// Hard limit 1000 doc.
	limit := q.Limit
	if limit <= 0 || limit > defaultMongoLimit {
		limit = defaultMongoLimit
	}
	findOpts.SetLimit(limit)

	filter := q.Filter
	if filter == nil {
		filter = bson.M{}
	}

	cur, err := c.client.Database(dbName).Collection(q.Collection).Find(ctx, filter, findOpts)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer cur.Close(ctx)

	var docs []bson.M
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("could not read results: %w", err)
	}

	return docsToResult(docs), nil
}

func (c *MongoConnector) Close() error {
	return c.client.Disconnect(context.Background())
}

func docsToResult(docs []bson.M) *Result {
	colSet := map[string]struct{}{}
	for _, d := range docs {
		for k := range d {
			colSet[k] = struct{}{}
		}
	}
	cols := make([]string, 0, len(colSet))
	for k := range colSet {
		cols = append(cols, k)
	}
	sort.Strings(cols)

	res := &Result{Columns: cols}
	for _, d := range docs {
		row := make(Row, len(cols))
		for i, c := range cols {
			row[i] = jsonSafe(d[c])
		}
		res.Rows = append(res.Rows, row)
	}
	return res
}

// jsonSafe converts a BSON-decoded value (which may be a mongo-driver
// primitive type like ObjectID or DateTime) into something plain enough
// for encoding/json to serialize in the HTTP response. Those primitive
// types already implement MarshalJSON for interop, so a marshal→unmarshal
// round trip gets a plain string/number/map/slice without needing to
// hand-list every BSON type here.
func jsonSafe(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	var out interface{}
	if err := json.Unmarshal(b, &out); err != nil {
		return string(b)
	}
	return out
}