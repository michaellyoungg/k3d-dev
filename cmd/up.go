package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"plat/pkg/config"
	"plat/pkg/orchestrator"
)

var upCmd = &cobra.Command{
	Use:   "up [service...]",
	Short: "Start the MSC development environment",
	Long: `Start the MSC development environment with k3d cluster and Helm services.

This command will:
• Create/start the k3d cluster with MSC defaults
• Deploy services using Helm with resolved values
• Handle service dependencies automatically
• Set up ingress for local access

Examples:
  plat up                     # Start all services
  plat up frontend user-api   # Start specific services only
  plat up --mode local        # Force local development mode`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		// Load configuration
		runtime, err := loadConfiguration()
		if err != nil {
			return err
		}

		// Filter to specific services if requested
		if len(args) > 0 {
			if err := filterRuntimeServices(runtime, args); err != nil {
				return fmt.Errorf("service filtering failed: %w", err)
			}

			if verbose {
				fmt.Printf("Deploying specific services: %s\n", strings.Join(args, ", "))
			}
		}

		// Create orchestrator and validate prerequisites
		orch := orchestrator.NewOrchestrator(verbose)

		printInfo("Validating prerequisites...")
		if err := orch.ValidatePrerequisites(ctx); err != nil {
			return fmt.Errorf("prerequisite validation failed: %w", err)
		}

		// Start the environment
		if err := orch.Up(ctx, runtime); err != nil {
			return fmt.Errorf("environment startup failed: %w", err)
		}

		return nil
	},
}

// filterRuntimeServices filters the runtime configuration to only include specified services
func filterRuntimeServices(runtime *config.RuntimeConfig, serviceNames []string) error {
	// Create a set of requested services
	requested := make(map[string]bool)
	for _, name := range serviceNames {
		requested[name] = true
	}

	// Check all requested services exist
	for name := range requested {
		if _, exists := runtime.ResolvedServices[name]; !exists {
			return fmt.Errorf("service '%s' not found in configuration", name)
		}
	}

	// Filter resolved services
	filteredServices := make(map[string]*config.ResolvedService)
	for name, service := range runtime.ResolvedServices {
		if requested[name] {
			filteredServices[name] = service
		}
	}

	runtime.ResolvedServices = filteredServices
	return nil
}

func init() {
	rootCmd.AddCommand(upCmd)

	upCmd.Flags().StringP("services", "s", "", "Comma-separated list of services to start (deprecated: use args)")
}