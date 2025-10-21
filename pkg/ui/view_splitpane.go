package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Split pane layout with left navigation and right detail view

const navPanelWidth = 30

// renderNavPanel renders the left navigation panel
func (m *Model) renderNavPanel() string {
	var b strings.Builder

	if len(m.navItems) == 0 {
		return dimStyle.Render("No items")
	}

	for i, item := range m.navItems {
		isSelected := i == m.selectedNav

		var icon string
		var line string

		switch item.Type {
		case NavItemCluster:
			icon = "🏗️ "
			line = icon + item.Name
			if m.status != nil && m.status.Cluster != nil {
				statusIcon := getStatusIcon(m.status.Cluster.Status)
				line = statusIcon + " " + item.Name
			}

		case NavItemService:
			if m.status != nil && m.status.Services != nil {
				if svc, ok := m.status.Services[item.ServiceName]; ok {
					statusIcon := getStatusIcon(svc.Status)
					line = statusIcon + " " + item.Name
				} else {
					line = "⚪ " + item.Name
				}
			} else {
				line = "⚪ " + item.Name
			}
		}

		// Apply selection style
		var style lipgloss.Style
		if isSelected {
			style = lipgloss.NewStyle().
				Background(lipgloss.Color("62")).
				Foreground(lipgloss.Color("230")).
				Width(navPanelWidth - 2).
				Padding(0, 1)
		} else {
			style = lipgloss.NewStyle().
				Width(navPanelWidth - 2).
				Padding(0, 1)
		}

		b.WriteString(style.Render(line))
		b.WriteString("\n")
	}

	// Style the entire nav panel
	navStyle := lipgloss.NewStyle().
		Width(navPanelWidth).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		BorderRight(true).
		Height(m.height - 8) // Account for header and footer

	return navStyle.Render(b.String())
}

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

	if m.status == nil || m.status.Cluster == nil {
		b.WriteString(dimStyle.Render("No cluster information available"))
		return b.String()
	}

	cluster := m.status.Cluster

	// Status
	icon := getStatusIcon(cluster.Status)
	statusLine := fmt.Sprintf("%s Status: %s", icon, cluster.Status)
	b.WriteString(statusLine)
	b.WriteString("\n\n")

	// Name
	if cluster.Name != "" {
		b.WriteString(fmt.Sprintf("Name: %s", cluster.Name))
		b.WriteString("\n")
	}

	// Nodes
	if cluster.Servers > 0 || cluster.Agents > 0 {
		b.WriteString(fmt.Sprintf("Nodes: %d server(s), %d agent(s)", cluster.Servers, cluster.Agents))
		b.WriteString("\n")
	}

	// Error
	if cluster.Error != "" {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %s", cluster.Error)))
		b.WriteString("\n")
	}

	// Environment info
	if m.status != nil {
		b.WriteString("\n")
		b.WriteString(dimStyle.Render(fmt.Sprintf("Environment: %s", m.status.Name)))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render(fmt.Sprintf("Mode: %s", m.status.Mode)))
		b.WriteString("\n")
	}

	// Actions help
	b.WriteString("\n")
	b.WriteString(sectionStyle.Render("Available Actions:"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  u - Start environment (bring up cluster)"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  d - Stop services"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  D - Stop services and delete cluster"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  r - Refresh status"))
	b.WriteString("\n")

	return b.String()
}

// renderServiceDetail renders detailed service information
func (m *Model) renderServiceDetail(serviceName string) string {
	var b strings.Builder

	b.WriteString(sectionStyle.Render(fmt.Sprintf("Service: %s", serviceName)))
	b.WriteString("\n\n")

	if m.status == nil || m.status.Services == nil {
		b.WriteString(dimStyle.Render("No service information available"))
		return b.String()
	}

	svc, ok := m.status.Services[serviceName]
	if !ok {
		b.WriteString(errorStyle.Render(fmt.Sprintf("Service %s not found", serviceName)))
		return b.String()
	}

	// Status
	icon := getStatusIcon(svc.Status)
	statusLine := fmt.Sprintf("%s Status: %s", icon, svc.Status)
	b.WriteString(statusLine)
	b.WriteString("\n\n")

	// Version
	if svc.Version != "" {
		b.WriteString(fmt.Sprintf("Version: %s", svc.Version))
		b.WriteString("\n")
	}

	// Type
	if svc.IsLocal {
		b.WriteString(activeStyle.Render("Type: Local"))
		b.WriteString("\n")
	} else {
		b.WriteString("Type: Remote (Helm)")
		b.WriteString("\n")
	}

	// Ports
	if len(svc.Ports) > 0 {
		b.WriteString(fmt.Sprintf("Ports: %v", svc.Ports))
		b.WriteString("\n")
	}

	// Chart info (for Helm services)
	if svc.Chart != "" {
		b.WriteString(fmt.Sprintf("Chart: %s", svc.Chart))
		b.WriteString("\n")
	}

	// Updated timestamp
	if svc.Updated != "" {
		b.WriteString(dimStyle.Render(fmt.Sprintf("Updated: %s", svc.Updated)))
		b.WriteString("\n")
	}

	// Actions help
	b.WriteString("\n")
	b.WriteString(sectionStyle.Render("Available Actions:"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  s - Start service"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  x - Stop service"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  R - Restart service"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  l - View logs"))
	b.WriteString("\n")

	return b.String()
}
