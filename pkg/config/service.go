package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
)

// Service represents a service definition with union type support
type Service struct {
	// For simple form: just service name (uses :latest)
	Name string `yaml:"-"`
	
	// For complex form: full service configuration
	ServiceName  string                 `yaml:"name,omitempty"`
	Version      string                 `yaml:"version,omitempty"`
	Chart        ServiceChart           `yaml:"chart,omitempty"`
	Values       map[string]interface{} `yaml:"values,omitempty"`
	ValuesFile   string                 `yaml:"values_file,omitempty"`
	Ports        []int                  `yaml:"ports,omitempty"`
	Environment  map[string]string      `yaml:"environment,omitempty"`
	Dependencies []string               `yaml:"dependencies,omitempty"`
}

// ServiceChart defines Helm chart specification
type ServiceChart struct {
	Name       string `yaml:"name"`
	Repository string `yaml:"repository,omitempty"`
	Version    string `yaml:"version,omitempty"`
}

// UnmarshalYAML implements custom unmarshaling for union types
func (s *Service) UnmarshalYAML(node *yaml.Node) error {
	// Try simple form first (just a string)
	var serviceName string
	if err := node.Decode(&serviceName); err == nil {
		s.Name = serviceName
		return nil
	}
	
	// Fall back to complex form
	type serviceAlias Service
	return node.Decode((*serviceAlias)(s))
}

// GetName returns the service name (handles both simple and complex forms)
func (s *Service) GetName() string {
	if s.Name != "" {
		return s.Name
	}
	return s.ServiceName
}

// GetVersion returns the service version (defaults to "latest")
func (s *Service) GetVersion() string {
	if s.Version != "" {
		return s.Version
	}
	return "latest"
}

// IsSimpleForm returns true if this is a simple service definition
func (s *Service) IsSimpleForm() bool {
	return s.Name != ""
}

// LocalSource represents a local source definition with union type support
type LocalSource struct {
	// For simple form: just a path string
	Path string `yaml:"-"`
	
	// For complex form: full local source configuration
	LocalPath  string `yaml:"path,omitempty"`
	Dockerfile string `yaml:"dockerfile,omitempty"`
	Context    string `yaml:"context,omitempty"`
	Chart      string `yaml:"chart,omitempty"`
}

// UnmarshalYAML implements custom unmarshaling for local sources
func (ls *LocalSource) UnmarshalYAML(node *yaml.Node) error {
	// Try simple form first (just a string path)
	var simplePath string
	if err := node.Decode(&simplePath); err == nil {
		ls.Path = simplePath
		return nil
	}
	
	// Fall back to complex form
	type localSourceAlias LocalSource
	return node.Decode((*localSourceAlias)(ls))
}

// GetPath returns the repository path
func (ls *LocalSource) GetPath() string {
	if ls.Path != "" {
		return ls.Path
	}
	return ls.LocalPath
}

// GetDockerfile returns the Dockerfile path (defaults to "Dockerfile")
func (ls *LocalSource) GetDockerfile() string {
	if ls.Path != "" {
		return "Dockerfile" // Convention for simple form
	}
	if ls.Dockerfile != "" {
		return ls.Dockerfile
	}
	return "Dockerfile" // Default
}

// GetContext returns the build context (defaults to repo root)
func (ls *LocalSource) GetContext() string {
	if ls.Path != "" {
		return "." // Convention for simple form
	}
	if ls.Context != "" {
		return ls.Context
	}
	return "." // Default
}

// GetChart returns the chart path (defaults to "chart/")
func (ls *LocalSource) GetChart() string {
	if ls.Path != "" {
		return "chart" // Convention for simple form
	}
	if ls.Chart != "" {
		return ls.Chart
	}
	return "chart" // Default
}

// IsSimpleForm returns true if this is a simple path definition
func (ls *LocalSource) IsSimpleForm() bool {
	return ls.Path != ""
}

// Validate validates the local source configuration
func (ls *LocalSource) Validate() error {
	path := ls.GetPath()
	if path == "" {
		return fmt.Errorf("path is required for local source")
	}
	
	// Additional validation can be added here (file existence, etc.)
	return nil
}