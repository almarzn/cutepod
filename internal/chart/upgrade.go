package chart

import (
	"context"
	"cutepod/internal/object"
	"fmt"
)

type UpgradeOptions struct {
	ChartPath string
	Namespace string
	DryRun    bool
	Verbose   bool
}

// Upgrade  chart templates
func Upgrade(opts UpgradeOptions) error {
	charts, err := Parse(ParseOptions{
		ChartPath: opts.ChartPath,
		Namespace: opts.Namespace,
		Verbose:   opts.Verbose,
	})

	if err != nil {
		fmt.Println(err)
		return err
	}

	installTarget := object.NewInstallTarget(opts.Namespace)

	counter := 0

	for name, chart := range charts {
		changes, err := chart.ComputeChanges(context.Background(), *installTarget)

		if err != nil {
			return fmt.Errorf("error in ComputeChanges: %s", err)
		}

		if changes == nil || len(changes) == 0 {
			if opts.Verbose {
				fmt.Printf("🚫 no changes detected for %s, skipping.\n", name)
			}
			continue
		}
		fmt.Printf("🔄️Changes detected for %s, upgrading...\n", name)

		counter++

		if opts.Verbose {
			fmt.Printf("  detected changes: %v\n", changes)
		}

		if !opts.DryRun {
			err = chart.Uninstall(context.Background(), *installTarget)
			if err != nil {
				return fmt.Errorf("error in Uninstall: %s", err)
			}

			err = chart.Install(context.Background(), *installTarget)
			if err != nil {
				return fmt.Errorf("error in Install: %s", err)
			}
		}

	}
	fmt.Printf("✅ Completed. Upgraded %d objects\n", counter)
	return nil
}
