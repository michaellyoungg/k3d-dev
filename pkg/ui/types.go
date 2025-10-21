package ui

import (
	"time"
)

// ViewMode represents the current view
type ViewMode int

const (
	HomeView ViewMode = iota
	ServiceLogsView
)

// ComponentType identifies the type of component
type ComponentType int

const (
	ComponentCluster ComponentType = iota
	ComponentService
)

// Component represents a managed component (cluster or service) with separate metadata and status
type Component struct {
	// Static metadata (doesn't change often)
	Type ComponentType
	Name string
	ID   string // Unique identifier (cluster name or service name)

	// Dynamic status (updated frequently)
	Status       string
	LastUpdated  time.Time
	LastChecked  time.Time
	Error        error
	StatusDetail interface{} // *orchestrator.ClusterStatus or *orchestrator.ServiceStatus
}

// ComponentStatus represents just the status portion for updates
type ComponentStatus struct {
	ID          string
	Status      string
	Error       error
	LastChecked time.Time
}
