package labels

import "fmt"

// Standard labels used for resource tracking and management
const (
	// LabelChart identifies the chart that created the resource
	LabelChart = "cutepod.io/chart"

	// LabelVersion identifies the version of the chart
	LabelVersion = "cutepod.io/version"

	// LabelManagedBy identifies the tool managing the resource
	LabelManagedBy = "cutepod.io/managed-by"

	// ManagedByValue is the value used for the managed-by label
	ManagedByValue = "cutepod-v1"
)

// GetStandardLabels returns the standard labels for a resource
func GetStandardLabels(chart, version string) map[string]string {
	return map[string]string{
		LabelChart:     chart,
		LabelVersion:   version,
		LabelManagedBy: ManagedByValue,
	}
}

// MergeLabels merges additional labels with standard labels
func MergeLabels(standardLabels, additionalLabels map[string]string) map[string]string {
	merged := make(map[string]string)

	// Copy standard labels first
	for k, v := range standardLabels {
		merged[k] = v
	}

	// Add additional labels (they can override standard labels if needed)
	for k, v := range additionalLabels {
		merged[k] = v
	}

	return merged
}

func GetChartLabelValue(name string) string {
	return fmt.Sprintf("%s=%s", LabelChart, name)
}
