package ui

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// Update handles all incoming messages and updates the model state
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width

		// Update viewport dimensions if it's been initialized
		if m.logsInitialized {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 10
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case statusRefreshMsg:
		m.loading = false
		if msg.err != nil {
			m.error = msg.err
		} else {
			m.status = msg.status
			m.error = nil
			// Rebuild navigation items when status changes
			m.navItems = m.buildNavItems()
		}
		m.lastRefresh = time.Now()
		return m, nil

	case actionCompleteMsg:
		m.loading = false
		m.operation = ""
		m.message = msg.message
		if msg.err != nil {
			m.error = msg.err
		}
		return m, tea.Batch(
			m.refreshStatus(),
			clearMessageAfter(3*time.Second),
		)

	case tickMsg:
		// Auto-refresh every 5 seconds
		return m, tea.Batch(
			m.refreshStatus(),
			tickEvery(5*time.Second),
		)

	case clearMsg:
		m.message = ""
		return m, nil

	case logsMsg:
		return m.handleLogsMsg(msg)

	case logStreamMsg:
		return m.handleLogStreamMsg(msg)

	case logStreamErrorMsg:
		return m.handleLogStreamErrorMsg(msg)
	}

	return m, nil
}
