package chart_test

import (
	"cutepod/internal/chart"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParse_ValidChart(t *testing.T) {
	dir := t.TempDir()

	// Create minimal Helm chart structure
	err := os.MkdirAll(filepath.Join(dir, "templates"), 0755)
	require.NoError(t, err)

	// Chart.yaml
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Chart.yaml"), []byte(`
name: testchart
version: 0.1.0
`), 0644))

	// values.yaml
	require.NoError(t, os.WriteFile(filepath.Join(dir, "values.yaml"), []byte(`
Image: ubuntu
`), 0644))

	// templates/deployment.yaml (note: removed namespace from template)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "templates", "deployment.yaml"), []byte(`
---
kind: CuteContainer
apiVersion: cutepod/v1alpha0
metadata:
  name: {{ .Release.Name }}-container
spec:
  image: {{ .Values.Image}}
`), 0644))

	opts := chart.ParseOptions{
		ChartPath: dir,
		Namespace: "default",
		Verbose:   true,
	}

	registry, err := chart.Parse(opts)
	require.NoError(t, err)

	// Verify chart metadata
	require.Equal(t, "testchart", registry.Chart.Name)
	require.Equal(t, "0.1.0", registry.Chart.Version)

	// Get all resources
	resources := registry.GetAllResources()
	require.Len(t, resources, 1)

	// Verify the container resource
	resource := resources[0]
	require.Equal(t, "testchart-container", resource.GetName())
	require.Equal(t, "default", resource.GetNamespace())

	// Verify labels were applied
	labels := resource.GetLabels()
	require.Equal(t, "default", labels["cutepod.io/namespace"])
	require.Equal(t, "testchart", labels["cutepod.io/chart"])
	require.Equal(t, "0.1.0", labels["cutepod.io/version"])
	require.Equal(t, "cutepod", labels["cutepod.io/managed-by"])
}
