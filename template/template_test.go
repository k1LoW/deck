package template

import (
	"testing"
)

func TestExpand(t *testing.T) {
	tests := []struct {
		name     string
		template string
		store    map[string]any
		expected string
		wantErr  bool
	}{
		{
			name:     "simple variable substitution",
			template: "Hello {{name}}!",
			store:    map[string]any{"name": "World"},
			expected: "Hello World!",
			wantErr:  false,
		},
		{
			name:     "multiple variables",
			template: "{{lang}} code: {{content}}",
			store:    map[string]any{"lang": "go", "content": "fmt.Println(\"hello\")"},
			expected: "go code: fmt.Println(\"hello\")",
			wantErr:  false,
		},
		{
			name:     "dot notation for nested maps",
			template: "Home directory: {{env.HOME}}",
			store: map[string]any{
				"env": map[string]any{"HOME": "/home/user"},
			},
			expected: "Home directory: /home/user",
			wantErr:  false,
		},
		{
			name:     "map[string]string access",
			template: "Path: {{env.PATH}}",
			store: map[string]any{
				"env": map[string]string{"PATH": "/usr/bin"},
			},
			expected: "Path: /usr/bin",
			wantErr:  false,
		},
		{
			name:     "integer values",
			template: "Count: {{count}}",
			store:    map[string]any{"count": 42},
			expected: "Count: 42",
			wantErr:  false,
		},
		{
			name:     "boolean values",
			template: "Enabled: {{enabled}}",
			store:    map[string]any{"enabled": true},
			expected: "Enabled: true",
			wantErr:  false,
		},
		{
			name:     "empty string value",
			template: "Value: {{empty}}",
			store:    map[string]any{"empty": ""},
			expected: "Value: ",
			wantErr:  false,
		},
		{
			name:     "no variables to expand",
			template: "No variables here",
			store:    map[string]any{"unused": "value"},
			expected: "No variables here",
			wantErr:  false,
		},
		{
			name:     "variable with spaces",
			template: "{{ name }}",
			store:    map[string]any{"name": "value"},
			expected: "value",
			wantErr:  false,
		},
		{
			name:     "CEL ternary operator - true condition",
			template: `{{lang == "" ? "md" : lang}}`,
			store:    map[string]any{"lang": ""},
			expected: "md",
			wantErr:  false,
		},
		{
			name:     "CEL ternary operator - false condition",
			template: `{{lang == "" ? "md" : lang}}`,
			store:    map[string]any{"lang": "go"},
			expected: "go",
			wantErr:  false,
		},
		{
			name:     "CEL arithmetic expression",
			template: "Result: {{count + 10}}",
			store:    map[string]any{"count": 32},
			expected: "Result: 42",
			wantErr:  false,
		},
		{
			name:     "CEL string concatenation",
			template: `{{prefix + " " + suffix}}`,
			store:    map[string]any{"prefix": "Hello", "suffix": "World"},
			expected: "Hello World",
			wantErr:  false,
		},
		{
			name:     "CEL logical expressions",
			template: "{{enabled && count > 0}}",
			store:    map[string]any{"enabled": true, "count": 5},
			expected: "true",
			wantErr:  false,
		},
		{
			name:     "CEL string functions",
			template: "{{name.size()}}",
			store:    map[string]any{"name": "test"},
			expected: "4",
			wantErr:  false,
		},
		{
			name:     "real-world code generation example",
			template: `convert {{lang == "" ? "md" : lang}} -o {{output}} < {{content}}`,
			store: map[string]any{
				"lang":    "go",
				"content": "main.go",
				"output":  "/tmp/out.png",
			},
			expected: "convert go -o /tmp/out.png < main.go",
			wantErr:  false,
		},
		{
			name:     "environment variables with ternary",
			template: `HOME={{env.HOME}} USER={{env.USER == "" ? "default" : env.USER}}`,
			store: map[string]any{
				"env": map[string]string{
					"HOME": "/home/testuser",
					"USER": "testuser",
				},
			},
			expected: "HOME=/home/testuser USER=testuser",
			wantErr:  false,
		},
		{
			name:     "complex nested expression",
			template: `{{config.database.host + ":" + string(config.database.port)}}`,
			store: map[string]any{
				"config": map[string]any{
					"database": map[string]any{
						"host": "localhost",
						"port": 5432,
					},
				},
			},
			expected: "localhost:5432",
			wantErr:  false,
		},
		{
			name:     "undefined variable",
			template: "{{undefined}}",
			store:    map[string]any{"other": "value"},
			expected: "",
			wantErr:  true,
		},
		{
			name:     "invalid CEL expression",
			template: "{{lang == }}",
			store:    map[string]any{"lang": "go"},
			expected: "",
			wantErr:  true,
		},
		{
			name:     "multiple CEL expressions in one template",
			template: `{{lang}} version {{version}} {{enabled ? "enabled" : "disabled"}}`,
			store: map[string]any{
				"lang":    "Go",
				"version": "1.21",
				"enabled": true,
			},
			expected: "Go version 1.21 enabled",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Expand(tt.template, tt.store)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expand() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Expand() unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("Expand() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestCreateCELEnv(t *testing.T) {
	store := map[string]any{
		"simple":  "value",
		"number":  42,
		"boolean": true,
		"env": map[string]string{
			"HOME": "/home/user",
			"PATH": "/usr/bin",
		},
		"config": map[string]any{
			"database": map[string]any{
				"host": "localhost",
				"port": 5432,
			},
		},
	}

	env, err := createCELEnv(store)
	if err != nil {
		t.Fatalf("createCELEnv() error = %v", err)
	}

	// Test that we can compile a simple expression
	_, issues := env.Compile(`simple + " test"`)
	if issues != nil && issues.Err() != nil {
		t.Errorf("Failed to compile expression: %v", issues.Err())
	}
}
