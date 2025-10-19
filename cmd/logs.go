package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs <service>",
	Short: "View logs for a service",
	Long: `View logs from a deployed service in the MSC development environment.

This command uses kubectl logs under the hood to stream logs from the service pods.

Examples:
  plat logs postgres           # View postgres logs
  plat logs postgres -f        # Follow/tail postgres logs
  plat logs postgres --tail 50 # Show last 50 lines
  plat logs postgres --since 5m # Show logs from last 5 minutes`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serviceName := args[0]

		// Load configuration to validate service exists
		runtime, err := loadConfiguration()
		if err != nil {
			return err
		}

		// Check if service exists
		if _, exists := runtime.ResolvedServices[serviceName]; !exists {
			return fmt.Errorf("service '%s' not found in configuration", serviceName)
		}

		// Get flags
		follow, _ := cmd.Flags().GetBool("follow")
		tailLines, _ := cmd.Flags().GetInt("tail")
		since, _ := cmd.Flags().GetString("since")
		previous, _ := cmd.Flags().GetBool("previous")
		container, _ := cmd.Flags().GetString("container")

		namespace := runtime.Base.Defaults.Namespace

		// Build kubectl logs command
		kubectlArgs := []string{"logs"}

		// Find pod for the service
		// Most Helm charts create pods with the release name as prefix
		podSelector := fmt.Sprintf("-l app.kubernetes.io/instance=%s", serviceName)
		kubectlArgs = append(kubectlArgs, podSelector)

		// Add namespace
		kubectlArgs = append(kubectlArgs, "-n", namespace)

		// Add optional flags
		if follow {
			kubectlArgs = append(kubectlArgs, "-f")
		}

		if tailLines > 0 {
			kubectlArgs = append(kubectlArgs, "--tail", fmt.Sprintf("%d", tailLines))
		}

		if since != "" {
			kubectlArgs = append(kubectlArgs, "--since", since)
		}

		if previous {
			kubectlArgs = append(kubectlArgs, "--previous")
		}

		if container != "" {
			kubectlArgs = append(kubectlArgs, "-c", container)
		}

		if verbose {
			fmt.Printf("Running: kubectl %v\n", kubectlArgs)
		}

		// Execute kubectl logs with streaming output
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		kubectlCmd := exec.CommandContext(ctx, "kubectl", kubectlArgs...)
		kubectlCmd.Stdout = os.Stdout
		kubectlCmd.Stderr = os.Stderr
		kubectlCmd.Stdin = os.Stdin

		if err := kubectlCmd.Run(); err != nil {
			// Check if no pods were found
			if exitErr, ok := err.(*exec.ExitError); ok {
				if exitErr.ExitCode() == 1 {
					return fmt.Errorf("no pods found for service '%s'. Is the service deployed? Run 'plat status' to check", serviceName)
				}
			}
			return fmt.Errorf("failed to get logs: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(logsCmd)

	logsCmd.Flags().BoolP("follow", "f", false, "Follow/stream logs")
	logsCmd.Flags().Int("tail", 100, "Number of lines to show from the end of the logs")
	logsCmd.Flags().String("since", "", "Show logs since duration (e.g., 5m, 1h)")
	logsCmd.Flags().BoolP("previous", "p", false, "Show logs from previous container instance")
	logsCmd.Flags().String("container", "", "Container name (for multi-container pods)")
}
