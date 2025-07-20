package chart

import (
	"cutepod/internal/resource"
	"fmt"
)

type ParseOptions struct {
	ChartPath string
	Verbose   bool
}

// Parse parses a chart using the new ManifestRegistry pattern
func Parse(opts ParseOptions) (*ChartRegistry, error) {
	registry, err := NewChartRegistry(opts.ChartPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create chart registry: %w", err)
	}

	if err := registry.ParseTemplates(opts.Verbose); err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	return registry, nil
}

// ParseToResources parses a chart and returns all resources
func ParseToResources(opts ParseOptions) ([]resource.Resource, error) {
	registry, err := Parse(opts)
	if err != nil {
		return nil, err
	}

	return registry.GetAllResources(), nil
}
