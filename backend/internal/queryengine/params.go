package queryengine

import (
	"fmt"
	"regexp"
	"strings"
)

// namedParamPattern matches `:name` placeholders while also matching
// `::name` so the replace func can recognize and skip Postgres type casts
// (a single leading colon is a param; a double colon is a cast).
var namedParamPattern = regexp.MustCompile(`::?[a-zA-Z_][a-zA-Z0-9_]*`)

// BindParams rewrites `:name` placeholders in sqlText into Postgres
// positional placeholders ($1, $2, ...) in first-appearance order, and
// returns the matching args slice ready to pass to db.QueryContext.
//
// Returns an error if the SQL references a name that's missing from
// params — better to fail loudly than silently run with a NULL.
func BindParams(sqlText string, params map[string]interface{}) (string, []interface{}, error) {
	var order []string
	seen := map[string]int{} // param name -> 1-based positional index
	var missing string

	rewritten := namedParamPattern.ReplaceAllStringFunc(sqlText, func(match string) string {
		if strings.HasPrefix(match, "::") {
			return match // type cast, e.g. value::text — leave untouched
		}
		name := strings.TrimPrefix(match, ":")
		idx, ok := seen[name]
		if !ok {
			if _, provided := params[name]; !provided {
				missing = name
				return match
			}
			order = append(order, name)
			idx = len(order)
			seen[name] = idx
		}
		return fmt.Sprintf("$%d", idx)
	})

	if missing != "" {
		return "", nil, fmt.Errorf("missing value for parameter %q", missing)
	}

	args := make([]interface{}, len(order))
	for i, name := range order {
		args[i] = params[name]
	}

	return rewritten, args, nil
}
