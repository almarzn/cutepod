package chart_test

import (
	"cutepod/internal/chart"
	"cutepod/internal/container"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-test/deep"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	// templates/deployment.yaml
	require.NoError(t, os.WriteFile(filepath.Join(dir, "templates", "deployment.yaml"), []byte(`
---
kind: CuteContainer
apiVersion: cutepod/v1alpha0
metadata:
  name: {{ .Release.Name }}-container
  namespace: {{ .Release.Namespace }}
spec:
  image: {{ .Values.Image}}
`), 0644))

	opts := chart.LintOptions{
		ChartPath: dir,
		Namespace: "default",
		Verbose:   true,
	}

	r, err := chart.Parse(opts)
	require.NoError(t, err)

	if diff := deep.Equal(map[string]interface{}{
		"testchart-container": &container.CuteContainer{
			TypeMeta: metav1.TypeMeta{
				Kind:       "CuteContainer",
				APIVersion: "cutepod/v1alpha0",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testchart-container",
				Namespace: "default",
			},
			Spec: container.CuteContainerSpec{
				Image: "ubuntu",
			},
		},
	}, r); diff != nil {
		t.Error(diff)
	}
}
