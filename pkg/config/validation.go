package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Value   string
	Message string
}

func (e ValidationError) Error() string {
	if e.Value != "" {
		return fmt.Sprintf("%s (%s): %s", e.Field, e.Value, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationErrors represents multiple validation errors
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	if len(e) == 1 {
		return e[0].Error()
	}

	var sb strings.Builder
	sb.WriteString("multiple validation errors:\n")
	for i, err := range e {
		sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, err.Error()))
	}
	return sb.String()
}

// ConfigValidator handles configuration validation
type ConfigValidator struct {
	configDir string
	strict    bool // Enable strict validation (fail on warnings)
}

// NewConfigValidator creates a new configuration validator
func NewConfigValidator(configDir string, strict bool) *ConfigValidator {
	return &ConfigValidator{
		configDir: configDir,
		strict:    strict,
	}
}

// ValidateBaseConfig validates the base configuration
func (cv *ConfigValidator) ValidateBaseConfig(config *BaseConfig) error {
	var errors ValidationErrors

	// Validate API version
	if config.APIVersion == "" {
		errors = append(errors, ValidationError{
			Field:   "apiVersion",
			Message: "apiVersion is required",
		})
	} else if config.APIVersion != "plat/v1" {
		errors = append(errors, ValidationError{
			Field:   "apiVersion",
			Value:   config.APIVersion,
			Message: "unsupported apiVersion, expected 'plat/v1'",
		})
	}

	// Validate kind
	if config.Kind == "" {
		errors = append(errors, ValidationError{
			Field:   "kind",
			Message: "kind is required",
		})
	} else if config.Kind != "Environment" {
		errors = append(errors, ValidationError{
			Field:   "kind",
			Value:   config.Kind,
			Message: "unsupported kind, expected 'Environment'",
		})
	}

	// Validate name
	if config.Name == "" {
		errors = append(errors, ValidationError{
			Field:   "name",
			Message: "name is required",
		})
	} else if !cv.isValidKubernetesSafeName(config.Name) {
		errors = append(errors, ValidationError{
			Field:   "name",
			Value:   config.Name,
			Message: "name must be a valid Kubernetes resource name (lowercase alphanumeric and hyphens)",
		})
	}

	// Validate services
	if len(config.Services) == 0 {
		errors = append(errors, ValidationError{
			Field:   "services",
			Message: "at least one service is required",
		})
	} else {
		serviceNames := make(map[string]bool)
		for i, service := range config.Services {
			serviceName := service.GetName()

			// Check for duplicate service names
			if serviceNames[serviceName] {
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("services[%d]", i),
					Value:   serviceName,
					Message: "duplicate service name",
				})
			}
			serviceNames[serviceName] = true

			// Validate individual service
			if serviceErrors := cv.validateService(&service, i); len(serviceErrors) > 0 {
				errors = append(errors, serviceErrors...)
			}
		}
	}

	// Validate defaults
	if config.Defaults != nil {
		if defaultsErrors := cv.validateDefaults(config.Defaults); len(defaultsErrors) > 0 {
			errors = append(errors, defaultsErrors...)
		}
	}

	if len(errors) > 0 {
		return errors
	}
	return nil
}

// ValidateLocalConfig validates the local configuration
func (cv *ConfigValidator) ValidateLocalConfig(config *LocalConfig) error {
	var errors ValidationErrors

	for name, source := range config.LocalSources {
		if !cv.isValidServiceName(name) {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("local_sources[%s]", name),
				Value:   name,
				Message: "invalid service name",
			})
		}

		if sourceErrors := cv.validateLocalSource(&source, name); len(sourceErrors) > 0 {
			errors = append(errors, sourceErrors...)
		}
	}

	if len(errors) > 0 {
		return errors
	}
	return nil
}

// ValidateRuntimeConfig validates the complete runtime configuration
func (cv *ConfigValidator) ValidateRuntimeConfig(runtime *RuntimeConfig) error {
	var errors ValidationErrors

	// Validate that all services have been resolved
	for _, service := range runtime.Base.Services {
		serviceName := service.GetName()
		if _, exists := runtime.ResolvedServices[serviceName]; !exists {
			errors = append(errors, ValidationError{
				Field:   "resolved_services",
				Value:   serviceName,
				Message: "service failed to resolve",
			})
		}
	}

	// Validate resolved services
	for name, service := range runtime.ResolvedServices {
		if serviceErrors := cv.validateResolvedService(service, name, runtime); len(serviceErrors) > 0 {
			errors = append(errors, serviceErrors...)
		}
	}

	// Validate dependency cycles
	if cycleError := cv.checkDependencyCycles(runtime); cycleError != nil {
		errors = append(errors, *cycleError)
	}

	if len(errors) > 0 {
		return errors
	}
	return nil
}

// validateService validates an individual service configuration
func (cv *ConfigValidator) validateService(service *Service, index int) ValidationErrors {
	var errors ValidationErrors
	prefix := fmt.Sprintf("services[%d]", index)

	serviceName := service.GetName()
	if serviceName == "" {
		errors = append(errors, ValidationError{
			Field:   prefix + ".name",
			Message: "service name cannot be empty",
		})
	} else if !cv.isValidServiceName(serviceName) {
		errors = append(errors, ValidationError{
			Field:   prefix + ".name",
			Value:   serviceName,
			Message: "invalid service name format",
		})
	}

	// Validate version format if specified
	if !service.IsSimpleForm() && service.Version != "" {
		if !cv.isValidVersionTag(service.Version) {
			errors = append(errors, ValidationError{
				Field:   prefix + ".version",
				Value:   service.Version,
				Message: "invalid version format",
			})
		}
	}

	// Validate ports
	for i, port := range service.Ports {
		if port < 1 || port > 65535 {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("%s.ports[%d]", prefix, i),
				Value:   fmt.Sprintf("%d", port),
				Message: "port must be between 1 and 65535",
			})
		}
	}

	// Validate environment variables
	for key, value := range service.Environment {
		if !cv.isValidEnvVarName(key) {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("%s.environment[%s]", prefix, key),
				Value:   key,
				Message: "invalid environment variable name",
			})
		}
		// Check for potentially sensitive values
		if cv.isPotentiallySensitive(key, value) {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("%s.environment[%s]", prefix, key),
				Value:   key,
				Message: "potentially sensitive value detected - consider using secrets",
			})
		}
	}

	// Validate chart configuration
	if service.Chart.Name != "" {
		if !cv.isValidChartName(service.Chart.Name) {
			errors = append(errors, ValidationError{
				Field:   prefix + ".chart.name",
				Value:   service.Chart.Name,
				Message: "invalid chart name format",
			})
		}
	}

	// Validate values file path
	if service.ValuesFile != "" {
		valuesPath := service.ValuesFile
		if !filepath.IsAbs(valuesPath) {
			valuesPath = filepath.Join(cv.configDir, valuesPath)
		}
		if _, err := os.Stat(valuesPath); os.IsNotExist(err) {
			errors = append(errors, ValidationError{
				Field:   prefix + ".values_file",
				Value:   service.ValuesFile,
				Message: "values file does not exist",
			})
		}
	}

	return errors
}

// validateLocalSource validates a local source configuration
func (cv *ConfigValidator) validateLocalSource(source *LocalSource, name string) ValidationErrors {
	var errors ValidationErrors
	prefix := fmt.Sprintf("local_sources[%s]", name)

	sourcePath := source.GetPath()
	if sourcePath == "" {
		errors = append(errors, ValidationError{
			Field:   prefix + ".path",
			Message: "path is required",
		})
		return errors
	}

	// Resolve and validate path
	absPath, err := filepath.Abs(sourcePath)
	if err != nil {
		errors = append(errors, ValidationError{
			Field:   prefix + ".path",
			Value:   sourcePath,
			Message: fmt.Sprintf("invalid path: %v", err),
		})
		return errors
	}

	// Check if path exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		errors = append(errors, ValidationError{
			Field:   prefix + ".path",
			Value:   sourcePath,
			Message: "path does not exist",
		})
	}

	// Validate dockerfile exists
	dockerfilePath := filepath.Join(absPath, source.GetDockerfile())
	if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
		if cv.strict {
			errors = append(errors, ValidationError{
				Field:   prefix + ".dockerfile",
				Value:   source.GetDockerfile(),
				Message: "dockerfile does not exist",
			})
		} else {
			// Just a warning in non-strict mode
			fmt.Printf("Warning: Dockerfile not found at %s\n", dockerfilePath)
		}
	}

	// Validate chart directory exists if using local charts
	chartPath := filepath.Join(absPath, source.GetChart())
	if _, err := os.Stat(chartPath); os.IsNotExist(err) {
		if cv.strict {
			errors = append(errors, ValidationError{
				Field:   prefix + ".chart",
				Value:   source.GetChart(),
				Message: "chart directory does not exist",
			})
		} else {
			fmt.Printf("Warning: Chart directory not found at %s\n", chartPath)
		}
	}

	return errors
}

// validateDefaults validates the defaults configuration
func (cv *ConfigValidator) validateDefaults(defaults *DefaultsConfig) ValidationErrors {
	var errors ValidationErrors

	// Validate registry URL format
	if defaults.Registry != "" {
		if !cv.isValidRegistryURL(defaults.Registry) {
			errors = append(errors, ValidationError{
				Field:   "defaults.registry",
				Value:   defaults.Registry,
				Message: "invalid registry URL format",
			})
		}
	}

	// Validate domain format
	if defaults.Domain != "" {
		if !cv.isValidDomain(defaults.Domain) {
			errors = append(errors, ValidationError{
				Field:   "defaults.domain",
				Value:   defaults.Domain,
				Message: "invalid domain format",
			})
		}
	}

	// Validate namespace
	if defaults.Namespace != "" {
		if !cv.isValidKubernetesSafeName(defaults.Namespace) {
			errors = append(errors, ValidationError{
				Field:   "defaults.namespace",
				Value:   defaults.Namespace,
				Message: "invalid namespace format",
			})
		}
	}

	return errors
}

// validateResolvedService validates a resolved service
func (cv *ConfigValidator) validateResolvedService(service *ResolvedService, name string, runtime *RuntimeConfig) ValidationErrors {
	var errors ValidationErrors
	prefix := fmt.Sprintf("resolved_services[%s]", name)

	// Validate local source consistency
	if service.IsLocal && service.LocalSource == nil {
		errors = append(errors, ValidationError{
			Field:   prefix,
			Message: "service marked as local but no local source provided",
		})
	}

	// Validate dependencies exist
	for _, dep := range service.Dependencies {
		if _, exists := runtime.ResolvedServices[dep]; !exists {
			errors = append(errors, ValidationError{
				Field:   prefix + ".dependencies",
				Value:   dep,
				Message: "dependency service not found",
			})
		}
	}

	return errors
}

// checkDependencyCycles detects circular dependencies
func (cv *ConfigValidator) checkDependencyCycles(runtime *RuntimeConfig) *ValidationError {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var hasCycle func(service string) bool
	hasCycle = func(service string) bool {
		if recStack[service] {
			return true // Found a cycle
		}
		if visited[service] {
			return false
		}

		visited[service] = true
		recStack[service] = true

		if resolvedService, exists := runtime.ResolvedServices[service]; exists {
			for _, dep := range resolvedService.Dependencies {
				if hasCycle(dep) {
					return true
				}
			}
		}

		recStack[service] = false
		return false
	}

	for serviceName := range runtime.ResolvedServices {
		if !visited[serviceName] {
			if hasCycle(serviceName) {
				return &ValidationError{
					Field:   "dependencies",
					Message: "circular dependency detected",
				}
			}
		}
	}

	return nil
}

// Validation helper functions
func (cv *ConfigValidator) isValidKubernetesSafeName(name string) bool {
	if len(name) == 0 || len(name) > 63 {
		return false
	}
	matched, _ := regexp.MatchString(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`, name)
	return matched
}

func (cv *ConfigValidator) isValidServiceName(name string) bool {
	return cv.isValidKubernetesSafeName(name)
}

func (cv *ConfigValidator) isValidVersionTag(version string) bool {
	if version == "" {
		return false
	}
	// Allow semantic versions, git hashes, and common patterns
	patterns := []string{
		`^v?\d+\.\d+\.\d+(-[a-zA-Z0-9]+)*$`,  // v1.2.3, 1.2.3-beta
		`^[a-f0-9]{7,40}$`,                   // git hash
		`^(latest|main|master|dev|develop)$`, // common tags
	}

	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, version); matched {
			return true
		}
	}
	return false
}

func (cv *ConfigValidator) isValidEnvVarName(name string) bool {
	matched, _ := regexp.MatchString(`^[a-zA-Z_][a-zA-Z0-9_]*$`, name)
	return matched
}

func (cv *ConfigValidator) isValidChartName(name string) bool {
	return cv.isValidKubernetesSafeName(name)
}

func (cv *ConfigValidator) isValidRegistryURL(url string) bool {
	// Basic registry URL validation
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9.-]+(\:[0-9]+)?(/[a-zA-Z0-9._-]+)*$`, url)
	return matched
}

func (cv *ConfigValidator) isValidDomain(domain string) bool {
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`, domain)
	return matched
}

func (cv *ConfigValidator) isPotentiallySensitive(key, value string) bool {
	sensitiveKeys := []string{"password", "secret", "key", "token", "credential"}
	keyLower := strings.ToLower(key)

	for _, sensitive := range sensitiveKeys {
		if strings.Contains(keyLower, sensitive) {
			return true
		}
	}
	return false
}
