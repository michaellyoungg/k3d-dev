package ui

import (
	"os"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	if m.status == nil || len(m.status.Services) == 0 {
		return nil
	}

	names := make([]string, 0, len(m.status.Services))
	for name := range m.status.Services {
		names = append(names, name)
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

// buildNavItems creates navigation items from current status
func (m *Model) buildNavItems() []NavItem {
	items := []NavItem{}

	// Always add cluster as first item
	clusterName := "Cluster"
	if m.status != nil && m.status.Cluster != nil && m.status.Cluster.Name != "" {
		clusterName = m.status.Cluster.Name
	}
	items = append(items, NavItem{
		Type: NavItemCluster,
		Name: clusterName,
	})

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
