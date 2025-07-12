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
	Use:   "install <namespace> <chart>",
	Short: "Install containers (use --dry-run to preview)",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		namespace, path := args[0], args[1]

		fmt.Printf("install called with namespace=%s path=%s dry-run=%v\n", namespace, path, installDryRun)

		err := chart.Install(chart.InstallOptions{
			ChartPath: path,
			Namespace: namespace,
			DryRun:    false,
			Verbose:   false,
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
