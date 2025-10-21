package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// HelmClient implements HelmProvider for Helm CLI
type HelmClient struct {
	executor ProcessExecutor
}

// NewHelmProvider creates a new Helm provider
func NewHelmProvider() HelmProvider {
	return &HelmClient{
		executor: NewProcessExecutor(),
	}
}

// InstallChart installs or upgrades a Helm chart
func (h *HelmClient) InstallChart(ctx context.Context, release HelmRelease) error {
	args := []string{"upgrade", "--install", release.Name}

	chartRef := release.Chart

	// Add repository if specified
	if release.Repository != "" {
		// Add repository first if it's a URL
		if strings.HasPrefix(release.Repository, "http") {
			repoName := fmt.Sprintf("plat-%s", release.Name)
			if err := h.addRepository(ctx, repoName, release.Repository); err != nil {
				return fmt.Errorf("failed to add helm repository: %w", err)
			}
			// Update chart reference to use repository
			chartRef = fmt.Sprintf("%s/%s", repoName, release.Chart)
		}
	} else {
		// No repository specified - chart must be a local path or from a configured repo
		// Check if it's a valid chart reference
		if !strings.Contains(release.Chart, "/") && !strings.HasPrefix(release.Chart, ".") {
			return fmt.Errorf("chart '%s' needs a repository. Either:\n  • Add a 'repository' field to the service config\n  • Use 'repo/chart' format (e.g., 'stable/nginx')\n  • Provide a local chart path", release.Chart)
		}
	}

	// Add chart reference
	args = append(args, chartRef)

	// Add version if specified
	if release.Version != "" {
		args = append(args, "--version", release.Version)
	}

	// Add namespace
	args = append(args, "--namespace", release.Namespace)
	args = append(args, "--create-namespace")

	// Add values files
	for _, valuesFile := range release.ValuesFiles {
		args = append(args, "--values", valuesFile)
	}

	// Add inline values
	if len(release.Values) > 0 {
		valuesFile, err := h.createTempValuesFile(release.Values)
		if err != nil {
			return fmt.Errorf("failed to create temporary values file: %w", err)
		}
		defer os.Remove(valuesFile)

		args = append(args, "--values", valuesFile)
	}

	// Add common options for better UX
	args = append(args, "--wait", "--timeout", "300s")

	cmd := Command{
		Name: "helm",
		Args: args,
	}

	result, err := h.executor.Execute(ctx, cmd)
	if err != nil {
		return fmt.Errorf("helm install failed (exit code %d): %s", result.ExitCode, result.Stderr)
	}

	return nil
}

// UninstallChart removes a Helm release
func (h *HelmClient) UninstallChart(ctx context.Context, releaseName, namespace string) error {
	args := []string{"uninstall", releaseName}

	if namespace != "" {
		args = append(args, "--namespace", namespace)
	}

	cmd := Command{
		Name: "helm",
		Args: args,
	}

	result, err := h.executor.Execute(ctx, cmd)
	if err != nil {
		// Check if release doesn't exist (not an error for our use case)
		if strings.Contains(result.Stderr, "not found") {
			return nil
		}
		return fmt.Errorf("helm uninstall failed: %s", result.Stderr)
	}

	return nil
}

// GetReleaseStatus returns status of a Helm release
func (h *HelmClient) GetReleaseStatus(ctx context.Context, releaseName, namespace string) (*ReleaseStatus, error) {
	args := []string{"status", releaseName, "--output", "json"}

	if namespace != "" {
		args = append(args, "--namespace", namespace)
	}

	cmd := Command{
		Name: "helm",
		Args: args,
	}

	result, err := h.executor.Execute(ctx, cmd)
	if err != nil {
		if strings.Contains(result.Stderr, "not found") {
			return nil, fmt.Errorf("release %s not found", releaseName)
		}
		return nil, fmt.Errorf("failed to get helm status: %s", result.Stderr)
	}

	var helmStatus map[string]any
	if err := json.Unmarshal([]byte(result.Stdout), &helmStatus); err != nil {
		return nil, fmt.Errorf("failed to parse helm status output: %w", err)
	}

	status := &ReleaseStatus{
		Name:      releaseName,
		Namespace: namespace,
		Status:    "unknown",
	}

	// Extract status information
	if info, ok := helmStatus["info"].(map[string]any); ok {
		if statusInfo, ok := info["status"].(string); ok {
			status.Status = strings.ToLower(statusInfo)
		}
		if lastDeployed, ok := info["last_deployed"].(string); ok {
			status.Updated = lastDeployed
		}
	}

	// Extract chart information
	if chart, ok := helmStatus["chart"].(map[string]any); ok {
		if metadata, ok := chart["metadata"].(map[string]any); ok {
			if name, ok := metadata["name"].(string); ok {
				if version, ok := metadata["version"].(string); ok {
					status.Chart = fmt.Sprintf("%s-%s", name, version)
				} else {
					status.Chart = name
				}
			}
			if version, ok := metadata["version"].(string); ok {
				status.Version = version
			}
		}
	}

	return status, nil
}

// ListReleases returns all releases in namespace
func (h *HelmClient) ListReleases(ctx context.Context, namespace string) ([]ReleaseInfo, error) {
	args := []string{"list", "--output", "json"}

	if namespace != "" {
		args = append(args, "--namespace", namespace)
	} else {
		args = append(args, "--all-namespaces")
	}

	cmd := Command{
		Name: "helm",
		Args: args,
	}

	result, err := h.executor.Execute(ctx, cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to list helm releases: %s", result.Stderr)
	}

	var helmReleases []map[string]any
	if err := json.Unmarshal([]byte(result.Stdout), &helmReleases); err != nil {
		return nil, fmt.Errorf("failed to parse helm list output: %w", err)
	}

	releases := make([]ReleaseInfo, 0, len(helmReleases))

	for _, release := range helmReleases {
		info := ReleaseInfo{
			Status: "unknown",
		}

		if name, ok := release["name"].(string); ok {
			info.Name = name
		}
		if ns, ok := release["namespace"].(string); ok {
			info.Namespace = ns
		}
		if status, ok := release["status"].(string); ok {
			info.Status = strings.ToLower(status)
		}
		if chart, ok := release["chart"].(string); ok {
			info.Chart = chart
		}

		releases = append(releases, info)
	}

	return releases, nil
}

// addRepository adds a Helm repository
func (h *HelmClient) addRepository(ctx context.Context, name, url string) error {
	// Check if repository already exists
	if exists, err := h.repositoryExists(ctx, name); err != nil {
		return err
	} else if exists {
		return nil // Repository already exists
	}

	cmd := Command{
		Name: "helm",
		Args: []string{"repo", "add", name, url},
	}

	result, err := h.executor.Execute(ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to add repository %s: %s", name, result.Stderr)
	}

	// Update repository index
	updateCmd := Command{
		Name: "helm",
		Args: []string{"repo", "update"},
	}

	_, err = h.executor.Execute(ctx, updateCmd)
	if err != nil {
		// Non-fatal error - continue
		fmt.Printf("Warning: failed to update helm repositories: %v\n", err)
	}

	return nil
}

// repositoryExists checks if a Helm repository is already configured
func (h *HelmClient) repositoryExists(ctx context.Context, name string) (bool, error) {
	cmd := Command{
		Name: "helm",
		Args: []string{"repo", "list", "--output", "json"},
	}

	result, err := h.executor.Execute(ctx, cmd)
	if err != nil {
		// If no repositories exist, helm returns exit code 1
		if strings.Contains(result.Stderr, "no repositories") {
			return false, nil
		}
		return false, fmt.Errorf("failed to list repositories: %s", result.Stderr)
	}

	var repos []map[string]any
	if err := json.Unmarshal([]byte(result.Stdout), &repos); err != nil {
		return false, fmt.Errorf("failed to parse repository list: %w", err)
	}

	for _, repo := range repos {
		if repoName, ok := repo["name"].(string); ok && repoName == name {
			return true, nil
		}
	}

	return false, nil
}

// createTempValuesFile creates a temporary YAML file with the given values
func (h *HelmClient) createTempValuesFile(values map[string]any) (string, error) {
	// Create temporary file
	tempFile, err := os.CreateTemp("", "plat-values-*.yaml")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tempFile.Close()

	// Convert values to YAML using proper library
	yamlData, err := yaml.Marshal(values)
	if err != nil {
		return "", fmt.Errorf("failed to convert values to YAML: %w", err)
	}

	if _, err := tempFile.Write(yamlData); err != nil {
		return "", fmt.Errorf("failed to write values to temp file: %w", err)
	}

	return tempFile.Name(), nil
}

// ValidateHelm checks if Helm is available and returns version
func ValidateHelm(ctx context.Context) error {
	if err := ValidateCommand("helm"); err != nil {
		return err
	}

	version, err := GetCommandVersion(ctx, "helm", "version", "--short")
	if err != nil {
		return fmt.Errorf("failed to get helm version: %w", err)
	}

	fmt.Printf("Found helm: %s\n", version)
	return nil
}
