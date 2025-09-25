package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"devenv/pkg/orchestrator"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show MSC environment and service status",
	Long: `Display the current status of the MSC development environment.
	
Shows information about:
• k3d cluster status and health
• Helm service deployment status
• Service access URLs and ports
• Local vs artifact execution mode`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		detailed, _ := cmd.Flags().GetBool("detailed")

		// Load configuration
		runtime, err := loadConfiguration()
		if err != nil {
			return err
		}

		// Create orchestrator and get status
		orch := orchestrator.NewOrchestrator(verbose)
		
		status, err := orch.Status(ctx, runtime)
		if err != nil {
			return fmt.Errorf("failed to get environment status: %w", err)
		}

		// Display status
		displayEnvironmentStatus(status, detailed)

		return nil
	},
}

func displayEnvironmentStatus(status *orchestrator.EnvironmentStatus, detailed bool) {
	fmt.Printf("📊 Environment Status: %s\n", status.Name)
	fmt.Printf("=========================\n\n")

	// Cluster status
	fmt.Printf("🏗️  Cluster Status\n")
	if status.Cluster.Error != "" {
		fmt.Printf("   Status: ❌ %s (%s)\n", status.Cluster.Status, status.Cluster.Error)
	} else {
		statusIcon := getStatusIcon(status.Cluster.Status)
		fmt.Printf("   Status: %s %s", statusIcon, status.Cluster.Status)
		if status.Cluster.Name != "" {
			fmt.Printf(" (%s)", status.Cluster.Name)
		}
		fmt.Println()
		
		if status.Cluster.Servers > 0 || status.Cluster.Agents > 0 {
			fmt.Printf("   Nodes: %d servers, %d agents\n", status.Cluster.Servers, status.Cluster.Agents)
		}
	}

	// Services status
	fmt.Printf("\n📦 Services (%s mode)\n", status.Mode)
	
	if len(status.Services) == 0 {
		fmt.Println("   No services configured")
		return
	}

	for serviceName, service := range status.Services {
		statusIcon := getStatusIcon(service.Status)
		fmt.Printf("   %s %s", statusIcon, serviceName)
		
		if service.Version != "" {
			fmt.Printf(" (%s)", service.Version)
		}
		
		if service.IsLocal && service.LocalPath != "" {
			fmt.Printf(" 🔧 local")
		}
		
		if service.Status != "deployed" && service.Status != "not-deployed" {
			fmt.Printf(" [%s]", service.Status)
		}
		
		fmt.Println()
		
		if detailed {
			if service.Chart != "" {
				fmt.Printf("      Chart: %s\n", service.Chart)
			}
			if service.IsLocal && service.LocalPath != "" {
				fmt.Printf("      Path: %s\n", service.LocalPath)
			}
			if len(service.Ports) > 0 {
				fmt.Printf("      Ports: %v\n", service.Ports)
			}
			if service.Updated != "" {
				fmt.Printf("      Updated: %s\n", service.Updated)
			}
		}
	}

	// Access information
	localServices := getLocalServices(status.Services)
	if len(localServices) > 0 {
		fmt.Printf("\n🌐 Service Access\n")
		for _, serviceName := range localServices {
			service := status.Services[serviceName]
			if len(service.Ports) > 0 {
				port := service.Ports[0]
				fmt.Printf("   • %s: http://localhost:%d\n", serviceName, port)
			}
		}
	}

	// Development info
	if status.Mode == "local" {
		localDevServices := getLocalDevServices(status.Services)
		if len(localDevServices) > 0 {
			fmt.Printf("\n📝 Local Development\n")
			for _, serviceName := range localDevServices {
				service := status.Services[serviceName]
				fmt.Printf("   • %s: %s\n", serviceName, service.LocalPath)
			}
		}
	}
}

func getStatusIcon(status string) string {
	switch strings.ToLower(status) {
	case "running", "deployed":
		return "✅"
	case "starting", "pending-install", "pending-upgrade":
		return "⏳"
	case "failed", "error":
		return "❌"
	case "stopped", "not-deployed", "not-found":
		return "⏸️ "
	default:
		return "⚠️ "
	}
}

func getLocalServices(services map[string]*orchestrator.ServiceStatus) []string {
	var local []string
	for name, service := range services {
		if len(service.Ports) > 0 {
			local = append(local, name)
		}
	}
	return local
}

func getLocalDevServices(services map[string]*orchestrator.ServiceStatus) []string {
	var localDev []string
	for name, service := range services {
		if service.IsLocal && service.LocalPath != "" {
			localDev = append(localDev, name)
		}
	}
	return localDev
}

func init() {
	rootCmd.AddCommand(statusCmd)
	
	statusCmd.Flags().Bool("detailed", false, "Show detailed status information")
}