package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// View renders the current view based on the model state
func (m *Model) View() string {
	switch m.view {
	case HomeView:
		return m.renderHomeView()
	case ServiceLogsView:
		return m.renderLogsView()
	default:
		return "Unknown view"
	}
}

func (m *Model) renderHeader() string {
	title := headerStyle.Render("🎯 Local Cluster")

	var status string
	if m.loading && m.operation != "" {
		// Show active operation with spinner
		status = activeStyle.Render(m.spinner.View() + " " + m.operation + "...")
	} else if m.message != "" {
		// Show success message
		status = successStyle.Render("✓ " + m.message)
	} else if m.error != nil {
		// Show error
		status = errorStyle.Render("✗ " + m.error.Error())
	}

	// Pad to fill width
	padding := m.width - lipgloss.Width(title) - lipgloss.Width(status) - 4
	if padding < 0 {
		padding = 0
	}

	return title + strings.Repeat(" ", padding) + status
}

func (m *Model) renderFooter() string {
	return m.help.View(m)
}
