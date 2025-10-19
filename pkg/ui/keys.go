package ui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// Key handling logic

func (m *Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global keys (work in all views)
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Help):
		m.help.ShowAll = !m.help.ShowAll
		return m, nil
	}

	// View-specific keys
	switch m.view {
	case ServiceLogsView:
		return m.handleLogsKeys(msg)
	case HomeView:
		return m.handleDashboardKeys(msg)
	default:
		return m.handleDashboardKeys(msg)
	}
}
