package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var installDryRun bool

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install <namespace> <chart>",
	Short: "Install containers (use --dry-run to preview)",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		namespace, chart := args[0], args[1]
		fmt.Printf("install called with namespace=%s chart=%s dry-run=%v\n", namespace, chart, installDryRun)
	},
}

func init() {
	installCmd.Flags().BoolVar(&installDryRun, "dry-run", false, "Preview changes without applying them")

	rootCmd.AddCommand(installCmd)
}
