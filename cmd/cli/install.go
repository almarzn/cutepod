package cli

import (
	"cutepod/internal/chart"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var installDryRun bool
var installVerbose bool

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install <chart>",
	Short: "Install containers (use --dry-run to preview)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		chartPath := args[0]

		fmt.Printf("install called with chart=%s dry-run=%v\n", chartPath, installDryRun)

		err := chart.Install(chart.InstallOptions{
			ChartPath: chartPath,
			DryRun:    installDryRun,
			Verbose:   installVerbose,
		})

		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

func init() {
	installCmd.Flags().BoolVar(&installDryRun, "dry-run", false, "Preview changes without applying them")
	installCmd.Flags().BoolVarP(&installVerbose, "verbose", "v", false, "Verbose mode")

	rootCmd.AddCommand(installCmd)
}
