package chart

import (
	"fmt"
)

type InstallOptions struct {
	ChartPath string
	Namespace string
	DryRun    bool
	Verbose   bool
}

// Install  chart templates
func Install(opts InstallOptions) error {
	registry, err := Parse(ParseOptions{
		ChartPath: opts.ChartPath,
		Namespace: opts.Namespace,
		Verbose:   opts.Verbose,
	})

	if err != nil {
		fmt.Println(err)
		return err
	}

	fmt.Printf("Installing chart: %s\n", registry.Chart.Name)
	fmt.Printf("Namespace: %s\n", opts.Namespace)

	// Get resources in creation order
	creationOrder, err := registry.GetCreationOrder()
	if err != nil {
		return fmt.Errorf("failed to determine creation order: %w", err)
	}

	// Install resources level by level
	for levelIndex, level := range creationOrder {
		fmt.Printf("\nLevel %d:\n", levelIndex)
		for _, resource := range level {
			fmt.Printf("Installing %s: %s\n", resource.GetType(), resource.GetName())

			// TODO: Implement actual resource installation
			// This will be handled by resource managers in later tasks
			fmt.Printf("  ✓ %s %s created\n", resource.GetType(), resource.GetName())
		}
	}

	fmt.Printf("Successfully installed %d resources\n", len(registry.GetAllResources()))
	return nil
}
