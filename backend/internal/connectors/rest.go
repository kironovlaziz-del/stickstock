// REST connector for external APIs. Unlike the SQL connectors, "query"
// here isn't SQL — it's a small JSON DSL (see restQuery) describing which
// endpoint to hit, since REST APIs don't have a native query language to
// pass through. The DSN is the API's base URL, optionally with headers:
// either a bare URL string, or JSON like
// {"base_url": "https://api.example.com", "headers": {"Authorization": "Bearer ..."}}.
package connectors

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

type RESTConnector struct {
	baseURL string
	client  *http.Client
	headers map[string]string
}

type restQuery struct {
	Path       string            `json:"path"`
	Method     string            `json:"method"`
	Query      map[string]string `json:"query"`
	ResultPath string            `json:"result_path"` // dot path to the array, if the response is a wrapped object
}

func NewRESTConnector(dsn string) (*RESTConnector, error) {
	baseURL := dsn
	var headers map[string]string

	trimmed := strings.TrimSpace(dsn)
	if strings.HasPrefix(trimmed, "{") {
		var cfg struct {
			BaseURL string            `json:"base_url"`
			Headers map[string]string `json:"headers"`
		}
		if err := json.Unmarshal([]byte(trimmed), &cfg); err != nil {
			return nil, fmt.Errorf("invalid REST connector config: %w", err)
		}
		baseURL = cfg.BaseURL
		headers = cfg.Headers
	}

	if baseURL == "" {
		return nil, fmt.Errorf("REST connector needs a base URL")
	}

	return &RESTConnector{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 20 * time.Second},
		headers: headers,
	}, nil
}

func (c *RESTConnector) TestConnection(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL, nil)
	if err != nil {
		return err
	}
	c.applyHeaders(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return fmt.Errorf("upstream returned %d", resp.StatusCode)
	}
	return nil
}

// ListSchemas isn't meaningful for a generic REST API without an OpenAPI
// spec to introspect — return an empty list so the caller can show "no
// browsable schema" and fall back to writing a query directly.
func (c *RESTConnector) ListSchemas(ctx context.Context) ([]TableInfo, error) {
	return nil, nil
}

// Query expects `query` to be the JSON restQuery DSL, not SQL. The
// queryengine SELECT-only guard doesn't apply to REST sources for this
// reason — see queryengine.ValidateReadOnly's callers, which only run
// for SQL-shaped connectors.
func (c *RESTConnector) Query(ctx context.Context, query string, args ...interface{}) (*Result, error) {
	var q restQuery
	if err := json.Unmarshal([]byte(query), &q); err != nil {
		return nil, fmt.Errorf(`REST query must be JSON like {"path":"/orders"}: %w`, err)
	}
	if q.Method == "" {
		q.Method = http.MethodGet
	}

	req, err := http.NewRequestWithContext(ctx, q.Method, c.baseURL+q.Path, nil)
	if err != nil {
		return nil, err
	}
	if len(q.Query) > 0 {
		qs := req.URL.Query()
		for k, v := range q.Query {
			qs.Set(k, v)
		}
		req.URL.RawQuery = qs.Encode()
	}
	c.applyHeaders(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20)) // 10MB cap
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("upstream returned %d: %s", resp.StatusCode, truncateForError(string(body), 200))
	}

	records, err := extractRecords(body, q.ResultPath)
	if err != nil {
		return nil, err
	}
	return recordsToResult(records), nil
}

func (c *RESTConnector) applyHeaders(req *http.Request) {
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
}

func (c *RESTConnector) Close() error {
	return nil
}

func truncateForError(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// extractRecords parses the response body as either a top-level JSON array
// of objects, or an object with the array nested at resultPath
// (dot-separated, e.g. "data.items").
func extractRecords(body []byte, resultPath string) ([]map[string]interface{}, error) {
	var raw interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("response was not valid JSON: %w", err)
	}

	if resultPath != "" {
		for _, key := range strings.Split(resultPath, ".") {
			obj, ok := raw.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("result_path %q does not match response shape", resultPath)
			}
			raw, ok = obj[key]
			if !ok {
				return nil, fmt.Errorf("result_path %q not found in response", resultPath)
			}
		}
	}

	arr, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("expected a JSON array at the result path, got something else")
	}

	records := make([]map[string]interface{}, 0, len(arr))
	for _, item := range arr {
		if obj, ok := item.(map[string]interface{}); ok {
			records = append(records, obj)
		}
		// non-object entries (scalars/nested arrays) are skipped rather
		// than failing the whole query.
	}
	return records, nil
}

// recordsToResult flattens a slice of JSON objects into a Result table.
// The column set is the union of keys across all records (sorted for
// determinism); nested objects/arrays are serialized back to a JSON
// string rather than being further flattened.
func recordsToResult(records []map[string]interface{}) *Result {
	colSet := map[string]struct{}{}
	for _, rec := range records {
		for k := range rec {
			colSet[k] = struct{}{}
		}
	}
	cols := make([]string, 0, len(colSet))
	for k := range colSet {
		cols = append(cols, k)
	}
	sort.Strings(cols)

	res := &Result{Columns: cols}
	for _, rec := range records {
		row := make(Row, len(cols))
		for i, c := range cols {
			v := rec[c]
			switch v.(type) {
			case map[string]interface{}, []interface{}:
				b, _ := json.Marshal(v)
				row[i] = string(b)
			default:
				row[i] = v
			}
		}
		res.Rows = append(res.Rows, row)
	}
	return res
}
