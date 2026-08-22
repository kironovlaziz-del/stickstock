package queryengine

import (
	"testing"
)

func TestBindParams(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		params   map[string]interface{}
		wantSQL  string
		wantArgs []interface{}
		err      bool
	}{
		{
			name:     "single_param",
			sql:      "SELECT * FROM users WHERE id = :id",
			params:   map[string]interface{}{"id": 42},
			wantSQL:  "SELECT * FROM users WHERE id = $1",
			wantArgs: []interface{}{42},
			err:      false,
		},
		{
			name:     "two_params",
			sql:      "SELECT * FROM users WHERE age > :age AND name = :name",
			params:   map[string]interface{}{"age": 18, "name": "John"},
			wantSQL:  "SELECT * FROM users WHERE age > $1 AND name = $2",
			wantArgs: []interface{}{18, "John"},
			err:      false,
		},
		{
			name:     "type_cast_ignore",
			sql:      "SELECT ::text FROM users WHERE id = :id",
			params:   map[string]interface{}{"id": 42},
			wantSQL:  "SELECT ::text FROM users WHERE id = $1",
			wantArgs: []interface{}{42},
			err:      false,
		},
		{
			name:     "missing_param",
			sql:      "SELECT * FROM users WHERE id = :id",
			params:   map[string]interface{}{"wrong": "x"},
			wantSQL:  "",
			wantArgs: nil,
			err:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSQL, gotArgs, err := BindParams(tt.sql, tt.params)
			if (err != nil) != tt.err {
				t.Errorf("BindParams() error = %v, wantErr %v", err, tt.err)
				return
			}
			if !tt.err {
				if gotSQL != tt.wantSQL {
					t.Errorf("BindParams() SQL = %v, want %v", gotSQL, tt.wantSQL)
				}
				if len(gotArgs) != len(tt.wantArgs) {
					t.Errorf("BindParams() args len = %v, want %v", len(gotArgs), len(tt.wantArgs))
				} else {
					for i := range gotArgs {
						if gotArgs[i] != tt.wantArgs[i] {
							t.Errorf("BindParams() args[%d] = %v, want %v", i, gotArgs[i], tt.wantArgs[i])
						}
					}
				}
			}
		})
	}
}
