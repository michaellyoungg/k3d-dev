package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"plat/pkg/ui"
)

var (
	verbose    bool
	configPath string
	mode       string
	strict     bool
)

var rootCmd = &cobra.Command{
	Use:   "plat",
	Short: "Local development platform for MSC microservices",
	Long: `plat is a command-line tool that orchestrates k3d and Helm
to provide seamless local development environments for MSC microservice applications.

"Docker Compose for Kubernetes" - Get your microservices running locally in 5 minutes.

Features:
• Convention over configuration with smart MSC defaults
• Switch between local development and published artifacts
• Hot-reload workflows for rapid development
• Integrated ingress and service discovery
• Helm-native deployment with values management`,
	Version: "0.1.0",
	RunE: func(cmd *cobra.Command, args []string) error {
		// If no subcommand provided, launch TUI dashboard
		if len(args) == 0 {
			// Load configuration for dashboard
			runtime, err := loadConfiguration()
			if err != nil {
				return err
			}
			return ui.RunDashboard(runtime)
		}
		return cmd.Help()
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "Config file (default is .plat/config.yml)")
	rootCmd.PersistentFlags().StringVarP(&mode, "mode", "m", "", "Execution mode: 'local' or 'artifact' (overrides config)")
	rootCmd.PersistentFlags().BoolVar(&strict, "strict", false, "Enable strict validation (fail on warnings)")
	
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if verbose {
			fmt.Printf("plat v%s\n", rootCmd.Version)
			if configPath != "" {
				fmt.Printf("Using config: %s\n", configPath)
			}
			if mode != "" {
				fmt.Printf("Mode override: %s\n", mode)
			}
		}
	}
}