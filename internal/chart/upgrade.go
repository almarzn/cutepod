package chart

import (
	"context"
	"cutepod/internal/resource"
	"fmt"
	"time"

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
	warnStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
)

func Upgrade(opts UpgradeOptions) error {
	// Parse the chart and get resources
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

	// Create reconciliation controller with Podman client
	controller := resource.NewReconciliationControllerWithURI(resource.GetPodmanURI())

	// Get all resources from the registry
	manifests := registry.GetAllResources()

	// Execute reconciliation
	ctx := context.Background()
	result, err := controller.Reconcile(ctx, manifests, opts.Namespace, opts.DryRun)
	if err != nil {
		return fmt.Errorf("reconciliation failed: %w", err)
	}

	// Display results
	displayReconciliationResult(result, opts.DryRun)

	// Return error if there were any non-recoverable errors
	for _, reconciliationError := range result.Errors {
		if !reconciliationError.Recoverable {
			return fmt.Errorf("reconciliation failed with non-recoverable errors")
		}
	}

	return nil
}

// displayReconciliationResult displays the results of reconciliation in a user-friendly format
func displayReconciliationResult(result *resource.ReconciliationResult, dryRun bool) {
	if dryRun {
		displayDryRunResult(result)
	} else {
		displayExecutionResult(result)
	}
}

// displayDryRunResult displays dry run results
func displayDryRunResult(result *resource.ReconciliationResult) {
	fmt.Println(lipgloss.NewStyle().Bold(true).Render("🧪 Dry Run: Showing planned changes...\n"))

	// Group actions by type
	if len(result.CreatedResources) > 0 {
		fmt.Println(lipgloss.NewStyle().Bold(true).Render("Resources to be created:"))
		for _, action := range result.CreatedResources {
			fmt.Printf("  + %s %s\n", action.Type, action.Name)
		}
		fmt.Println()
	}

	if len(result.UpdatedResources) > 0 {
		fmt.Println(lipgloss.NewStyle().Bold(true).Render("Resources to be updated:"))
		for _, action := range result.UpdatedResources {
			fmt.Printf("  ~ %s %s\n", action.Type, action.Name)
		}
		fmt.Println()
	}

	if len(result.DeletedResources) > 0 {
		fmt.Println(lipgloss.NewStyle().Bold(true).Render("Resources to be deleted:"))
		for _, action := range result.DeletedResources {
			fmt.Printf("  - %s %s\n", action.Type, action.Name)
		}
		fmt.Println()
	}

	if len(result.Errors) > 0 {
		fmt.Println(failStyle.Bold(true).Render("Potential issues:"))
		for _, err := range result.Errors {
			fmt.Printf("  ! %s: %s\n", err.Resource.Name, err.Message)
		}
		fmt.Println()
	}

	fmt.Println(lipgloss.NewStyle().Bold(true).Render("Run without --dry-run to apply changes"))
}

// displayExecutionResult displays actual execution results
func displayExecutionResult(result *resource.ReconciliationResult) {
	fmt.Println(lipgloss.NewStyle().Bold(true).Render("🚀 Reconciliation Results\n"))

	// Display successful actions
	successCount := 0

	if len(result.CreatedResources) > 0 {
		fmt.Println(lipgloss.NewStyle().Bold(true).Render("Created:"))
		for _, action := range result.CreatedResources {
			if action.Error == "" {
				fmt.Printf("  %s %s %s\n", doneStyle.Render("✓"), action.Type, action.Name)
				successCount++
			} else {
				fmt.Printf("  %s %s %s - %s\n", failStyle.Render("✗"), action.Type, action.Name, action.Error)
			}
		}
		fmt.Println()
	}

	if len(result.UpdatedResources) > 0 {
		fmt.Println(lipgloss.NewStyle().Bold(true).Render("Updated:"))
		for _, action := range result.UpdatedResources {
			if action.Error == "" {
				fmt.Printf("  %s %s %s\n", doneStyle.Render("✓"), action.Type, action.Name)
				successCount++
			} else {
				fmt.Printf("  %s %s %s - %s\n", failStyle.Render("✗"), action.Type, action.Name, action.Error)
			}
		}
		fmt.Println()
	}

	if len(result.DeletedResources) > 0 {
		fmt.Println(lipgloss.NewStyle().Bold(true).Render("Deleted:"))
		for _, action := range result.DeletedResources {
			if action.Error == "" {
				fmt.Printf("  %s %s %s\n", doneStyle.Render("✓"), action.Type, action.Name)
				successCount++
			} else {
				fmt.Printf("  %s %s %s - %s\n", failStyle.Render("✗"), action.Type, action.Name, action.Error)
			}
		}
		fmt.Println()
	}

	// Display errors
	if len(result.Errors) > 0 {
		fmt.Println(failStyle.Bold(true).Render("Errors:"))
		for _, err := range result.Errors {
			if err.Recoverable {
				fmt.Printf("  %s %s: %s\n", warnStyle.Render("⚠"), err.Resource.Name, err.Message)
			} else {
				fmt.Printf("  %s %s: %s\n", failStyle.Render("✗"), err.Resource.Name, err.Message)
			}
		}
		fmt.Println()
	}

	// Display summary
	totalActions := len(result.CreatedResources) + len(result.UpdatedResources) + len(result.DeletedResources)
	errorCount := len(result.Errors)

	if errorCount == 0 {
		fmt.Printf("%s\n", doneStyle.Bold(true).Render(fmt.Sprintf("🎉 Reconciliation complete: %d actions succeeded in %v",
			totalActions, result.Duration.Round(time.Millisecond))))
	} else {
		fmt.Printf("%s\n", warnStyle.Bold(true).Render(fmt.Sprintf("⚠ Reconciliation completed with issues: %d succeeded, %d errors in %v",
			successCount, errorCount, result.Duration.Round(time.Millisecond))))
	}

	fmt.Println(result.Summary)
}
