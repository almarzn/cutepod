package chart

import (
	"context"
	"cutepod/internal/resource"
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
)

type InstallOptions struct {
	ChartPath string
	DryRun    bool
	Verbose   bool
}

var (
	installDoneStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("70"))
	installFailStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	installWarnStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
)

// Install chart templates
func Install(opts InstallOptions) error {
	// Parse the chart and get resources
	registry, err := Parse(ParseOptions{
		ChartPath: opts.ChartPath,
		Verbose:   opts.Verbose,
	})
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	fmt.Printf("Installing chart: %s\n", registry.Chart.Name)

	// Create reconciliation controller with Podman client and registry
	controller := resource.NewReconciliationControllerWithURIAndRegistry(resource.GetPodmanURI(), registry.Registry)

	// Get all resources from the registry
	manifests := registry.GetAllResources()

	// Execute reconciliation (install is just reconciliation with empty current state)
	ctx := context.Background()
	result, err := controller.Reconcile(ctx, manifests, registry.Chart.Name, opts.DryRun)
	if err != nil {
		return fmt.Errorf("installation failed: %w", err)
	}

	// Display results
	displayInstallationResult(result, opts.DryRun)

	// Return error if there were any non-recoverable errors
	for _, reconciliationError := range result.Errors {
		if !reconciliationError.Recoverable {
			return fmt.Errorf("installation failed with non-recoverable errors")
		}
	}

	return nil
}

// displayInstallationResult displays the results of installation in a user-friendly format
func displayInstallationResult(result *resource.ReconciliationResult, dryRun bool) {
	if dryRun {
		displayInstallDryRunResult(result)
	} else {
		displayInstallExecutionResult(result)
	}
}

// displayInstallDryRunResult displays dry run results for installation
func displayInstallDryRunResult(result *resource.ReconciliationResult) {
	fmt.Println(lipgloss.NewStyle().Bold(true).Render("🧪 Dry Run: Showing planned installation...\n"))

	if len(result.CreatedResources) > 0 {
		fmt.Println(lipgloss.NewStyle().Bold(true).Render("Resources to be created:"))
		for _, action := range result.CreatedResources {
			fmt.Printf("  + %s %s\n", action.Type, action.Name)
		}
		fmt.Println()
	}

	if len(result.Errors) > 0 {
		fmt.Println(installFailStyle.Bold(true).Render("Potential issues:"))
		for _, err := range result.Errors {
			fmt.Printf("  ! %s: %s\n", err.Resource.Name, err.Message)
		}
		fmt.Println()
	}

	fmt.Println(lipgloss.NewStyle().Bold(true).Render("Run without --dry-run to install"))
}

// displayInstallExecutionResult displays actual installation results
func displayInstallExecutionResult(result *resource.ReconciliationResult) {
	fmt.Println(lipgloss.NewStyle().Bold(true).Render("🚀 Installation Results\n"))

	successCount := 0

	if len(result.CreatedResources) > 0 {
		fmt.Println(lipgloss.NewStyle().Bold(true).Render("Installed:"))
		for _, action := range result.CreatedResources {
			if action.Error == "" {
				fmt.Printf("  %s %s %s\n", installDoneStyle.Render("✓"), action.Type, action.Name)
				successCount++
			} else {
				fmt.Printf("  %s %s %s - %s\n", installFailStyle.Render("✗"), action.Type, action.Name, action.Error)
			}
		}
		fmt.Println()
	}

	// Display any updates (shouldn't happen in fresh install, but could occur)
	if len(result.UpdatedResources) > 0 {
		fmt.Println(lipgloss.NewStyle().Bold(true).Render("Updated:"))
		for _, action := range result.UpdatedResources {
			if action.Error == "" {
				fmt.Printf("  %s %s %s\n", installDoneStyle.Render("✓"), action.Type, action.Name)
				successCount++
			} else {
				fmt.Printf("  %s %s %s - %s\n", installFailStyle.Render("✗"), action.Type, action.Name, action.Error)
			}
		}
		fmt.Println()
	}

	// Display errors
	if len(result.Errors) > 0 {
		fmt.Println(installFailStyle.Bold(true).Render("Errors:"))
		for _, err := range result.Errors {
			if err.Recoverable {
				fmt.Printf("  %s %s: %s\n", installWarnStyle.Render("⚠"), err.Resource.Name, err.Message)
			} else {
				fmt.Printf("  %s %s: %s\n", installFailStyle.Render("✗"), err.Resource.Name, err.Message)
			}
		}
		fmt.Println()
	}

	// Display summary
	totalResources := len(result.CreatedResources) + len(result.UpdatedResources)
	errorCount := len(result.Errors)

	if errorCount == 0 {
		fmt.Printf("%s\n", installDoneStyle.Bold(true).Render(fmt.Sprintf("🎉 Installation complete: %d resources installed in %v",
			totalResources, result.Duration.Round(time.Millisecond))))
	} else {
		fmt.Printf("%s\n", installWarnStyle.Bold(true).Render(fmt.Sprintf("⚠ Installation completed with issues: %d succeeded, %d errors in %v",
			successCount, errorCount, result.Duration.Round(time.Millisecond))))
	}

	fmt.Println(result.Summary)
}
