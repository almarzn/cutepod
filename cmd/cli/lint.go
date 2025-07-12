package cli

import (
	"cutepod/internal/chart"

	"github.com/spf13/cobra"
)

// lintCmd represents the lint command
func init() {
	var Verbose = false
	var lintCmd = &cobra.Command{
		Use:   "lint <path-to-chart>",
		Short: "Validate templates and YAML structure",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			chart.Lint(chart.LintOptions{
				ChartPath: args[0],
				Verbose:   Verbose,
				Namespace: "<namespace>",
			})
		},
	}

	lintCmd.Flags().BoolVarP(&Verbose, "verbose", "v", Verbose, "Verbose output")

	rootCmd.AddCommand(lintCmd)
}
