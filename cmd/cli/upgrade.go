package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var upgradeDryRun bool

// upgradeCmd represents the upgrade command
var upgradeCmd = &cobra.Command{
	Use:   "upgrade <namespace> <chart>",
	Short: "Reconcile and update containers (use --dry-run)",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		namespace, chart := args[0], args[1]
		fmt.Printf("upgrade called with namespace=%s chart=%s dry-run=%v\n", namespace, chart, upgradeDryRun)
	},
}

func init() {
	upgradeCmd.Flags().BoolVar(&upgradeDryRun, "dry-run", false, "Preview changes without applying them")

	rootCmd.AddCommand(upgradeCmd)
}
