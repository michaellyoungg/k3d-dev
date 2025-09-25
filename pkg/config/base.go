package config

import "time"

// BaseConfig represents the main .plat/config.yml structure
type BaseConfig struct {
	APIVersion string            `yaml:"apiVersion"`
	Kind       string            `yaml:"kind"`
	Name       string            `yaml:"name"`
	Services   []Service         `yaml:"services"`
	Defaults   *DefaultsConfig   `yaml:"defaults,omitempty"`
}

// LocalConfig represents the .plat/local.yml structure  
type LocalConfig struct {
	LocalSources map[string]LocalSource `yaml:"local_sources"`
}

// DefaultsConfig contains MSC-specific default settings
type DefaultsConfig struct {
	Registry   string `yaml:"registry,omitempty"`
	Domain     string `yaml:"domain,omitempty"`
	Namespace  string `yaml:"namespace,omitempty"`
	Chart      string `yaml:"chart,omitempty"`
}

// RuntimeConfig represents the resolved configuration at runtime
type RuntimeConfig struct {
	Base         *BaseConfig
	Local        *LocalConfig
	Mode         ExecutionMode
	ResolvedServices map[string]*ResolvedService
	Timestamp    time.Time
}

// ResolvedService is a service with all overrides and defaults applied
type ResolvedService struct {
	Name             string
	Version          string
	IsLocal          bool
	LocalSource      *LocalSource
	Chart            ServiceChart
	Values           map[string]interface{}
	ValuesFile       string
	Ports            []int
	Environment      map[string]string
	Dependencies     []string
}

// ExecutionMode defines how services should be executed
type ExecutionMode string

const (
	ModeArtifact ExecutionMode = "artifact"
	ModeLocal    ExecutionMode = "local"
)

// DefaultBaseConfig creates a base config with MSC defaults
func DefaultBaseConfig(name string) *BaseConfig {
	return &BaseConfig{
		APIVersion: "plat/v1",
		Kind:       "Environment", 
		Name:       name,
		Services:   []Service{},
		Defaults: &DefaultsConfig{
			Registry:  "msc-registry.minitab.com",
			Domain:    "platform.local",
			Namespace: "default",
			Chart:     "microservice",
		},
	}
}