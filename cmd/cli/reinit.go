package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// reinitCmd represents the reinit command
var reinitCmd = &cobra.Command{
	Use:   "reinit [namespace]",
	Short: "Restart containers after system/podman restart",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		namespace := ""
		if len(args) > 0 {
			namespace = args[0]
		}
		fmt.Printf("reinit called with namespace=%s\n", namespace)
	},
}

func init() {
	rootCmd.AddCommand(reinitCmd)
}
