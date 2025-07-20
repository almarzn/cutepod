package chart

import (
	"bytes"
	"cutepod/internal/labels"
	"cutepod/internal/resource"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/goccy/go-yaml"
)

// ChartRegistry manages chart parsing and resource registry
type ChartRegistry struct {
	ChartPath string
	Chart     ChartMetadata
	Values    map[string]interface{}
	Registry  *resource.ManifestRegistry
}

// ChartMetadata represents the Chart.yaml metadata
type ChartMetadata struct {
	Name        string `yaml:"name"`
	Version     string `yaml:"version"`
	Description string `yaml:"description,omitempty"`
}

// NewChartRegistry creates a new chart registry
func NewChartRegistry(chartPath string) (*ChartRegistry, error) {
	registry := &ChartRegistry{
		ChartPath: chartPath,
		Registry:  resource.NewManifestRegistry(),
	}

	if err := registry.loadChartMetadata(); err != nil {
		return nil, err
	}

	if err := registry.loadValues(); err != nil {
		return nil, err
	}

	return registry, nil
}

// loadChartMetadata loads the Chart.yaml file
func (c *ChartRegistry) loadChartMetadata() error {
	chartYamlPath := filepath.Join(c.ChartPath, "Chart.yaml")
	chartBytes, err := os.ReadFile(chartYamlPath)
	if err != nil {
		return fmt.Errorf("failed to read Chart.yaml: %w", err)
	}

	if err := yaml.Unmarshal(chartBytes, &c.Chart); err != nil {
		return fmt.Errorf("failed to parse Chart.yaml: %w", err)
	}

	return nil
}

// loadValues loads the values.yaml file
func (c *ChartRegistry) loadValues() error {
	valuesPath := filepath.Join(c.ChartPath, "values.yaml")
	valuesBytes, err := os.ReadFile(valuesPath)
	if err != nil {
		return fmt.Errorf("failed to read values.yaml: %w", err)
	}

	if err := yaml.Unmarshal(valuesBytes, &c.Values); err != nil {
		return fmt.Errorf("failed to parse values.yaml: %w", err)
	}

	return nil
}

// ParseTemplates parses all template files and populates the registry
func (c *ChartRegistry) ParseTemplates(verbose bool) error {
	templatesDir := filepath.Join(c.ChartPath, "templates")

	// Build template context
	context := map[string]interface{}{
		"Values": c.Values,
		"Release": map[string]interface{}{
			"Name": c.Chart.Name,
		},
		"Chart": map[string]interface{}{
			"Name":    c.Chart.Name,
			"Version": c.Chart.Version,
		},
	}

	parser := resource.NewManifestParser()

	err := filepath.WalkDir(templatesDir, func(path string, d fs.DirEntry, err error) error {
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

		// Parse and execute template
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

		if verbose {
			fmt.Printf("# Source: %s\n", path)
			fmt.Println(buf.String())
		}

		// Parse the rendered YAML into resources
		if err := parser.ParseManifest(buf.Bytes()); err != nil {
			return fmt.Errorf("failed to parse rendered template %s: %w", path, err)
		}

		return nil
	})

	if err != nil {
		return err
	}

	// Get the populated registry from the parser
	c.Registry = parser.GetRegistry()

	// Apply labels to all resources
	if err := c.applyLabels(); err != nil {
		return err
	}

	// Validate dependencies
	if err := c.Registry.ValidateDependencies(); err != nil {
		return fmt.Errorf("dependency validation failed: %w", err)
	}

	return nil
}

// applyLabels applies standard labels to all resources
func (c *ChartRegistry) applyLabels() error {
	labels := labels.GetStandardLabels(c.Chart.Name, c.Chart.Version)

	for _, res := range c.Registry.GetAllResources() {
		// Merge with existing labels
		existingLabels := res.GetLabels()
		for k, v := range labels {
			existingLabels[k] = v
		}
		res.SetLabels(existingLabels)
	}

	return nil
}

// GetResourcesByType returns all resources of a specific type
func (c *ChartRegistry) GetResourcesByType(resourceType resource.ResourceType) []resource.Resource {
	return c.Registry.GetResourcesByType(resourceType)
}

// GetAllResources returns all resources in the registry
func (c *ChartRegistry) GetAllResources() []resource.Resource {
	return c.Registry.GetAllResources()
}

// GetCreationOrder returns resources in dependency order for creation
func (c *ChartRegistry) GetCreationOrder() ([][]resource.Resource, error) {
	return c.Registry.GetCreationOrder()
}

// GetDeletionOrder returns resources in reverse dependency order for deletion
func (c *ChartRegistry) GetDeletionOrder() ([][]resource.Resource, error) {
	return c.Registry.GetDeletionOrder()
}
