package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// DefaultConfigPaths are the standard locations to look for config files
var DefaultConfigPaths = []string{
	".plat/config.yml",
	".plat/config.yaml",
}

// Loader handles configuration loading and merging
type Loader struct {
	configPath string
	mode       ExecutionMode
	validator  *ConfigValidator
}

// NewLoader creates a new configuration loader
func NewLoader(configPath string, mode ExecutionMode) *Loader {
	return &Loader{
		configPath: configPath,
		mode:       mode,
		validator:  NewConfigValidator("", false), // Will be updated with actual config dir
	}
}

// NewLoaderWithValidation creates a new configuration loader with validation options
func NewLoaderWithValidation(configPath string, mode ExecutionMode, strict bool) *Loader {
	return &Loader{
		configPath: configPath,
		mode:       mode,
		validator:  NewConfigValidator("", strict),
	}
}

// Load loads and merges configuration from files
func (l *Loader) Load() (*RuntimeConfig, error) {
	// Find config file if not specified
	configFile := l.configPath
	if configFile == "" {
		found, err := l.findConfigFile()
		if err != nil {
			return nil, err
		}
		configFile = found
	}

	// Update validator with config directory
	configDir := filepath.Dir(configFile)
	l.validator.configDir = configDir

	// Load base configuration
	baseConfig, err := l.loadBaseConfig(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load config file %s: %w", configFile, err)
	}

	// Validate base configuration
	if err := l.validator.ValidateBaseConfig(baseConfig); err != nil {
		return nil, fmt.Errorf("invalid base configuration: %w", err)
	}

	// Load local configuration if exists
	localConfig, err := l.loadLocalConfig(configDir)
	if err != nil {
		// Local config is optional
		localConfig = &LocalConfig{
			LocalSources: make(map[string]LocalSource),
		}
	} else {
		// Validate local configuration
		if err := l.validator.ValidateLocalConfig(localConfig); err != nil {
			return nil, fmt.Errorf("invalid local configuration: %w", err)
		}
	}

	// Create runtime config
	runtime := &RuntimeConfig{
		Base:             baseConfig,
		Local:            localConfig,
		Mode:             l.mode,
		ResolvedServices: make(map[string]*ResolvedService),
		Timestamp:        time.Now(),
	}

	// Resolve services
	if err := l.resolveServices(runtime); err != nil {
		return nil, fmt.Errorf("failed to resolve services: %w", err)
	}

	// Validate final runtime configuration
	if err := l.validator.ValidateRuntimeConfig(runtime); err != nil {
		return nil, fmt.Errorf("invalid runtime configuration: %w", err)
	}

	return runtime, nil
}

// findConfigFile looks for config file in standard locations
func (l *Loader) findConfigFile() (string, error) {
	for _, path := range DefaultConfigPaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("no config file found in standard locations: %s",
		strings.Join(DefaultConfigPaths, ", "))
}

// loadBaseConfig loads the base configuration file
func (l *Loader) loadBaseConfig(path string) (*BaseConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config BaseConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Apply MSC defaults if not specified
	if config.Defaults == nil {
		config.Defaults = &DefaultsConfig{}
	}
	if config.Defaults.Registry == "" {
		config.Defaults.Registry = "msc-registry.minitab.com"
	}
	if config.Defaults.Domain == "" {
		config.Defaults.Domain = "platform.local"
	}
	if config.Defaults.Namespace == "" {
		config.Defaults.Namespace = "default"
	}
	if config.Defaults.Chart == "" {
		config.Defaults.Chart = "microservice"
	}

	return &config, nil
}

// loadLocalConfig loads the local configuration file
func (l *Loader) loadLocalConfig(configDir string) (*LocalConfig, error) {
	localPath := filepath.Join(configDir, "local.yml")
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		// Try .yaml extension
		localPath = filepath.Join(configDir, "local.yaml")
		if _, err := os.Stat(localPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("local config file not found")
		}
	}

	data, err := os.ReadFile(localPath)
	if err != nil {
		return nil, err
	}

	var config LocalConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse local YAML: %w", err)
	}

	// Validate local sources
	for name, source := range config.LocalSources {
		if err := source.Validate(); err != nil {
			return nil, fmt.Errorf("invalid local source %s: %w", name, err)
		}
	}

	return &config, nil
}

// resolveServices creates resolved service configurations
func (l *Loader) resolveServices(runtime *RuntimeConfig) error {
	for _, service := range runtime.Base.Services {
		serviceName := service.GetName()

		resolved := &ResolvedService{
			Name:         serviceName,
			Version:      service.GetVersion(),
			Environment:  make(map[string]string),
			Dependencies: []string{},
		}

		// Copy base service configuration
		if !service.IsSimpleForm() {
			resolved.Chart = service.Chart
			resolved.Values = service.Values
			resolved.ValuesFile = service.ValuesFile
			resolved.Ports = service.Ports
			resolved.Environment = service.Environment
			resolved.Dependencies = service.Dependencies
		} else {
			// Apply defaults for simple form
			if runtime.Base.Defaults != nil && runtime.Base.Defaults.Chart != "" {
				resolved.Chart = ServiceChart{
					Name: runtime.Base.Defaults.Chart,
				}
			}
		}

		// Check if local source is available and mode supports it
		if localSource, hasLocal := runtime.Local.LocalSources[serviceName]; hasLocal {
			if runtime.Mode == ModeLocal {
				resolved.IsLocal = true
				resolved.LocalSource = &localSource
			}
		}

		runtime.ResolvedServices[serviceName] = resolved
	}

	return nil
}

// GetService returns a resolved service by name
func (r *RuntimeConfig) GetService(name string) (*ResolvedService, bool) {
	service, exists := r.ResolvedServices[name]
	return service, exists
}

// ListServices returns all service names
func (r *RuntimeConfig) ListServices() []string {
	names := make([]string, 0, len(r.ResolvedServices))
	for name := range r.ResolvedServices {
		names = append(names, name)
	}
	return names
}

// FilterServices returns services matching the given names
func (r *RuntimeConfig) FilterServices(names []string) map[string]*ResolvedService {
	if len(names) == 0 {
		return r.ResolvedServices
	}

	filtered := make(map[string]*ResolvedService)
	for _, name := range names {
		if service, exists := r.ResolvedServices[name]; exists {
			filtered[name] = service
		}
	}
	return filtered
}
