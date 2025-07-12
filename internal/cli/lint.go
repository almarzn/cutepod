package cli

import (
	"github.com/spf13/cobra"
)

// lintCmd represents the lint command
var lintCmd = &cobra.Command{
	Use:   "lint <path-to-chart>",
	Short: "Validate templates and YAML structure",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: implement lint functionality
	},
}
