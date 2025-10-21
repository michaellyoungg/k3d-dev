package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"plat/pkg/config"
)

var initCmd = &cobra.Command{
	Use:   "init [project-name]",
	Short: "Initialize a new MSC development environment",
	Long: `Initialize a new MSC development environment with smart defaults.

Creates the .plat/ directory structure with:
• Base configuration (config.yml) with MSC defaults
• Local source declarations (local.yml) - gitignored
• Example service definitions based on template

Templates:
  microservices  - Standard MSC microservice stack (default)
  fullstack      - Frontend + backend + database
  backend-only   - API services without frontend`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName := "platform-backend"
		if len(args) > 0 {
			projectName = args[0]
		}

		template, _ := cmd.Flags().GetString("template")
		force, _ := cmd.Flags().GetBool("force")
		scanLocal, _ := cmd.Flags().GetBool("scan-local")

		return initializeEnvironment(projectName, template, force, scanLocal)
	},
}

func initializeEnvironment(projectName, template string, force, scanLocal bool) error {
	// Check if .plat directory already exists
	platDir := ".plat"
	if _, err := os.Stat(platDir); err == nil && !force {
		return fmt.Errorf(".plat directory already exists (use --force to overwrite)")
	}

	// Create .plat directory
	if err := os.MkdirAll(platDir, 0755); err != nil {
		return fmt.Errorf("failed to create .plat directory: %w", err)
	}

	printInfo("Created .plat directory")

	// Create base configuration
	baseConfig := createBaseConfig(projectName, template)
	configPath := filepath.Join(platDir, "config.yml")

	if err := writeYAMLFile(configPath, baseConfig); err != nil {
		return fmt.Errorf("failed to write config.yml: %w", err)
	}

	printSuccess("Created config.yml with MSC defaults")

	// Create local configuration (empty initially)
	localConfig := &config.LocalConfig{
		LocalSources: make(map[string]config.LocalSource),
	}

	// Scan for local repositories if requested
	if scanLocal {
		printInfo("Scanning for local repositories...")
		scannedSources := scanForLocalSources()
		if len(scannedSources) > 0 {
			localConfig.LocalSources = scannedSources
			printSuccess(fmt.Sprintf("Found %d local repositories", len(scannedSources)))
		}
	}

	localPath := filepath.Join(platDir, "local.yml")
	if err := writeYAMLFile(localPath, localConfig); err != nil {
		return fmt.Errorf("failed to write local.yml: %w", err)
	}

	printSuccess("Created local.yml for local source declarations")

	// Create .gitignore for .plat directory
	if err := createPlatGitignore(platDir); err != nil {
		printWarning(fmt.Sprintf("Failed to update .gitignore: %v", err))
	}

	// Print usage instructions
	printInitializationComplete(projectName, template)

	return nil
}

func createBaseConfig(projectName, template string) interface{} {
	// Create a YAML-friendly structure instead of using config structs
	// to avoid union type marshaling issues during init
	baseConfig := map[string]interface{}{
		"apiVersion": "plat/v1",
		"kind":       "Environment",
		"name":       projectName,
		"defaults": map[string]interface{}{
			"registry":  "msc-registry.minitab.com",
			"domain":    "platform.local",
			"namespace": "default",
			"chart":     "microservice",
		},
	}

	// Add services based on template
	var services []interface{}
	switch template {
	case "fullstack":
		services = []interface{}{
			"frontend",
			"backend-api",
			map[string]interface{}{
				"name": "postgres",
				"chart": map[string]interface{}{
					"name":       "postgresql",
					"repository": "https://charts.bitnami.com/bitnami",
					"version":    "12.1.9",
				},
			},
		}
	case "backend-only":
		services = []interface{}{
			"user-api",
			"payment-api",
			map[string]interface{}{
				"name": "postgres",
				"chart": map[string]interface{}{
					"name":       "postgresql",
					"repository": "https://charts.bitnami.com/bitnami",
					"version":    "12.1.9",
				},
			},
		}
	default: // microservices
		services = []interface{}{
			"frontend",
			"user-api",
			"payment-api",
			"order-api",
			map[string]interface{}{
				"name": "postgres",
				"chart": map[string]interface{}{
					"name":       "postgresql",
					"repository": "https://charts.bitnami.com/bitnami",
					"version":    "12.1.9",
				},
			},
		}
	}

	baseConfig["services"] = services
	return baseConfig
}

func scanForLocalSources() map[string]config.LocalSource {
	sources := make(map[string]config.LocalSource)

	// Look for common patterns in parent directory
	parentDir := ".."
	entries, err := os.ReadDir(parentDir)
	if err != nil {
		return sources
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		entryPath := filepath.Join(parentDir, entry.Name())

		// Check if it looks like a service repository
		if isServiceRepository(entryPath) {
			serviceName := entry.Name()
			sources[serviceName] = config.LocalSource{
				Path: entryPath,
			}

			if verbose {
				fmt.Printf("  Found: %s\n", entryPath)
			}
		}
	}

	return sources
}

func isServiceRepository(path string) bool {
	// Check for common service repository indicators
	indicators := []string{"Dockerfile", "package.json", "pom.xml", "go.mod", "requirements.txt"}

	for _, indicator := range indicators {
		if _, err := os.Stat(filepath.Join(path, indicator)); err == nil {
			return true
		}
	}

	return false
}

func createPlatGitignore(_ string) error {
	gitignoreContent := `# Plat local configuration
.plat/local.yml
.plat/.platconfig
`

	gitignorePath := ".gitignore"

	// Check if .gitignore exists
	var existingContent []byte
	if content, err := os.ReadFile(gitignorePath); err == nil {
		existingContent = content

		// Check if plat entries already exist
		if filepath.Base(string(existingContent)) != "" {
			existingContent = append(existingContent, '\n')
		}
	}

	// Append plat-specific entries
	newContent := append(existingContent, []byte(gitignoreContent)...)
	return os.WriteFile(gitignorePath, newContent, 0644)
}

func writeYAMLFile(path string, data interface{}) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(2)
	return encoder.Encode(data)
}

func printInitializationComplete(projectName, template string) {
	fmt.Printf("\n🎉 Environment '%s' initialized successfully!\n\n", projectName)
	fmt.Printf("Template: %s\n", template)
	fmt.Printf("Configuration: .plat/config.yml\n")
	fmt.Printf("Local sources: .plat/local.yml\n\n")

	fmt.Println("Next steps:")
	fmt.Println("1. Review and customize .plat/config.yml")
	fmt.Println("2. Declare local sources in .plat/local.yml")
	fmt.Println("3. Run 'plat up' to start your environment")
	fmt.Println("4. Use 'plat status' to check service health")
	fmt.Println("\nTip: Run 'plat init --scan-local' to auto-discover local repositories")
}

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().StringP("template", "t", "microservices", "Project template: microservices, fullstack, backend-only")
	initCmd.Flags().BoolP("force", "f", false, "Overwrite existing .plat configuration")
	initCmd.Flags().Bool("scan-local", false, "Automatically scan for local repositories")
}
