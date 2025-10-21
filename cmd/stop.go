package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"plat/pkg/orchestrator"
)

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Stop the MSC development environment",
	Long: `Stop the MSC development environment services and optionally the cluster.
	
This command will:
• Undeploy all Helm services in dependency order
• Optionally delete the k3d cluster
• Clean up resources while preserving configuration

Examples:
  plat down              # Stop services, keep cluster
  plat down --cluster    # Stop services and delete cluster
  plat down --confirm    # Skip confirmation prompt`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		deleteCluster, _ := cmd.Flags().GetBool("cluster")
		skipConfirm, _ := cmd.Flags().GetBool("confirm")

		// Load configuration
		runtime, err := loadConfiguration()
		if err != nil {
			return err
		}

		// Confirmation prompt
		if !skipConfirm {
			message := "Stop all services"
			if deleteCluster {
				message = "Stop all services and delete cluster"
			}

			if !confirmAction(message + "?") {
				fmt.Println("Operation cancelled")
				return nil
			}
		}

		// Create orchestrator and stop environment
		orch := orchestrator.NewOrchestrator(verbose)

		if err := orch.Down(ctx, runtime, deleteCluster); err != nil {
			return fmt.Errorf("environment shutdown failed: %w", err)
		}

		return nil
	},
}

// Legacy stop command for compatibility
var stopCmd = &cobra.Command{
	Use:        "stop",
	Short:      "Stop the development environment (alias for 'down')",
	Hidden:     true, // Hide from help but keep for compatibility
	Deprecated: "use 'plat down' instead",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Forward to down command
		return downCmd.RunE(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(downCmd)
	rootCmd.AddCommand(stopCmd) // Legacy alias

	downCmd.Flags().Bool("cluster", false, "Also delete the k3d cluster")
	downCmd.Flags().Bool("confirm", false, "Skip confirmation prompt")

	// Legacy flags for stop command
	stopCmd.Flags().Bool("cluster", false, "Also delete the k3d cluster")
	stopCmd.Flags().Bool("confirm", false, "Skip confirmation prompt")
}
