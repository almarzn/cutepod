package chart

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"text/template"

	"gopkg.in/yaml.v3"
)

// LintChart parses and renders chart templates and validates resulting YAML.
func LintChart(chartPath string) error {
	valuesPath := filepath.Join(chartPath, "values.yaml")
	valuesBytes, err := os.ReadFile(valuesPath)
	if err != nil {
		return fmt.Errorf("failed to read values.yaml: %w", err)
	}
	var values map[string]interface{}
	if err := yaml.Unmarshal(valuesBytes, &values); err != nil {
		return fmt.Errorf("failed to parse values.yaml: %w", err)
	}

	templatesDir := filepath.Join(chartPath, "templates")
	return filepath.WalkDir(templatesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", path, err)
		}
		tmpl, err := template.New(filepath.Base(path)).Parse(string(content))
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", path, err)
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, values); err != nil {
			return fmt.Errorf("failed to execute template %s: %w", path, err)
		}
		decoder := yaml.NewDecoder(bytes.NewReader(buf.Bytes()))
		for {
			var doc interface{}
			if err := decoder.Decode(&doc); err != nil {
				if err == io.EOF {
					break
				}
				return fmt.Errorf("invalid YAML in template %s: %w", path, err)
			}
		}
		return nil
	})
}
