package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// K3dProvider implements ClusterProvider for k3d
type K3dProvider struct {
	executor ProcessExecutor
}

// NewK3dProvider creates a new k3d provider
func NewK3dProvider() ClusterProvider {
	return &K3dProvider{
		executor: NewProcessExecutor(),
	}
}

// CreateCluster creates a new k3d cluster
func (k *K3dProvider) CreateCluster(ctx context.Context, config ClusterConfig) error {
	args := []string{"cluster", "create", config.Name}

	if config.Image != "" {
		args = append(args, "--image", config.Image)
	}

	if config.Servers > 0 {
		args = append(args, "--servers", fmt.Sprintf("%d", config.Servers))
	}

	if config.Agents > 0 {
		args = append(args, "--agents", fmt.Sprintf("%d", config.Agents))
	}

	// Add port mappings
	for _, port := range config.Ports {
		args = append(args, "--port", port)
	}

	// Add volume mappings
	for _, volume := range config.Volumes {
		args = append(args, "--volume", volume)
	}

	// Add additional options
	args = append(args, config.Options...)

	cmd := Command{
		Name: "k3d",
		Args: args,
	}

	_, err := k.executor.Execute(ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to create k3d cluster: %w", err)
	}

	return nil
}

// DeleteCluster removes a k3d cluster
func (k *K3dProvider) DeleteCluster(ctx context.Context, name string) error {
	cmd := Command{
		Name: "k3d",
		Args: []string{"cluster", "delete", name},
	}

	_, err := k.executor.Execute(ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to delete k3d cluster: %w", err)
	}

	return nil
}

// GetClusterStatus returns current cluster information
func (k *K3dProvider) GetClusterStatus(ctx context.Context, name string) (*ClusterStatus, error) {
	cmd := Command{
		Name: "k3d",
		Args: []string{"cluster", "get", name, "-o", "json"},
	}

	result, err := k.executor.Execute(ctx, cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to get k3d cluster status: %w", err)
	}

	// Parse k3d JSON output into our status structure
	var k3dClusters []map[string]any
	if err := json.Unmarshal([]byte(result.Stdout), &k3dClusters); err != nil {
		return nil, fmt.Errorf("failed to parse k3d cluster info: %w", err)
	}

	if len(k3dClusters) == 0 {
		return nil, fmt.Errorf("cluster %s not found", name)
	}

	cluster := k3dClusters[0]
	status := &ClusterStatus{
		Name:   name,
		Status: "unknown",
	}

	// Extract relevant information from k3d output
	if nodes, ok := cluster["nodes"].([]any); ok {
		serverCount := 0
		agentCount := 0

		for _, node := range nodes {
			if nodeMap, ok := node.(map[string]any); ok {
				if role, ok := nodeMap["role"].(string); ok {
					if strings.Contains(role, "server") {
						serverCount++
					} else if strings.Contains(role, "agent") {
						agentCount++
					}
				}
			}
		}

		status.Servers = serverCount
		status.Agents = agentCount
	}

	// Determine overall cluster status based on node states
	status.Status = "running" // Simplified - would need to check individual node states

	return status, nil
}

// ListClusters returns all managed clusters
func (k *K3dProvider) ListClusters(ctx context.Context) ([]ClusterInfo, error) {
	cmd := Command{
		Name: "k3d",
		Args: []string{"cluster", "list", "-o", "json"},
	}

	result, err := k.executor.Execute(ctx, cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to list k3d clusters: %w", err)
	}

	var k3dClusters []map[string]any
	if err := json.Unmarshal([]byte(result.Stdout), &k3dClusters); err != nil {
		return nil, fmt.Errorf("failed to parse k3d cluster list: %w", err)
	}

	clusters := make([]ClusterInfo, 0, len(k3dClusters))

	for _, cluster := range k3dClusters {
		info := ClusterInfo{}

		if name, ok := cluster["name"].(string); ok {
			info.Name = name
		}

		// Extract status and other information as available
		info.Status = "running" // Simplified

		clusters = append(clusters, info)
	}

	return clusters, nil
}

// ValidateK3d checks if k3d is available and returns version
func ValidateK3d(ctx context.Context) error {
	if err := ValidateCommand("k3d"); err != nil {
		return err
	}

	version, err := GetCommandVersion(ctx, "k3d", "version")
	if err != nil {
		return fmt.Errorf("failed to get k3d version: %w", err)
	}

	fmt.Printf("Found k3d: %s\n", version)
	return nil
}
