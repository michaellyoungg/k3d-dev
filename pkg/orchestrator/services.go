package orchestrator

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"plat/pkg/config"
	"plat/pkg/tools"
)

// ServiceOrchestrator manages service deployment and lifecycle
type ServiceOrchestrator struct {
	helmProvider  tools.HelmProvider
	valuesManager *config.ValuesManager
	verbose       bool
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
	// Group services by dependency level for concurrent deployment
	serviceLevels, err := so.groupServicesByDependencyLevel(runtime)
	if err != nil {
		return fmt.Errorf("failed to resolve service dependencies: %w", err)
	}

	if so.verbose {
		fmt.Printf("🚀 Deploying %d services across %d level(s)\n", len(runtime.ResolvedServices), len(serviceLevels))
		for levelIdx, level := range serviceLevels {
			if len(level) == 1 {
				fmt.Printf("  Level %d: %s\n", levelIdx, level[0])
			} else {
				fmt.Printf("  Level %d: %s (concurrent)\n", levelIdx, strings.Join(level, ", "))
			}
		}
	}

	// Deploy each level, services within a level deploy concurrently
	for levelIdx, level := range serviceLevels {
		if so.verbose && len(level) > 1 {
			fmt.Printf("📦 Deploying level %d (%d services concurrently)...\n", levelIdx, len(level))
		}

		if err := so.deployServicesInLevel(ctx, level, runtime); err != nil {
			return fmt.Errorf("failed to deploy level %d: %w", levelIdx, err)
		}

		if so.verbose {
			fmt.Printf("✅ Level %d deployed successfully\n", levelIdx)
		}
	}

	return nil
}

// deployServicesInLevel deploys multiple services concurrently
func (so *ServiceOrchestrator) deployServicesInLevel(ctx context.Context, serviceNames []string, runtime *config.RuntimeConfig) error {
	// Use error group for concurrent deployment with error aggregation
	type deployResult struct {
		serviceName string
		err         error
	}

	resultChan := make(chan deployResult, len(serviceNames))
	var wg sync.WaitGroup

	// Deploy all services in this level concurrently
	for _, serviceName := range serviceNames {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()

			service := runtime.ResolvedServices[name]

			if so.verbose {
				fmt.Printf("📦 Deploying %s...\n", name)
			}

			err := so.deployService(ctx, service, runtime)

			if err != nil {
				resultChan <- deployResult{serviceName: name, err: err}
			} else {
				if so.verbose {
					fmt.Printf("✅ %s deployed successfully\n", name)
				}
				resultChan <- deployResult{serviceName: name, err: nil}
			}
		}(serviceName)
	}

	// Wait for all deployments to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results and aggregate errors
	var errors []error
	for result := range resultChan {
		if result.err != nil {
			errors = append(errors, fmt.Errorf("%s: %w", result.serviceName, result.err))
		}
	}

	// If any deployments failed, return combined error
	if len(errors) > 0 {
		var errMsg strings.Builder
		errMsg.WriteString("service deployment failures:\n")
		for _, err := range errors {
			errMsg.WriteString(fmt.Sprintf("  - %v\n", err))
		}
		return fmt.Errorf(errMsg.String())
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

	// Group services by dependency level
	serviceLevels, err := so.groupServicesByDependencyLevel(runtime)
	if err != nil {
		return fmt.Errorf("failed to resolve service dependencies: %w", err)
	}

	// Undeploy in reverse level order (reverse dependencies)
	for i := len(serviceLevels) - 1; i >= 0; i-- {
		level := serviceLevels[i]

		if so.verbose && len(level) > 1 {
			fmt.Printf("🗑️  Undeploying level %d (%d services concurrently)...\n", i, len(level))
		}

		if err := so.undeployServicesInLevel(ctx, level, platReleases, runtime, namespace); err != nil {
			// Continue with other levels even if this one has errors
			fmt.Printf("⚠️  Level %d undeployment had errors: %v\n", i, err)
		}
	}

	return nil
}

// undeployServicesInLevel undeploys multiple services concurrently
func (so *ServiceOrchestrator) undeployServicesInLevel(ctx context.Context, serviceNames []string, platReleases []tools.ReleaseInfo, runtime *config.RuntimeConfig, namespace string) error {
	var wg sync.WaitGroup
	errorsChan := make(chan error, len(serviceNames))

	// Undeploy all services in this level concurrently
	for _, serviceName := range serviceNames {
		// Check if this service has a release
		var releaseExists bool
		for _, release := range platReleases {
			if release.Name == serviceName || release.Name == so.getReleaseName(serviceName, runtime) {
				releaseExists = true
				break
			}
		}

		if !releaseExists {
			continue
		}

		wg.Add(1)
		go func(name string) {
			defer wg.Done()

			if so.verbose {
				fmt.Printf("🗑️  Undeploying %s...\n", name)
			}

			releaseName := so.getReleaseName(name, runtime)
			if err := so.helmProvider.UninstallChart(ctx, releaseName, namespace); err != nil {
				errorsChan <- fmt.Errorf("%s: %w", name, err)
				fmt.Printf("⚠️  Failed to undeploy %s: %v\n", name, err)
			} else if so.verbose {
				fmt.Printf("✅ %s undeployed\n", name)
			}
		}(serviceName)
	}

	// Wait for all undeployments
	go func() {
		wg.Wait()
		close(errorsChan)
	}()

	// Collect errors (but don't fail - best effort undeployment)
	var errors []error
	for err := range errorsChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("some services failed to undeploy: %d errors", len(errors))
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

// DeployService deploys a single service (public method)
func (so *ServiceOrchestrator) DeployService(ctx context.Context, service *config.ResolvedService, runtime *config.RuntimeConfig) error {
	if so.verbose {
		fmt.Printf("📦 Deploying %s...\n", service.Name)
	}

	if err := so.deployService(ctx, service, runtime); err != nil {
		return err
	}

	if so.verbose {
		fmt.Printf("✅ %s deployed successfully\n", service.Name)
	}

	return nil
}

// UndeployService removes a single service from the environment
func (so *ServiceOrchestrator) UndeployService(ctx context.Context, runtime *config.RuntimeConfig, serviceName string) error {
	namespace := runtime.Base.Defaults.Namespace
	releaseName := so.getReleaseName(serviceName, runtime)

	if so.verbose {
		fmt.Printf("🗑️  Undeploying %s...\n", serviceName)
	}

	if err := so.helmProvider.UninstallChart(ctx, releaseName, namespace); err != nil {
		return fmt.Errorf("failed to undeploy: %w", err)
	}

	if so.verbose {
		fmt.Printf("✅ %s undeployed\n", serviceName)
	}

	return nil
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
		Name:       so.getReleaseName(service.Name, runtime),
		Chart:      service.Chart.Name,
		Version:    service.Chart.Version,
		Repository: service.Chart.Repository,
		Namespace:  runtime.Base.Defaults.Namespace,
		Values:     values,
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

// groupServicesByDependencyLevel groups services by dependency level for concurrent deployment
// Services in the same level have no dependencies on each other and can deploy concurrently
func (so *ServiceOrchestrator) groupServicesByDependencyLevel(runtime *config.RuntimeConfig) ([][]string, error) {
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

	// Group services by level using modified Kahn's algorithm
	var levels [][]string
	processedCount := 0

	for processedCount < len(runtime.ResolvedServices) {
		// Find all services with no remaining dependencies (current level)
		var currentLevel []string
		for service, degree := range inDegree {
			if degree == 0 {
				currentLevel = append(currentLevel, service)
			}
		}

		if len(currentLevel) == 0 {
			return nil, fmt.Errorf("circular dependency detected in services")
		}

		// Sort for deterministic ordering
		sort.Strings(currentLevel)
		levels = append(levels, currentLevel)

		// Remove current level from graph and update in-degrees
		for _, service := range currentLevel {
			inDegree[service] = -1 // Mark as processed
			processedCount++

			// Decrease in-degree for services that depend on this one
			for _, dependency := range graph[service] {
				if inDegree[dependency] > 0 {
					inDegree[dependency]--
				}
			}
		}
	}

	return levels, nil
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
