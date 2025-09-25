package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ValuesManager handles Helm values resolution and merging
type ValuesManager struct {
	configDir string
}

// NewValuesManager creates a new values manager
func NewValuesManager(configDir string) *ValuesManager {
	return &ValuesManager{
		configDir: configDir,
	}
}

// ResolveValues resolves final Helm values for a service
func (vm *ValuesManager) ResolveValues(service *ResolvedService, runtime *RuntimeConfig) (map[string]interface{}, error) {
	values := make(map[string]interface{})

	// 1. Start with MSC chart defaults
	defaults, err := vm.getChartDefaults(service.Chart.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get chart defaults: %w", err)
	}
	vm.mergeValues(values, defaults)

	// 2. Apply service-specific values from config
	if service.Values != nil {
		vm.mergeValues(values, service.Values)
	}

	// 3. Load values from external file if specified
	if service.ValuesFile != "" {
		fileValues, err := vm.loadValuesFile(service.ValuesFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load values file %s: %w", service.ValuesFile, err)
		}
		vm.mergeValues(values, fileValues)
	}

	// 4. Apply local development overrides
	localOverrides := vm.buildLocalOverrides(service, runtime)
	vm.mergeValues(values, localOverrides)

	// 5. Apply runtime-specific overrides (ingress, resources, etc.)
	runtimeOverrides := vm.buildRuntimeOverrides(service, runtime)
	vm.mergeValues(values, runtimeOverrides)

	return values, nil
}

// getChartDefaults returns default values for MSC chart types
func (vm *ValuesManager) getChartDefaults(chartName string) (map[string]interface{}, error) {
	switch chartName {
	case "microservice":
		return map[string]interface{}{
			"replicaCount": 1,
			"image": map[string]interface{}{
				"pullPolicy": "IfNotPresent",
			},
			"service": map[string]interface{}{
				"type": "ClusterIP",
				"port": 80,
			},
			"ingress": map[string]interface{}{
				"enabled": true,
				"className": "nginx",
			},
			"resources": map[string]interface{}{
				"limits": map[string]interface{}{
					"cpu":    "500m",
					"memory": "512Mi",
				},
				"requests": map[string]interface{}{
					"cpu":    "100m",
					"memory": "128Mi",
				},
			},
			"autoscaling": map[string]interface{}{
				"enabled": false,
			},
		}, nil
	case "postgresql":
		return map[string]interface{}{
			"auth": map[string]interface{}{
				"postgresPassword": "development",
				"database":         "app",
			},
			"primary": map[string]interface{}{
				"persistence": map[string]interface{}{
					"enabled": false, // No persistence in local dev
				},
			},
		}, nil
	default:
		// For unknown charts, return empty defaults
		return map[string]interface{}{}, nil
	}
}

// loadValuesFile loads values from a YAML file
func (vm *ValuesManager) loadValuesFile(valuesFile string) (map[string]interface{}, error) {
	// Support relative paths from config directory
	filePath := valuesFile
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(vm.configDir, filePath)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var values map[string]interface{}
	if err := yaml.Unmarshal(data, &values); err != nil {
		return nil, fmt.Errorf("failed to parse values YAML: %w", err)
	}

	return values, nil
}

// buildLocalOverrides creates values for local development
func (vm *ValuesManager) buildLocalOverrides(service *ResolvedService, runtime *RuntimeConfig) map[string]interface{} {
	overrides := make(map[string]interface{})

	if service.IsLocal {
		// Override image for local builds
		overrides["image"] = map[string]interface{}{
			"repository": service.Name,
			"tag":        "dev",
			"pullPolicy": "Never", // Don't pull local images
		}

		// Override full image reference if needed
		overrides["fullnameOverride"] = service.Name

		// Add development-friendly settings
		if service.LocalSource != nil {
			// Add build context info as annotation
			overrides["podAnnotations"] = map[string]interface{}{
				"dev.plat.io/local-source": service.LocalSource.GetPath(),
				"dev.plat.io/dockerfile":   service.LocalSource.GetDockerfile(),
			}
		}

		// Disable resource limits for local dev
		overrides["resources"] = map[string]interface{}{
			"limits":   map[string]interface{}{},
			"requests": map[string]interface{}{},
		}
	} else {
		// Use registry image
		overrides["image"] = map[string]interface{}{
			"repository": fmt.Sprintf("%s/%s", runtime.Base.Defaults.Registry, service.Name),
			"tag":        service.Version,
			"pullPolicy": "IfNotPresent",
		}
	}

	return overrides
}

// buildRuntimeOverrides creates runtime-specific values
func (vm *ValuesManager) buildRuntimeOverrides(service *ResolvedService, runtime *RuntimeConfig) map[string]interface{} {
	overrides := make(map[string]interface{})

	// Configure ingress with platform domain
	if runtime.Base.Defaults.Domain != "" {
		host := fmt.Sprintf("%s.%s", service.Name, runtime.Base.Defaults.Domain)
		overrides["ingress"] = map[string]interface{}{
			"enabled": true,
			"hosts": []map[string]interface{}{
				{
					"host": host,
					"paths": []map[string]interface{}{
						{
							"path":     "/",
							"pathType": "Prefix",
						},
					},
				},
			},
		}
	}

	// Apply environment variables
	if len(service.Environment) > 0 {
		env := make([]map[string]interface{}, 0, len(service.Environment))
		for key, value := range service.Environment {
			env = append(env, map[string]interface{}{
				"name":  key,
				"value": value,
			})
		}
		overrides["env"] = env
	}

	// Configure service ports
	if len(service.Ports) > 0 {
		// Use first port as primary service port
		overrides["service"] = map[string]interface{}{
			"port": service.Ports[0],
		}

		// If multiple ports, configure container ports
		if len(service.Ports) > 1 {
			containerPorts := make([]map[string]interface{}, len(service.Ports))
			for i, port := range service.Ports {
				containerPorts[i] = map[string]interface{}{
					"name":          fmt.Sprintf("port-%d", i),
					"containerPort": port,
					"protocol":      "TCP",
				}
			}
			overrides["containerPorts"] = containerPorts
		}
	}

	return overrides
}

// mergeValues merges source values into target (deep merge)
func (vm *ValuesManager) mergeValues(target, source map[string]interface{}) {
	for key, sourceValue := range source {
		if targetValue, exists := target[key]; exists {
			// Both exist, try to merge if both are maps
			if targetMap, targetIsMap := targetValue.(map[string]interface{}); targetIsMap {
				if sourceMap, sourceIsMap := sourceValue.(map[string]interface{}); sourceIsMap {
					vm.mergeValues(targetMap, sourceMap)
					continue
				}
			}
		}
		// Either target doesn't exist or can't merge, so overwrite
		target[key] = sourceValue
	}
}

// ValidateValues validates the final values for common issues
func (vm *ValuesManager) ValidateValues(service *ResolvedService, values map[string]interface{}) error {
	var errors []string

	// Check required image configuration
	if image, hasImage := values["image"]; hasImage {
		if imageMap, isMap := image.(map[string]interface{}); isMap {
			if _, hasRepo := imageMap["repository"]; !hasRepo {
				errors = append(errors, "missing image.repository")
			}
			if _, hasTag := imageMap["tag"]; !hasTag {
				errors = append(errors, "missing image.tag")
			}
		} else {
			errors = append(errors, "image configuration must be an object")
		}
	} else {
		errors = append(errors, "missing required image configuration")
	}

	// Validate service configuration
	if serviceConf, hasService := values["service"]; hasService {
		if serviceMap, isMap := serviceConf.(map[string]interface{}); isMap {
			if port, hasPort := serviceMap["port"]; hasPort {
				if portFloat, isNumber := port.(float64); isNumber {
					if portInt := int(portFloat); portInt < 1 || portInt > 65535 {
						errors = append(errors, fmt.Sprintf("invalid service port %d (must be 1-65535)", portInt))
					}
				}
			}
		}
	}

	// Validate ingress configuration
	if ingress, hasIngress := values["ingress"]; hasIngress {
		if ingressMap, isMap := ingress.(map[string]interface{}); isMap {
			if enabled, hasEnabled := ingressMap["enabled"]; hasEnabled {
				if enabledBool, isBool := enabled.(bool); isBool {
					if !enabledBool && service.IsLocal {
						fmt.Printf("Warning: Local service %s has ingress disabled - may not be accessible\n", service.Name)
					}
				}
			}
		}
	}

	// Validate resource limits
	if resources, hasResources := values["resources"]; hasResources {
		if resourcesMap, isMap := resources.(map[string]interface{}); isMap {
			if limits, hasLimits := resourcesMap["limits"]; hasLimits {
				if limitsMap, isLimitsMap := limits.(map[string]interface{}); isLimitsMap && len(limitsMap) == 0 {
					if !service.IsLocal {
						fmt.Printf("Warning: Service %s has no resource limits - consider setting limits for production\n", service.Name)
					}
				}
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation failed for service %s: %s", service.Name, strings.Join(errors, "; "))
	}

	return nil
}

// GetValidationReport generates a validation report for all resolved values
func (vm *ValuesManager) GetValidationReport(runtime *RuntimeConfig) map[string][]string {
	report := make(map[string][]string)

	for name, service := range runtime.ResolvedServices {
		var issues []string

		values, err := vm.ResolveValues(service, runtime)
		if err != nil {
			issues = append(issues, fmt.Sprintf("Failed to resolve values: %v", err))
		} else {
			if err := vm.ValidateValues(service, values); err != nil {
				issues = append(issues, err.Error())
			}
		}

		// Additional service-specific checks
		if service.IsLocal && service.LocalSource == nil {
			issues = append(issues, "Service marked as local but no local source configured")
		}

		if len(service.Dependencies) > 0 {
			for _, dep := range service.Dependencies {
				if _, exists := runtime.ResolvedServices[dep]; !exists {
					issues = append(issues, fmt.Sprintf("Dependency '%s' not found", dep))
				}
			}
		}

		if len(issues) > 0 {
			report[name] = issues
		}
	}

	return report
}