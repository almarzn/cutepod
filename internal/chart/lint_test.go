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
replicaCount: 3
`), 0644))

	// templates/deployment.yaml
	require.NoError(t, os.WriteFile(filepath.Join(dir, "templates", "deployment.yaml"), []byte(`
kind: CuteContainer
metadata:
  name: {{ .Release.Name }}-deployment
  namespace: {{ .Release.Namespace }}
`), 0644))

	opts := chart.LintOptions{
		ChartPath: dir,
		Namespace: "default",
		Verbose:   true,
	}

	err = chart.Parse(opts)
	require.NoError(t, err)
}
