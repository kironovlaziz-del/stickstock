package queryengine

import "testing"

func TestValidateReadOnly(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want bool // true = ok, false = error expected
	}{
		{"simple_select", "SELECT * FROM users", true},
		{"with_cte", "WITH t AS (SELECT 1) SELECT * FROM t", true},
		{"insert", "INSERT INTO users VALUES (1)", false},
		{"update", "UPDATE users SET name='x'", false},
		{"delete", "DELETE FROM users", false},
		{"drop", "DROP TABLE users", false},
		{"multiple_statements", "SELECT 1; SELECT 2", false},
		{"empty", "", false},
		{"with_keyword_in_string", "SELECT * FROM users WHERE status = 'created'", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateReadOnly(tt.sql)
			if (err == nil) != tt.want {
				t.Errorf("ValidateReadOnly() error = %v, wantOk = %v", err, tt.want)
			}
		})
	}
}

func TestValidateFileScope(t *testing.T) {
	tests := []struct {
		name         string
		sql          string
		allowedTable string
		want         bool
	}{
		{"valid_upload_table", "SELECT * FROM uploads.t_abc", "uploads.t_abc", true},
		{"public_table", "SELECT * FROM public.users", "uploads.t_abc", false},
		{"no_allowed_table", "SELECT * FROM other", "uploads.t_abc", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFileScope(tt.sql, tt.allowedTable)
			if (err == nil) != tt.want {
				t.Errorf("ValidateFileScope() error = %v, wantOk = %v", err, tt.want)
			}
		})
	}
}
