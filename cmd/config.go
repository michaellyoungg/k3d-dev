package cmd

import (
	"fmt"

	"plat/pkg/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage plat configuration",
	Long: `Manage plat environment configuration and settings.
	
Configuration commands help you:
• View current environment setup
• Validate configuration files
• Set execution mode preferences
• Manage local source declarations`,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display current configuration",
	Long: `Display the current environment configuration including:
- Resolved service sources and versions
- Local vs artifact execution mode
- Applied MSC defaults
- Service dependencies and ports`,
	RunE: func(cmd *cobra.Command, args []string) error {
		runtime, err := loadConfiguration()
		if err != nil {
			return err
		}

		fmt.Printf("📋 Environment Configuration\n")
		fmt.Printf("==========================\n\n")
		
		fmt.Printf("Name: %s\n", runtime.Base.Name)
		fmt.Printf("Mode: %s\n", runtime.Mode)
		fmt.Printf("Registry: %s\n", runtime.Base.Defaults.Registry)
		fmt.Printf("Domain: %s\n", runtime.Base.Defaults.Domain)
		fmt.Printf("Namespace: %s\n", runtime.Base.Defaults.Namespace)
		fmt.Printf("Services: %d\n", len(runtime.ResolvedServices))
		
		fmt.Printf("\n🔧 Service Configuration\n")
		fmt.Printf("========================\n")
		
		for name, service := range runtime.ResolvedServices {
			fmt.Printf("\n%s:\n", name)
			if service.IsLocal {
				fmt.Printf("  Source: Local (%s)\n", service.LocalSource.GetPath())
				fmt.Printf("  Build: %s\n", service.LocalSource.GetDockerfile())
			} else {
				fmt.Printf("  Source: Registry\n")
				fmt.Printf("  Version: %s\n", service.Version)
			}
			
			if service.Chart.Name != "" {
				fmt.Printf("  Chart: %s", service.Chart.Name)
				if service.Chart.Repository != "" {
					fmt.Printf(" (%s)", service.Chart.Repository)
				}
				fmt.Printf("\n")
			}
			
			if len(service.Ports) > 0 {
				fmt.Printf("  Ports: %v\n", service.Ports)
			}
			
			if len(service.Environment) > 0 {
				fmt.Printf("  Environment: %d variables\n", len(service.Environment))
			}
			
			if len(service.Dependencies) > 0 {
				fmt.Printf("  Dependencies: %v\n", service.Dependencies)
			}
		}

		return nil
	},
}

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration files",
	Long: `Validate the configuration files for syntax and logical errors.
	
Performs comprehensive validation including:
• YAML syntax validation
• Required fields verification  
• Service dependency cycles
• Local source path existence
• Helm values validation`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("🔍 Validating configuration...")
		
		runtime, err := loadConfiguration()
		if err != nil {
			fmt.Printf("❌ Configuration validation failed:\n%v\n", err)
			return err
		}
		
		// Use values manager for additional validation
		valuesManager := config.NewValuesManager(".plat")
		report := valuesManager.GetValidationReport(runtime)
		
		if len(report) == 0 {
			fmt.Println("✅ Configuration is valid!")
			
			fmt.Printf("\nSummary:\n")
			fmt.Printf("  Services: %d\n", len(runtime.ResolvedServices))
			
			localCount := 0
			artifactCount := 0
			
			for _, service := range runtime.ResolvedServices {
				if service.IsLocal {
					localCount++
				} else {
					artifactCount++
				}
			}
			
			fmt.Printf("  Local: %d, Artifact: %d\n", localCount, artifactCount)
			fmt.Printf("  Mode: %s\n", runtime.Mode)
		} else {
			fmt.Printf("⚠️  Found validation issues:\n")
			for serviceName, issues := range report {
				fmt.Printf("\n%s:\n", serviceName)
				for _, issue := range issues {
					fmt.Printf("  • %s\n", issue)
				}
			}
		}
		
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set configuration values",
	Long: `Set configuration values for persistent CLI settings.

Available settings:
  mode     - Default execution mode (local|artifact)
  domain   - Default domain for ingress (overrides config)
  strict   - Enable strict validation (true|false)`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := args[1]
		
		// TODO: Implement persistent config storage
		fmt.Printf("Setting %s = %s\n", key, value)
		fmt.Println("(Persistent configuration storage not yet implemented)")
		
		return nil
	},
}

var configExampleCmd = &cobra.Command{
	Use:   "example",
	Short: "Generate example configuration",
	Long: `Generate an example configuration showing common MSC patterns.
	
The example includes:
• Simple service definitions
• Complex services with ports and environment
• PostgreSQL database setup
• Local source declarations`,
	RunE: func(cmd *cobra.Command, args []string) error {
		example := createExampleConfig()
		
		data, err := yaml.Marshal(example)
		if err != nil {
			return fmt.Errorf("failed to generate example: %w", err)
		}
		
		fmt.Printf("# Example Plat Configuration\n")
		fmt.Printf("# Save this as .plat/config.yml in your project\n\n")
		fmt.Print(string(data))
		
		fmt.Printf("\n\n# Example Local Sources\n")
		fmt.Printf("# Save this as .plat/local.yml (gitignored)\n\n")
		
		localExample := map[string]interface{}{
			"local_sources": map[string]interface{}{
				"frontend":     "../frontend-app",
				"user-api":     "../user-service",
				"payment-api": map[string]interface{}{
					"path":       "~/dev/payments-monorepo",
					"dockerfile": "services/api/Dockerfile",
					"context":    "services/api",
					"chart":      "charts/payment-api",
				},
			},
		}
		
		localData, _ := yaml.Marshal(localExample)
		fmt.Print(string(localData))
		
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configValidateCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configExampleCmd)
}

// createExampleConfig generates an example configuration
func createExampleConfig() interface{} {
	return map[string]interface{}{
		"apiVersion": "plat/v1",
		"kind":       "Environment",
		"name":       "my-project",
		"services": []interface{}{
			"frontend",
			"user-api",
			map[string]interface{}{
				"name":    "payment-api",
				"version": "v2.1.0",
				"ports":   []int{3000, 9229},
				"environment": map[string]string{
					"NODE_ENV": "development",
					"DEBUG":    "payment:*",
				},
			},
			map[string]interface{}{
				"name": "postgres",
				"chart": map[string]interface{}{
					"name":       "postgresql",
					"repository": "https://charts.bitnami.com/bitnami",
					"version":    "12.1.9",
				},
			},
		},
		"defaults": map[string]interface{}{
			"registry":  "msc-registry.minitab.com",
			"domain":    "platform.local",
			"namespace": "default",
			"chart":     "microservice",
		},
	}
}