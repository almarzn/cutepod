package chart

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLintChart(t *testing.T) {
	t.Log("starting TestLintChart")
	tests := []struct {
		name        string
		values      string
		templates   map[string]string
		wantErr     bool
		errContains string
	}{
		{
			name:   "valid chart",
			values: "foo: bar\n",
			templates: map[string]string{
				"tpl.yaml": "value: {{ .foo }}\n",
			},
		},
		{
			name:        "invalid values",
			values:      ": bad\n",
			templates:   map[string]string{},
			wantErr:     true,
			errContains: "failed to parse values.yaml",
		},
		{
			name:   "invalid template executes",
			values: "foo: bar\n",
			templates: map[string]string{
				"tpl.yaml": "value: {{ .baz }}\n",
			},
			wantErr:     true,
			errContains: "failed to execute template",
		},
		{
			name:   "invalid yaml after render",
			values: "foo: bar\n",
			templates: map[string]string{
				"tpl.yaml": "value: {{ .foo }}\n:bad\n",
			},
			wantErr:     true,
			errContains: "invalid YAML",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			dir, err := os.MkdirTemp("", "charttest")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(dir)
			// write values.yaml
			if err := os.WriteFile(filepath.Join(dir, "values.yaml"), []byte(tt.values), 0644); err != nil {
				t.Fatalf("failed to write values.yaml: %v", err)
			}
			// write templates
			templDir := filepath.Join(dir, "templates")
			if err := os.Mkdir(templDir, 0755); err != nil {
				t.Fatalf("failed to create templates dir: %v", err)
			}
			for name, content := range tt.templates {
				if err := os.WriteFile(filepath.Join(templDir, name), []byte(content), 0644); err != nil {
					t.Fatalf("failed to write template %s: %v", name, err)
				}
			}
			err = LintChart(dir)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error but got nil")
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf("expected error to contain %q, got %v", tt.errContains, err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
