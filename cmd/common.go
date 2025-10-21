package cmd

import (
	"fmt"
	"os"

	"plat/pkg/config"
)

// loadConfiguration loads and validates the configuration with CLI overrides
func loadConfiguration() (*config.RuntimeConfig, error) {
	// Determine execution mode
	execMode := config.ModeArtifact // Default mode
	if mode != "" {
		switch mode {
		case "local":
			execMode = config.ModeLocal
		case "artifact":
			execMode = config.ModeArtifact
		default:
			return nil, fmt.Errorf("invalid mode %q, must be 'local' or 'artifact'", mode)
		}
	}

	// Create loader with validation options
	var loader *config.Loader
	if strict {
		loader = config.NewLoaderWithValidation(configPath, execMode, true)
	} else {
		loader = config.NewLoader(configPath, execMode)
	}

	// Load configuration
	runtime, err := loader.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	if verbose {
		fmt.Printf("Loaded %d services in %s mode\n", len(runtime.ResolvedServices), execMode)
		for name, service := range runtime.ResolvedServices {
			if service.IsLocal {
				fmt.Printf("  • %s (local: %s)\n", name, service.LocalSource.GetPath())
			} else {
				fmt.Printf("  • %s (%s)\n", name, service.Version)
			}
		}
	}

	return runtime, nil
}

// confirmAction prompts for confirmation if not in CI/automated mode
func confirmAction(message string) bool {
	if os.Getenv("CI") != "" || os.Getenv("PLAT_AUTO_CONFIRM") != "" {
		return true
	}

	fmt.Printf("%s [y/N]: ", message)
	var response string
	fmt.Scanln(&response)

	return response == "y" || response == "Y" || response == "yes" || response == "Yes"
}

// printSuccess prints a success message with formatting
func printSuccess(message string) {
	if verbose {
		fmt.Printf("✅ %s\n", message)
	}
}

// printWarning prints a warning message with formatting
func printWarning(message string) {
	fmt.Printf("⚠️  %s\n", message)
}

// printError prints an error message with formatting
func printError(message string) {
	fmt.Printf("❌ %s\n", message)
}

// printInfo prints an info message with formatting
func printInfo(message string) {
	if verbose {
		fmt.Printf("ℹ️  %s\n", message)
	}
}
