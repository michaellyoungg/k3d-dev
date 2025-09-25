package config

// This file contains legacy type definitions that may still be referenced
// Main configuration types are now in base.go and service.go

// ClusterConfig defines k3d cluster settings (kept for compatibility)
type ClusterConfig struct {
	Name    string    `yaml:"name"`
	K3d     K3dConfig `yaml:"k3d"`
}

// K3dConfig contains k3d-specific settings
type K3dConfig struct {
	Image     string            `yaml:"image,omitempty"`
	Servers   int               `yaml:"servers,omitempty"`
	Agents    int               `yaml:"agents,omitempty"`
	Ports     []string          `yaml:"ports,omitempty"`
	Volumes   []string          `yaml:"volumes,omitempty"`
	Options   []string          `yaml:"options,omitempty"`
	Labels    map[string]string `yaml:"labels,omitempty"`
}