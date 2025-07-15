package cli

import (
	"cutepod/internal/chart"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var upgradeDryRun bool
var upgradeVerbose bool

// upgradeCmd represents the upgrade command
var upgradeCmd = &cobra.Command{
	Use:   "upgrade <namespace> <chart>",
	Short: "Reconcile and update containers (use --dry-run)",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		namespace, path := args[0], args[1]
		if upgradeVerbose {
			fmt.Printf("upgrade called with namespace=%s path=%s dry-run=%v\n", namespace, path, installDryRun)
		}

		err := chart.Upgrade(chart.UpgradeOptions{
			ChartPath: path,
			Namespace: namespace,
			DryRun:    upgradeDryRun,
			Verbose:   upgradeVerbose,
		})

		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

func init() {
	upgradeCmd.Flags().BoolVar(&upgradeDryRun, "dry-run", false, "Preview changes without applying them")
	upgradeCmd.Flags().BoolVarP(&upgradeVerbose, "verbose", "v", false, "Verbose mode")

	rootCmd.AddCommand(upgradeCmd)
}
