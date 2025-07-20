package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// reinitCmd represents the reinit command
var reinitCmd = &cobra.Command{
	Use:   "reinit <chart>",
	Short: "Restart containers after system/podman restart",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("reinit called")
	},
}

func init() {
	rootCmd.AddCommand(reinitCmd)
}
