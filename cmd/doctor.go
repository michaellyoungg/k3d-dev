package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"plat/pkg/tools"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check system prerequisites and tool availability",
	Long: `Diagnose environment health and validate that all required tools are available.
	
This command checks:
- k3d installation and version
- Helm installation and version  
- Docker daemon status
- System resources`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		fmt.Println("🔍 Diagnosing system health...")
		fmt.Println()

		// Check k3d
		fmt.Print("Checking k3d... ")
		if err := tools.ValidateK3d(ctx); err != nil {
			fmt.Printf("❌ %v\n", err)
		} else {
			fmt.Println("✅")
		}

		// Check helm
		fmt.Print("Checking helm... ")
		if err := tools.ValidateCommand("helm"); err != nil {
			fmt.Printf("❌ %v\n", err)
		} else {
			if version, err := tools.GetCommandVersion(ctx, "helm", "version", "--short"); err == nil {
				fmt.Printf("✅ %s\n", version)
			} else {
				fmt.Println("✅ Available")
			}
		}

		// Terraform removed from toolchain - k3d + Helm only

		// Check docker
		fmt.Print("Checking docker... ")
		if err := tools.ValidateCommand("docker"); err != nil {
			fmt.Printf("❌ %v\n", err)
		} else {
			// Test docker daemon connectivity
			executor := tools.NewProcessExecutor()
			cmd := tools.Command{Name: "docker", Args: []string{"info", "--format", "{{.ServerVersion}}"}}
			if result, err := executor.Execute(ctx, cmd); err != nil {
				fmt.Printf("❌ Docker daemon not running\n")
			} else {
				fmt.Printf("✅ Docker daemon running (v%s)\n", result.Stdout)
			}
		}

		fmt.Println()
		fmt.Println("💡 Install missing tools:")
		fmt.Println("  k3d: https://k3d.io/stable/#installation")
		fmt.Println("  helm: https://helm.sh/docs/intro/install/")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
