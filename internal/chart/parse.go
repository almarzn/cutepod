package chart

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/goccy/go-yaml"
)

type LintOptions struct {
	ChartPath string
	Verbose   bool
	Namespace string
}

func Parse(opts LintOptions) error {
	chartYamlPath := filepath.Join(opts.ChartPath, "Chart.yaml")
	chartBytes, err := os.ReadFile(chartYamlPath)
	if err != nil {
		return fmt.Errorf("failed to read Chart.yaml: %w", err)
	}

	var chart struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version"`
	}
	if err := yaml.Unmarshal(chartBytes, &chart); err != nil {
		return fmt.Errorf("failed to parse Chart.yaml: \n\n%s", yaml.FormatError(err, true, true))
	}
	releaseName := chart.Name

	valuesPath := filepath.Join(opts.ChartPath, "values.yaml")
	valuesBytes, err := os.ReadFile(valuesPath)
	if err != nil {
		return fmt.Errorf("failed to read values.yaml: %w", err)
	}
	var values map[string]interface{}
	if err := yaml.Unmarshal(valuesBytes, &values); err != nil {
		return fmt.Errorf("failed to parse values.yaml: \n\n%s", yaml.FormatError(err, true, true))
	}

	// Build template context (simulate Helm)
	context := map[string]interface{}{
		"Values": values,
		"Release": map[string]interface{}{
			"Name":      releaseName,
			"Namespace": opts.Namespace,
		},
		"Chart": map[string]interface{}{
			"Name":    chart.Name,
			"Version": chart.Version,
		},
	}

	templatesDir := filepath.Join(opts.ChartPath, "templates")
	return filepath.WalkDir(templatesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		if ext != ".yaml" && ext != ".yml" && ext != ".tpl" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", path, err)
		}

		tmpl, err := template.
			New(filepath.Base(path)).
			Option("missingkey=error").
			Funcs(sprig.TxtFuncMap()).
			Parse(string(content))
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", path, err)
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, context); err != nil {
			return fmt.Errorf("failed to render template %s: %w", path, err)
		}

		if opts.Verbose {
			fmt.Printf("# Source: %s\n", path)
			fmt.Println(buf.String())
		}

		// Validate YAML
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
