package ui

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"plat/pkg/orchestrator"
)

// Helper functions

// suppressOutput redirects stdout/stderr to null during execution
func suppressOutput(fn func() error) error {
	// Save original stdout/stderr
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	// Redirect to null (open for writing)
	null, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0666)
	if err != nil {
		return fn() // If we can't open null, just run normally
	}
	defer null.Close()

	os.Stdout = null
	os.Stderr = null

	// Restore after execution
	defer func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	}()

	return fn()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// getSortedServiceNames returns service names in alphabetical order for stable display
func (m *Model) getSortedServiceNames() []string {
	names := make([]string, 0)
	for id, comp := range m.components {
		if comp.Type == ComponentService {
			names = append(names, id)
		}
	}
	sort.Strings(names)
	return names
}

func getStatusIcon(status string) string {
	switch strings.ToLower(status) {
	case "running", "deployed":
		return "✅"
	case "starting", "pending-install", "pending-upgrade":
		return "⏳"
	case "failed", "error":
		return "❌"
	case "stopped", "not-deployed", "not-found":
		return "⏸️"
	default:
		return "⚠️"
	}
}

func tickEvery(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func clearMessageAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return clearMsg{}
	})
}

func (m *Model) createViewport(width, height int) viewport.Model {
	vp := viewport.New(width, height)
	vp.Style = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62"))
	return vp
}

// buildNavItems creates navigation items from current components
func (m *Model) buildNavItems() []NavItem {
	items := []NavItem{}

	// Add cluster as first item
	if cluster := m.getClusterComponent(); cluster != nil {
		clusterName := "Cluster"
		if cluster.Name != "" {
			clusterName = cluster.Name
		}
		items = append(items, NavItem{
			Type: NavItemCluster,
			Name: clusterName,
		})
	}

	// Add services in alphabetical order
	serviceNames := m.getSortedServiceNames()
	for _, name := range serviceNames {
		items = append(items, NavItem{
			Type:        NavItemService,
			Name:        name,
			ServiceName: name,
		})
	}

	return items
}

// getSelectedNavItem returns the currently selected navigation item
func (m *Model) getSelectedNavItem() *NavItem {
	if m.selectedNav < 0 || m.selectedNav >= len(m.navItems) {
		return nil
	}
	return &m.navItems[m.selectedNav]
}

// Component access helpers

// getComponent returns a component by ID
func (m *Model) getComponent(id string) *Component {
	return m.components[id]
}

// getClusterComponent returns the cluster component
func (m *Model) getClusterComponent() *Component {
	for _, c := range m.components {
		if c.Type == ComponentCluster {
			return c
		}
	}
	return nil
}

// getServiceComponent returns a service component by name
func (m *Model) getServiceComponent(name string) *Component {
	return m.components[name]
}

// updateComponentStatus updates just the status portion of a component
func (m *Model) updateComponentStatus(id string, status string, err error) {
	if comp := m.components[id]; comp != nil {
		comp.Status = status
		comp.Error = err
		comp.LastChecked = time.Now()
	}
}

// syncComponentsFromStatus syncs components from orchestrator.EnvironmentStatus
// This is a migration helper to populate the new component model from the old status
func (m *Model) syncComponentsFromStatus(status *orchestrator.EnvironmentStatus) {
	now := time.Now()

	// Update environment metadata
	m.envName = status.Name
	m.envMode = status.Mode

	// Sync cluster component
	if status.Cluster != nil {
		clusterID := "cluster"
		if existing := m.components[clusterID]; existing == nil {
			// Create new cluster component
			m.components[clusterID] = &Component{
				Type:         ComponentCluster,
				Name:         status.Cluster.Name,
				ID:           clusterID,
				Status:       status.Cluster.Status,
				LastUpdated:  now,
				LastChecked:  now,
				StatusDetail: status.Cluster,
			}
		} else {
			// Update existing cluster component
			existing.Status = status.Cluster.Status
			existing.LastChecked = now
			existing.StatusDetail = status.Cluster
			if status.Cluster.Error != "" {
				existing.Error = fmt.Errorf("%s", status.Cluster.Error)
			}
		}
	}

	// Sync service components
	for name, svc := range status.Services {
		if existing := m.components[name]; existing == nil {
			// Create new service component
			m.components[name] = &Component{
				Type:         ComponentService,
				Name:         svc.Name,
				ID:           name,
				Status:       svc.Status,
				LastUpdated:  now,
				LastChecked:  now,
				StatusDetail: svc,
			}
		} else {
			// Update existing service component
			existing.Status = svc.Status
			existing.LastChecked = now
			existing.StatusDetail = svc
		}
	}

	// Remove components that no longer exist in status
	for id, comp := range m.components {
		if comp.Type == ComponentService {
			if _, exists := status.Services[id]; !exists {
				delete(m.components, id)
			}
		}
	}
}
