package orchestrator

import (
	"context"
	"fmt"

	"devenv/pkg/config"
)

// Orchestrator manages the complete environment lifecycle
type Orchestrator struct {
	clusterManager *ClusterManager
	serviceManager *ServiceOrchestrator
	verbose        bool
}

// NewOrchestrator creates a new orchestrator
func NewOrchestrator(verbose bool) *Orchestrator {
	return &Orchestrator{
		clusterManager: NewClusterManager(verbose),
		serviceManager: NewServiceOrchestrator(verbose),
		verbose:        verbose,
	}
}

// Up brings up the entire environment (cluster + services)
func (o *Orchestrator) Up(ctx context.Context, runtime *config.RuntimeConfig) error {
	if o.verbose {
		fmt.Printf("🚀 Starting environment: %s\n", runtime.Base.Name)
	}

	// 1. Ensure cluster is running
	if err := o.clusterManager.EnsureCluster(ctx, runtime); err != nil {
		return fmt.Errorf("cluster setup failed: %w", err)
	}

	// 2. Deploy services
	if err := o.serviceManager.DeployServices(ctx, runtime); err != nil {
		return fmt.Errorf("service deployment failed: %w", err)
	}

	// 3. Print access information
	o.printEnvironmentInfo(runtime)

	if o.verbose {
		fmt.Printf("✅ Environment %s is ready!\n", runtime.Base.Name)
	}

	return nil
}

// Down brings down the entire environment
func (o *Orchestrator) Down(ctx context.Context, runtime *config.RuntimeConfig, deleteCluster bool) error {
	if o.verbose {
		fmt.Printf("🛑 Stopping environment: %s\n", runtime.Base.Name)
	}

	// 1. Undeploy services first
	if err := o.serviceManager.UndeployServices(ctx, runtime); err != nil {
		fmt.Printf("⚠️  Service undeployment warnings: %v\n", err)
		// Continue to cluster deletion even if some services failed
	}

	// 2. Delete cluster if requested
	if deleteCluster {
		if err := o.clusterManager.DeleteCluster(ctx, runtime); err != nil {
			return fmt.Errorf("cluster deletion failed: %w", err)
		}
	} else if o.verbose {
		fmt.Printf("🔄 Cluster kept running (use --cluster to delete)\n")
	}

	if o.verbose {
		fmt.Printf("✅ Environment %s stopped\n", runtime.Base.Name)
	}

	return nil
}

// Status returns the current status of the environment
func (o *Orchestrator) Status(ctx context.Context, runtime *config.RuntimeConfig) (*EnvironmentStatus, error) {
	status := &EnvironmentStatus{
		Name:     runtime.Base.Name,
		Mode:     string(runtime.Mode),
		Services: make(map[string]*ServiceStatus),
	}

	// Get cluster status
	clusterStatus, err := o.clusterManager.GetClusterStatus(ctx, runtime)
	if err != nil {
		status.Cluster = &ClusterStatus{
			Status: "not-found",
			Error:  err.Error(),
		}
	} else {
		status.Cluster = &ClusterStatus{
			Name:    clusterStatus.Name,
			Status:  clusterStatus.Status,
			Servers: clusterStatus.Servers,
			Agents:  clusterStatus.Agents,
		}
	}

	// Get service statuses
	serviceStatuses, err := o.serviceManager.GetServiceStatuses(ctx, runtime)
	if err != nil {
		return nil, fmt.Errorf("failed to get service statuses: %w", err)
	}

	for serviceName, service := range runtime.ResolvedServices {
		helmStatus := serviceStatuses[serviceName]
		
		serviceStatus := &ServiceStatus{
			Name:     serviceName,
			Status:   helmStatus.Status,
			Version:  service.Version,
			IsLocal:  service.IsLocal,
			Chart:    service.Chart.Name,
			Updated:  helmStatus.Updated,
		}

		if service.IsLocal && service.LocalSource != nil {
			serviceStatus.LocalPath = service.LocalSource.GetPath()
		}

		if len(service.Ports) > 0 {
			serviceStatus.Ports = service.Ports
		}

		status.Services[serviceName] = serviceStatus
	}

	return status, nil
}

// ValidatePrerequisites checks that all required tools are available
func (o *Orchestrator) ValidatePrerequisites(ctx context.Context) error {
	if err := o.clusterManager.ValidatePrerequisites(ctx); err != nil {
		return err
	}

	if err := o.serviceManager.ValidatePrerequisites(ctx); err != nil {
		return err
	}

	return nil
}

// printEnvironmentInfo displays information about how to access the environment
func (o *Orchestrator) printEnvironmentInfo(runtime *config.RuntimeConfig) {
	fmt.Printf("\n🌐 Environment Access Information\n")
	fmt.Printf("=================================\n")
	
	domain := runtime.Base.Defaults.Domain
	
	fmt.Printf("\nServices available at:\n")
	for serviceName, service := range runtime.ResolvedServices {
		if len(service.Ports) > 0 {
			// Show primary port
			port := service.Ports[0]
			if domain != "" {
				fmt.Printf("  • %s: http://%s.%s", serviceName, serviceName, domain)
				if port != 80 {
					fmt.Printf(":%d", port)
				}
				fmt.Printf("\n")
			} else {
				fmt.Printf("  • %s: http://localhost:%d\n", serviceName, port)
			}
		}
	}

	fmt.Printf("\nManagement commands:\n")
	fmt.Printf("  • plat status     - Check environment health\n")
	fmt.Printf("  • plat down       - Stop services\n")
	fmt.Printf("  • plat logs <svc> - View service logs\n")
	
	if runtime.Mode == config.ModeLocal {
		fmt.Printf("\n📝 Local Development:\n")
		for serviceName, service := range runtime.ResolvedServices {
			if service.IsLocal && service.LocalSource != nil {
				fmt.Printf("  • %s: %s\n", serviceName, service.LocalSource.GetPath())
			}
		}
		fmt.Printf("  Changes will be hot-reloaded automatically\n")
	}
	
	fmt.Println()
}

// Status types

type EnvironmentStatus struct {
	Name     string                    `json:"name"`
	Mode     string                    `json:"mode"`
	Cluster  *ClusterStatus           `json:"cluster"`
	Services map[string]*ServiceStatus `json:"services"`
}

type ClusterStatus struct {
	Name    string `json:"name,omitempty"`
	Status  string `json:"status"`
	Servers int    `json:"servers,omitempty"`
	Agents  int    `json:"agents,omitempty"`
	Error   string `json:"error,omitempty"`
}

type ServiceStatus struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	Version   string `json:"version"`
	IsLocal   bool   `json:"is_local"`
	LocalPath string `json:"local_path,omitempty"`
	Chart     string `json:"chart,omitempty"`
	Ports     []int  `json:"ports,omitempty"`
	Updated   string `json:"updated,omitempty"`
}