package tools

import (
	"context"
	"io"
)

// ClusterProvider manages Kubernetes cluster lifecycle
type ClusterProvider interface {
	// CreateCluster creates a new k3d cluster
	CreateCluster(ctx context.Context, config ClusterConfig) error

	// DeleteCluster removes a k3d cluster
	DeleteCluster(ctx context.Context, name string) error

	// GetClusterStatus returns current cluster information
	GetClusterStatus(ctx context.Context, name string) (*ClusterStatus, error)

	// ListClusters returns all managed clusters
	ListClusters(ctx context.Context) ([]ClusterInfo, error)
}

// HelmProvider manages Helm chart deployments
type HelmProvider interface {
	// InstallChart installs or upgrades a Helm chart
	InstallChart(ctx context.Context, release HelmRelease) error

	// UninstallChart removes a Helm release
	UninstallChart(ctx context.Context, releaseName, namespace string) error

	// GetReleaseStatus returns status of a Helm release
	GetReleaseStatus(ctx context.Context, releaseName, namespace string) (*ReleaseStatus, error)

	// ListReleases returns all releases in namespace
	ListReleases(ctx context.Context, namespace string) ([]ReleaseInfo, error)
}

// TerraformProvider removed - using k3d + Helm only for simplicity

// ProcessExecutor abstracts external command execution
type ProcessExecutor interface {
	// Execute runs a command and returns output
	Execute(ctx context.Context, cmd Command) (*ExecuteResult, error)

	// Stream runs a command with streaming output
	Stream(ctx context.Context, cmd Command, output io.Writer) error
}

// Configuration types

type ClusterConfig struct {
	Name    string            `yaml:"name"`
	Image   string            `yaml:"image,omitempty"`
	Servers int               `yaml:"servers"`
	Agents  int               `yaml:"agents"`
	Ports   []string          `yaml:"ports,omitempty"`
	Volumes []string          `yaml:"volumes,omitempty"`
	Options []string          `yaml:"options,omitempty"`
	Labels  map[string]string `yaml:"labels,omitempty"`
}

type ClusterStatus struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Servers int    `json:"servers"`
	Agents  int    `json:"agents"`
	Network string `json:"network"`
}

type ClusterInfo struct {
	Name      string            `json:"name"`
	Status    string            `json:"status"`
	CreatedAt string            `json:"created_at"`
	Labels    map[string]string `json:"labels"`
}

type HelmRelease struct {
	Name        string         `yaml:"name"`
	Chart       string         `yaml:"chart"`
	Version     string         `yaml:"version,omitempty"`
	Repository  string         `yaml:"repository,omitempty"`
	Namespace   string         `yaml:"namespace"`
	Values      map[string]any `yaml:"values,omitempty"`
	ValuesFiles []string       `yaml:"values_files,omitempty"`
}

type ReleaseStatus struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Status    string `json:"status"`
	Chart     string `json:"chart"`
	Version   string `json:"version"`
	Updated   string `json:"updated"`
}

type ReleaseInfo struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Status    string `json:"status"`
	Chart     string `json:"chart"`
}

// Terraform types removed - using k3d + Helm only

// Command execution types

type Command struct {
	Name string            `json:"name"`
	Args []string          `json:"args"`
	Dir  string            `json:"dir,omitempty"`
	Env  map[string]string `json:"env,omitempty"`
}

type ExecuteResult struct {
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
}
