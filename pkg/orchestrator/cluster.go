package orchestrator

import (
	"context"
	"fmt"
	"time"

	"plat/pkg/config"
	"plat/pkg/tools"
)

// ClusterManager orchestrates k3d cluster lifecycle for plat environments
type ClusterManager struct {
	provider tools.ClusterProvider
	verbose  bool
}

// NewClusterManager creates a new cluster manager
func NewClusterManager(verbose bool) *ClusterManager {
	return &ClusterManager{
		provider: tools.NewK3dProvider(),
		verbose:  verbose,
	}
}

// EnsureCluster ensures the cluster exists and is running for the environment
func (cm *ClusterManager) EnsureCluster(ctx context.Context, runtime *config.RuntimeConfig) error {
	clusterName := cm.getClusterName(runtime)
	
	if cm.verbose {
		fmt.Printf("🔍 Checking cluster: %s\n", clusterName)
	}

	// Check if cluster already exists
	status, err := cm.provider.GetClusterStatus(ctx, clusterName)
	if err == nil && status.Status == "running" {
		if cm.verbose {
			fmt.Printf("✅ Cluster %s is already running (%d servers, %d agents)\n", 
				clusterName, status.Servers, status.Agents)
		}
		return nil
	}

	// Create cluster if it doesn't exist or isn't running
	if cm.verbose {
		fmt.Printf("🚀 Creating k3d cluster: %s\n", clusterName)
	}

	clusterConfig := cm.buildClusterConfig(runtime)
	if err := cm.provider.CreateCluster(ctx, clusterConfig); err != nil {
		return fmt.Errorf("failed to create cluster: %w", err)
	}

	// Wait for cluster to be ready
	if err := cm.waitForClusterReady(ctx, clusterName); err != nil {
		return fmt.Errorf("cluster failed to become ready: %w", err)
	}

	if cm.verbose {
		fmt.Printf("✅ Cluster %s is ready\n", clusterName)
	}

	return nil
}

// DeleteCluster removes the cluster for the environment
func (cm *ClusterManager) DeleteCluster(ctx context.Context, runtime *config.RuntimeConfig) error {
	clusterName := cm.getClusterName(runtime)
	
	if cm.verbose {
		fmt.Printf("🗑️  Deleting cluster: %s\n", clusterName)
	}

	if err := cm.provider.DeleteCluster(ctx, clusterName); err != nil {
		return fmt.Errorf("failed to delete cluster: %w", err)
	}

	if cm.verbose {
		fmt.Printf("✅ Cluster %s deleted\n", clusterName)
	}

	return nil
}

// GetClusterStatus returns the current cluster status
func (cm *ClusterManager) GetClusterStatus(ctx context.Context, runtime *config.RuntimeConfig) (*tools.ClusterStatus, error) {
	clusterName := cm.getClusterName(runtime)
	return cm.provider.GetClusterStatus(ctx, clusterName)
}

// ListClusters returns all plat-managed clusters
func (cm *ClusterManager) ListClusters(ctx context.Context) ([]tools.ClusterInfo, error) {
	allClusters, err := cm.provider.ListClusters(ctx)
	if err != nil {
		return nil, err
	}

	// Filter to only plat-managed clusters
	var platClusters []tools.ClusterInfo
	for _, cluster := range allClusters {
		if cm.isPlatCluster(cluster.Name) {
			platClusters = append(platClusters, cluster)
		}
	}

	return platClusters, nil
}

// getClusterName generates a consistent cluster name from environment config
func (cm *ClusterManager) getClusterName(runtime *config.RuntimeConfig) string {
	// Use environment name with plat prefix for consistency
	return fmt.Sprintf("plat-%s", runtime.Base.Name)
}

// isPlatCluster checks if a cluster name indicates it's managed by plat
func (cm *ClusterManager) isPlatCluster(name string) bool {
	return len(name) > 5 && name[:5] == "plat-"
}

// buildClusterConfig creates k3d cluster configuration from environment config
func (cm *ClusterManager) buildClusterConfig(runtime *config.RuntimeConfig) tools.ClusterConfig {
	clusterName := cm.getClusterName(runtime)
	
	config := tools.ClusterConfig{
		Name:    clusterName,
		Servers: 1, // Single server for local development
		Agents:  0, // No agents needed for local dev
		Ports: []string{
			// Standard web traffic
			"80:80@loadbalancer",
			"443:443@loadbalancer",
		},
		Options: []string{
			// Disable default traefik since we'll use nginx ingress
			"--k3s-arg=--disable=traefik@server:0",
		},
		Labels: map[string]string{
			"plat.env":       runtime.Base.Name,
			"plat.domain":    runtime.Base.Defaults.Domain,
			"plat.namespace": runtime.Base.Defaults.Namespace,
		},
	}

	// Add additional port mappings for services that need them
	servicePorts := cm.collectServicePorts(runtime)
	for _, port := range servicePorts {
		portMapping := fmt.Sprintf("%d:%d@loadbalancer", port, port)
		config.Ports = append(config.Ports, portMapping)
	}

	return config
}

// collectServicePorts gathers unique ports needed by services
func (cm *ClusterManager) collectServicePorts(runtime *config.RuntimeConfig) []int {
	portSet := make(map[int]bool)
	
	for _, service := range runtime.ResolvedServices {
		for _, port := range service.Ports {
			if port > 0 && port != 80 && port != 443 {
				portSet[port] = true
			}
		}
	}

	// Convert to sorted slice
	var ports []int
	for port := range portSet {
		ports = append(ports, port)
	}

	return ports
}

// waitForClusterReady waits for the cluster to be fully operational
func (cm *ClusterManager) waitForClusterReady(ctx context.Context, clusterName string) error {
	timeout := 60 * time.Second
	interval := 2 * time.Second
	
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for cluster %s to be ready", clusterName)
		case <-ticker.C:
			status, err := cm.provider.GetClusterStatus(ctx, clusterName)
			if err != nil {
				if cm.verbose {
					fmt.Printf("⏳ Waiting for cluster (error: %v)\n", err)
				}
				continue
			}

			if status.Status == "running" {
				return nil
			}

			if cm.verbose {
				fmt.Printf("⏳ Cluster status: %s\n", status.Status)
			}
		}
	}
}

// ValidatePrerequisites checks that k3d is available
func (cm *ClusterManager) ValidatePrerequisites(ctx context.Context) error {
	if err := tools.ValidateK3d(ctx); err != nil {
		return fmt.Errorf("k3d validation failed: %w", err)
	}
	return nil
}