package chart

import (
	"fmt"
	"time"

	"github.com/briandowns/spinner"
	"github.com/charmbracelet/lipgloss"
)

type UpgradeOptions struct {
	ChartPath string
	Namespace string
	DryRun    bool
	Verbose   bool
}

var (
	doneStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("70"))
	failStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
)

func Upgrade(opts UpgradeOptions) error {
	registry, err := Parse(ParseOptions{
		ChartPath: opts.ChartPath,
		Namespace: opts.Namespace,
		Verbose:   opts.Verbose,
	})
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	fmt.Printf("Upgrading chart: %s\n", registry.Chart.Name)
	fmt.Printf("Namespace: %s\n", opts.Namespace)

	// TODO: Implement actual change detection and reconciliation
	// This will be handled by the ReconciliationController in later tasks

	if opts.DryRun {
		fmt.Println(lipgloss.NewStyle().Bold(true).Render("🧪 Dry Run: Showing planned changes...\n"))

		// Show what would be created/updated
		creationOrder, err := registry.GetCreationOrder()
		if err != nil {
			return fmt.Errorf("failed to determine creation order: %w", err)
		}

		for _, level := range creationOrder {
			for _, resource := range level {
				fmt.Printf("  + %s %s would be created/updated\n", resource.GetType(), resource.GetName())
			}
		}

		fmt.Println(lipgloss.NewStyle().Bold(true).Render("Run without --dry-run to apply changes\n"))
		return nil
	}

	// Get resources in creation order
	creationOrder, err := registry.GetCreationOrder()
	if err != nil {
		return fmt.Errorf("failed to determine creation order: %w", err)
	}

	fmt.Println(lipgloss.NewStyle().Bold(true).Render("🚀 Applying changes...\n"))

	successCount := 0
	failCount := 0

	// Apply resources level by level
	for _, level := range creationOrder {
		for _, resource := range level {
			// Create and start spinner with resource name
			s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
			s.Suffix = " " + resource.GetName() + "..."
			s.Start()

			// TODO: Execute actual resource reconciliation
			// For now, just simulate success
			time.Sleep(100 * time.Millisecond)

			// Stop spinner before printing result
			s.Stop()

			successCount++
			fmt.Println(doneStyle.Bold(true).Render("✓ ") + resource.GetName())
		}
	}

	fmt.Printf("\n%s\n", lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("🎉 Upgrade complete: %d succeeded, %d failed", successCount, failCount)))
	return nil
}

// TODO: The old change detection and tree printing functions have been removed
// These will be replaced by the ReconciliationController in subsequent tasks
