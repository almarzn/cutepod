package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cutepod",
	Short: "Cutepod is an ephemeral Kubernetes-inspired tool for local container management",
	Long:  `Cutepod is an ephemeral, Kubernetes-inspired orchestration tool for local container management using Podman.`,
}

// Execute executes the root CLI command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(lintCmd)
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(upgradeCmd)
	rootCmd.AddCommand(reinitCmd)
}
