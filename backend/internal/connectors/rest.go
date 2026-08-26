package connectors

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
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
	ResultPath string            `json:"result_path"`
}

// allowedHostsFromEnv reads REST_ALLOWED_HOSTS (comma-separated) from env.
// Default: "demo-rest" (for demo purposes) – in production, admin should set this.
func allowedHosts() map[string]bool {
	allowed := map[string]bool{
		"demo-rest": true, // always allow for demo
	}
	if env := os.Getenv("REST_ALLOWED_HOSTS"); env != "" {
		for _, h := range strings.Split(env, ",") {
			h = strings.TrimSpace(h)
			if h != "" {
				allowed[h] = true
			}
		}
	}
	return allowed
}

func isPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return true
	}
	privateIPv4Blocks := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"169.254.0.0/16",
	}
	for _, cidr := range privateIPv4Blocks {
		_, block, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if block.Contains(ip) {
			return true
		}
	}
	return false
}

func validateRESTURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("only HTTP/HTTPS schemes are allowed")
	}

	host := u.Hostname()
	// Check if host is in the allowed list (bypass SSRF check)
	allowed := allowedHosts()
	if allowed[host] {
		return nil
	}

	// Resolve and check IP
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("cannot resolve host: %w", err)
	}
	for _, ip := range ips {
		if isPrivateIP(ip) {
			return fmt.Errorf("private/internal IP not allowed: %s", ip.String())
		}
	}

	// Additional block for internal Docker services (forbidden by name)
	forbiddenHosts := []string{"backend", "analytics-service", "nginx", "frontend", "localhost", "127.0.0.1"}
	for _, forbidden := range forbiddenHosts {
		if strings.Contains(strings.ToLower(host), forbidden) {
			return fmt.Errorf("hostname %q is forbidden", host)
		}
	}
	return nil
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

	if err := validateRESTURL(baseURL); err != nil {
		return nil, fmt.Errorf("URL validation failed: %w", err)
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

func (c *RESTConnector) ListSchemas(ctx context.Context) ([]TableInfo, error) {
	return nil, nil
}

func (c *RESTConnector) Query(ctx context.Context, query string, args ...interface{}) (*Result, error) {
	var q restQuery
	if err := json.Unmarshal([]byte(query), &q); err != nil {
		return nil, fmt.Errorf(`REST query must be JSON like {"path":"/orders"}: %w`, err)
	}
	if q.Method == "" {
		q.Method = http.MethodGet
	}

	fullURL := c.baseURL + q.Path
	if err := validateRESTURL(fullURL); err != nil {
		return nil, fmt.Errorf("target URL validation failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, q.Method, fullURL, nil)
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

	// Prevent redirect to forbidden hosts
	redirectChecker := func(req *http.Request, via []*http.Request) error {
		if err := validateRESTURL(req.URL.String()); err != nil {
			return fmt.Errorf("redirect blocked: %w", err)
		}
		if len(via) >= 10 {
			return fmt.Errorf("too many redirects")
		}
		return nil
	}
	client := &http.Client{
		Timeout:       20 * time.Second,
		CheckRedirect: redirectChecker,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
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
	}
	return records, nil
}

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
