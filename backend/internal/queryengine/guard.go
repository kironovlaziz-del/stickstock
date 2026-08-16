// Package queryengine holds logic shared by every "run this SQL" code
// path: the ad-hoc query runner, saved query execution, and (later) the
// visual query builder's generated SQL.
package queryengine

import (
	"fmt"
	"regexp"
	"strings"
)

var forbiddenKeyword = regexp.MustCompile(`(?i)\b(insert|update|delete|drop|alter|truncate|grant|revoke|create|exec|execute|call|copy|vacuum|merge|replace|lock)\b`)

// ValidateReadOnly rejects anything that isn't a single read-only SELECT
// (optionally starting with WITH for CTEs).
//
// This is defense-in-depth, not a substitute for proper isolation: in
// production, data source DSNs should point at a read-only DB role/user
// so a gap in this check can't turn into a write. The keyword blacklist
// below can also false-positive on a string literal that happens to
// contain one of these words (e.g. WHERE status = 'created') — reject
// on suspicion rather than trying to be clever about literals.
func ValidateReadOnly(sqlText string) error {
	trimmed := strings.TrimSpace(sqlText)
	if trimmed == "" {
		return fmt.Errorf("query is empty")
	}

	// Disallow stacked statements: strip at most one trailing semicolon,
	// then reject if another one remains anywhere in the body.
	body := strings.TrimSpace(strings.TrimSuffix(trimmed, ";"))
	if strings.Contains(body, ";") {
		return fmt.Errorf("only a single statement is allowed")
	}

	lower := strings.ToLower(body)
	if !strings.HasPrefix(lower, "select") && !strings.HasPrefix(lower, "with") {
		return fmt.Errorf("only SELECT statements are allowed")
	}

	if forbiddenKeyword.MatchString(body) {
		return fmt.Errorf("query contains a disallowed keyword")
	}

	return nil
}

// ValidateFileScope is an extra check applied only to "file" data sources
// (CSV uploads materialized as tables in the "uploads" Postgres schema —
// see migrations/0002_uploads.sql). File sources share the app's own
// Postgres connection (the DSN is the app's DATABASE_URL, not a separate
// credential), so this stops a query from reaching outside the uploads
// schema into app metadata tables like "profiles" or "data_sources".
//
// This is a substring check, not a parser — good enough given upload
// table names are random (see handlers.randomSuffix), but not a
// substitute for a properly scoped DB role if that's ever worth the
// added ops complexity (see README notes).
func ValidateFileScope(sqlText, allowedTable string) error {
	lower := strings.ToLower(sqlText)
	if strings.Contains(lower, "public.") {
		return fmt.Errorf("file-based queries cannot reference the public schema")
	}
	if allowedTable == "" || !strings.Contains(lower, strings.ToLower(allowedTable)) {
		return fmt.Errorf("query must reference table %s", allowedTable)
	}
	return nil
}
