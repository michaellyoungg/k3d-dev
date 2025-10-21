package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"plat/pkg/orchestrator"
)

// Split pane layout components for home view
// This file contains:
// - Left navigation panel rendering
// - Right detail panel rendering (cluster and service views)
// - Formatting helpers for navigation items and action lists

const navPanelWidth = 30

// Navigation Panel

// renderNavPanel renders the left navigation panel
func (m *Model) renderNavPanel() string {
	var b strings.Builder

	if len(m.navItems) == 0 {
		return dimStyle.Render("No items")
	}

	for i, item := range m.navItems {
		isSelected := i == m.selectedNav
		line := m.formatNavItem(item)
		style := m.navItemStyle(isSelected)

		b.WriteString(style.Render(line))
		b.WriteString("\n")
	}

	// Style the entire nav panel with border
	navStyle := lipgloss.NewStyle().
		Width(navPanelWidth).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		BorderRight(true).
		Height(m.height - 8) // Account for header and footer

	return navStyle.Render(b.String())
}

// formatNavItem formats a navigation item with icon and name
func (m *Model) formatNavItem(item NavItem) string {
	switch item.Type {
	case NavItemCluster:
		if cluster := m.getClusterComponent(); cluster != nil {
			icon := getStatusIcon(cluster.Status)
			return icon + " " + item.Name
		}
		return "🏗️  " + item.Name

	case NavItemService:
		if svc := m.getServiceComponent(item.ServiceName); svc != nil {
			icon := getStatusIcon(svc.Status)
			return icon + " " + item.Name
		}
		return "⚪ " + item.Name

	default:
		return item.Name
	}
}

// navItemStyle returns the style for a navigation item
func (m *Model) navItemStyle(isSelected bool) lipgloss.Style {
	style := lipgloss.NewStyle().
		Width(navPanelWidth - 2).
		Padding(0, 1)

	if isSelected {
		style = style.
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("230"))
	}

	return style
}

// Detail Panel

// renderDetailPanel renders the right detail panel based on selected navigation item
func (m *Model) renderDetailPanel() string {
	item := m.getSelectedNavItem()
	if item == nil {
		return dimStyle.Render("No item selected")
	}

	var content string
	switch item.Type {
	case NavItemCluster:
		content = m.renderClusterDetail()
	case NavItemService:
		content = m.renderServiceDetail(item.ServiceName)
	default:
		content = dimStyle.Render("Unknown item type")
	}

	// Style the detail panel
	detailStyle := lipgloss.NewStyle().
		Width(m.width - navPanelWidth - 4). // Account for borders and padding
		Height(m.height - 8).                // Account for header and footer
		Padding(1, 2)

	return detailStyle.Render(content)
}

// renderClusterDetail renders detailed cluster information
func (m *Model) renderClusterDetail() string {
	var b strings.Builder

	b.WriteString(sectionStyle.Render("Cluster Information"))
	b.WriteString("\n\n")

	comp := m.getClusterComponent()
	if comp == nil {
		b.WriteString(dimStyle.Render("No cluster information available"))
		return b.String()
	}

	// Status
	icon := getStatusIcon(comp.Status)
	statusLine := fmt.Sprintf("%s Status: %s", icon, comp.Status)
	b.WriteString(statusLine)
	b.WriteString("\n\n")

	// Get cluster details from StatusDetail
	if clusterStatus, ok := comp.StatusDetail.(*orchestrator.ClusterStatus); ok && clusterStatus != nil {
		// Name
		if clusterStatus.Name != "" {
			b.WriteString(fmt.Sprintf("Name: %s", clusterStatus.Name))
			b.WriteString("\n")
		}

		// Nodes
		if clusterStatus.Servers > 0 || clusterStatus.Agents > 0 {
			b.WriteString(fmt.Sprintf("Nodes: %d server(s), %d agent(s)", clusterStatus.Servers, clusterStatus.Agents))
			b.WriteString("\n")
		}

		// Error
		if clusterStatus.Error != "" {
			b.WriteString("\n")
			b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %s", clusterStatus.Error)))
			b.WriteString("\n")
		}
	}

	// Environment info
	if m.envName != "" {
		b.WriteString("\n")
		b.WriteString(dimStyle.Render(fmt.Sprintf("Environment: %s", m.envName)))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render(fmt.Sprintf("Mode: %s", m.envMode)))
		b.WriteString("\n")
	}

	// Actions help
	actions := []string{
		"u - Start environment (bring up cluster)",
		"d - Stop services",
		"D - Stop services and delete cluster",
		"r - Refresh status",
	}
	b.WriteString(m.renderActionsHelp(actions))

	return b.String()
}

// renderServiceDetail renders detailed service information
func (m *Model) renderServiceDetail(serviceName string) string {
	var b strings.Builder

	b.WriteString(sectionStyle.Render(fmt.Sprintf("Service: %s", serviceName)))
	b.WriteString("\n\n")

	comp := m.getServiceComponent(serviceName)
	if comp == nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("Service %s not found", serviceName)))
		return b.String()
	}

	// Status
	icon := getStatusIcon(comp.Status)
	statusLine := fmt.Sprintf("%s Status: %s", icon, comp.Status)
	b.WriteString(statusLine)
	b.WriteString("\n\n")

	// Get service details from StatusDetail
	if svcStatus, ok := comp.StatusDetail.(*orchestrator.ServiceStatus); ok && svcStatus != nil {
		// Version
		if svcStatus.Version != "" {
			b.WriteString(fmt.Sprintf("Version: %s", svcStatus.Version))
			b.WriteString("\n")
		}

		// Type
		if svcStatus.IsLocal {
			b.WriteString(activeStyle.Render("Type: Local"))
			b.WriteString("\n")
		} else {
			b.WriteString("Type: Remote (Helm)")
			b.WriteString("\n")
		}

		// Ports
		if len(svcStatus.Ports) > 0 {
			b.WriteString(fmt.Sprintf("Ports: %v", svcStatus.Ports))
			b.WriteString("\n")
		}

		// Chart info (for Helm services)
		if svcStatus.Chart != "" {
			b.WriteString(fmt.Sprintf("Chart: %s", svcStatus.Chart))
			b.WriteString("\n")
		}

		// Updated timestamp
		if svcStatus.Updated != "" {
			b.WriteString(dimStyle.Render(fmt.Sprintf("Updated: %s", svcStatus.Updated)))
			b.WriteString("\n")
		}
	}

	// Actions help
	actions := []string{
		"s - Start service",
		"x - Stop service",
		"R - Restart service",
		"l - View logs",
	}
	b.WriteString(m.renderActionsHelp(actions))

	return b.String()
}

// Helpers

// renderActionsHelp renders the available actions section
func (m *Model) renderActionsHelp(actions []string) string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(sectionStyle.Render("Available Actions:"))
	b.WriteString("\n")
	for _, action := range actions {
		b.WriteString(dimStyle.Render("  " + action))
		b.WriteString("\n")
	}
	return b.String()
}
