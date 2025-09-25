package orchestrator

import (
	"context"
	"fmt"
	"sort"

	"plat/pkg/config"
	"plat/pkg/tools"
)

// ServiceOrchestrator manages service deployment and lifecycle
type ServiceOrchestrator struct {
	helmProvider   tools.HelmProvider
	valuesManager  *config.ValuesManager
	verbose        bool
}

// NewServiceOrchestrator creates a new service orchestrator
func NewServiceOrchestrator(verbose bool) *ServiceOrchestrator {
	return &ServiceOrchestrator{
		helmProvider:  tools.NewHelmProvider(),
		valuesManager: config.NewValuesManager(".plat"),
		verbose:       verbose,
	}
}

// DeployServices deploys all services in the environment with dependency ordering
func (so *ServiceOrchestrator) DeployServices(ctx context.Context, runtime *config.RuntimeConfig) error {
	// Order services by dependencies
	orderedServices, err := so.orderServicesByDependencies(runtime)
	if err != nil {
		return fmt.Errorf("failed to resolve service dependencies: %w", err)
	}

	if so.verbose {
		fmt.Printf("🚀 Deploying %d services in dependency order\n", len(orderedServices))
		for i, serviceName := range orderedServices {
			fmt.Printf("  %d. %s\n", i+1, serviceName)
		}
	}

	// Deploy services in order
	for _, serviceName := range orderedServices {
		service := runtime.ResolvedServices[serviceName]
		
		if so.verbose {
			fmt.Printf("📦 Deploying %s...\n", serviceName)
		}

		if err := so.deployService(ctx, service, runtime); err != nil {
			return fmt.Errorf("failed to deploy service %s: %w", serviceName, err)
		}

		if so.verbose {
			fmt.Printf("✅ %s deployed successfully\n", serviceName)
		}
	}

	return nil
}

// UndeployServices removes all services from the environment
func (so *ServiceOrchestrator) UndeployServices(ctx context.Context, runtime *config.RuntimeConfig) error {
	namespace := runtime.Base.Defaults.Namespace
	
	if so.verbose {
		fmt.Printf("🗑️  Undeploying services from namespace: %s\n", namespace)
	}

	// Get all releases in the namespace
	releases, err := so.helmProvider.ListReleases(ctx, namespace)
	if err != nil {
		return fmt.Errorf("failed to list helm releases: %w", err)
	}

	// Filter to only plat-managed releases
	platReleases := so.filterPlatReleases(releases, runtime)

	// Undeploy in reverse dependency order
	orderedServices, err := so.orderServicesByDependencies(runtime)
	if err != nil {
		return fmt.Errorf("failed to resolve service dependencies: %w", err)
	}

	// Reverse the order for undeployment
	for i := len(orderedServices) - 1; i >= 0; i-- {
		serviceName := orderedServices[i]
		
		// Check if this service has a release
		var releaseExists bool
		for _, release := range platReleases {
			if release.Name == serviceName || release.Name == so.getReleaseName(serviceName, runtime) {
				releaseExists = true
				break
			}
		}

		if releaseExists {
			if so.verbose {
				fmt.Printf("🗑️  Undeploying %s...\n", serviceName)
			}

			releaseName := so.getReleaseName(serviceName, runtime)
			if err := so.helmProvider.UninstallChart(ctx, releaseName, namespace); err != nil {
				fmt.Printf("⚠️  Failed to undeploy %s: %v\n", serviceName, err)
				// Continue with other services
			} else if so.verbose {
				fmt.Printf("✅ %s undeployed\n", serviceName)
			}
		}
	}

	return nil
}

// GetServiceStatuses returns the status of all services in the environment
func (so *ServiceOrchestrator) GetServiceStatuses(ctx context.Context, runtime *config.RuntimeConfig) (map[string]*tools.ReleaseStatus, error) {
	statuses := make(map[string]*tools.ReleaseStatus)
	namespace := runtime.Base.Defaults.Namespace

	for serviceName := range runtime.ResolvedServices {
		releaseName := so.getReleaseName(serviceName, runtime)
		
		status, err := so.helmProvider.GetReleaseStatus(ctx, releaseName, namespace)
		if err != nil {
			// Service not deployed - create a placeholder status
			status = &tools.ReleaseStatus{
				Name:      releaseName,
				Namespace: namespace,
				Status:    "not-deployed",
			}
		}
		
		statuses[serviceName] = status
	}

	return statuses, nil
}

// deployService deploys a single service
func (so *ServiceOrchestrator) deployService(ctx context.Context, service *config.ResolvedService, runtime *config.RuntimeConfig) error {
	// Resolve Helm values for the service
	values, err := so.valuesManager.ResolveValues(service, runtime)
	if err != nil {
		return fmt.Errorf("failed to resolve values: %w", err)
	}

	// Validate values
	if err := so.valuesManager.ValidateValues(service, values); err != nil {
		if so.verbose {
			fmt.Printf("⚠️  Values validation warning for %s: %v\n", service.Name, err)
		}
	}

	// Create Helm release configuration
	release := tools.HelmRelease{
		Name:      so.getReleaseName(service.Name, runtime),
		Chart:     service.Chart.Name,
		Version:   service.Chart.Version,
		Repository: service.Chart.Repository,
		Namespace: runtime.Base.Defaults.Namespace,
		Values:    values,
	}

	// Add values file if specified
	if service.ValuesFile != "" {
		release.ValuesFiles = []string{service.ValuesFile}
	}

	// Install/upgrade the chart
	if err := so.helmProvider.InstallChart(ctx, release); err != nil {
		return fmt.Errorf("helm deployment failed: %w", err)
	}

	return nil
}

// orderServicesByDependencies returns services ordered by their dependencies
func (so *ServiceOrchestrator) orderServicesByDependencies(runtime *config.RuntimeConfig) ([]string, error) {
	// Build dependency graph
	graph := make(map[string][]string)
	inDegree := make(map[string]int)
	
	// Initialize graph
	for serviceName, service := range runtime.ResolvedServices {
		graph[serviceName] = service.Dependencies
		inDegree[serviceName] = 0
	}
	
	// Calculate in-degrees
	for _, dependencies := range graph {
		for _, dep := range dependencies {
			if _, exists := inDegree[dep]; exists {
				inDegree[dep]++
			}
		}
	}
	
	// Topological sort using Kahn's algorithm
	var result []string
	var queue []string
	
	// Find nodes with no incoming edges
	for service, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, service)
		}
	}
	
	// Sort queue for deterministic ordering
	sort.Strings(queue)
	
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)
		
		// Process dependencies
		var nextQueue []string
		for _, dependency := range graph[current] {
			if _, exists := inDegree[dependency]; exists {
				inDegree[dependency]--
				if inDegree[dependency] == 0 {
					nextQueue = append(nextQueue, dependency)
				}
			}
		}
		
		// Sort for deterministic ordering
		sort.Strings(nextQueue)
		queue = append(queue, nextQueue...)
	}
	
	// Check for cycles
	if len(result) != len(runtime.ResolvedServices) {
		return nil, fmt.Errorf("circular dependency detected in services")
	}
	
	return result, nil
}

// getReleaseName generates a consistent Helm release name for a service
func (so *ServiceOrchestrator) getReleaseName(serviceName string, _ *config.RuntimeConfig) string {
	// Use simple service name for release - Helm will handle namespace isolation
	return serviceName
}

// filterPlatReleases filters releases to only those managed by this plat environment
func (so *ServiceOrchestrator) filterPlatReleases(releases []tools.ReleaseInfo, runtime *config.RuntimeConfig) []tools.ReleaseInfo {
	var platReleases []tools.ReleaseInfo
	
	// Create a set of expected service names
	expectedServices := make(map[string]bool)
	for serviceName := range runtime.ResolvedServices {
		expectedServices[serviceName] = true
		expectedServices[so.getReleaseName(serviceName, runtime)] = true
	}
	
	for _, release := range releases {
		if expectedServices[release.Name] {
			platReleases = append(platReleases, release)
		}
	}
	
	return platReleases
}

// ValidatePrerequisites checks that Helm is available
func (so *ServiceOrchestrator) ValidatePrerequisites(ctx context.Context) error {
	if err := tools.ValidateHelm(ctx); err != nil {
		return fmt.Errorf("helm validation failed: %w", err)
	}
	return nil
}