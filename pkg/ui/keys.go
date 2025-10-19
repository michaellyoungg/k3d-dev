package ui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// keyMap defines all key bindings for the TUI
type keyMap struct {
	// Navigation
	Up   key.Binding
	Down key.Binding

	// Dashboard actions
	Start   key.Binding
	Stop    key.Binding
	StopAll key.Binding
	Refresh key.Binding
	Logs    key.Binding

	// Logs actions
	ToggleTimestamp key.Binding
	TogglePodName   key.Binding
	Back            key.Binding

	// Global
	Help key.Binding
	Quit key.Binding
}

// ShortHelp returns context-aware short help based on current view
func (m *Model) ShortHelp() []key.Binding {
	switch m.view {
	case HomeView:
		return []key.Binding{m.keys.Start, m.keys.Stop, m.keys.Logs, m.keys.Refresh, m.keys.Quit}
	case ServiceLogsView:
		return []key.Binding{m.keys.Up, m.keys.Down, m.keys.ToggleTimestamp, m.keys.TogglePodName, m.keys.Logs, m.keys.Back, m.keys.Quit}
	default:
		return []key.Binding{}
	}
}

// FullHelp returns context-aware full help based on current view
func (m *Model) FullHelp() [][]key.Binding {
	switch m.view {
	case HomeView:
		return [][]key.Binding{
			{m.keys.Up, m.keys.Down},
			{m.keys.Start, m.keys.Stop, m.keys.StopAll},
			{m.keys.Logs, m.keys.Refresh},
			{m.keys.Help, m.keys.Quit},
		}
	case ServiceLogsView:
		return [][]key.Binding{
			{m.keys.Up, m.keys.Down},
			{m.keys.ToggleTimestamp, m.keys.TogglePodName},
			{m.keys.Logs, m.keys.Back, m.keys.Help, m.keys.Quit},
		}
	}
	return [][]key.Binding{}
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "move down"),
	),
	Start: key.NewBinding(
		key.WithKeys("u"),
		key.WithHelp("u", "start env"),
	),
	Stop: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "stop services"),
	),
	StopAll: key.NewBinding(
		key.WithKeys("D"),
		key.WithHelp("D", "stop + delete cluster"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),
	Logs: key.NewBinding(
		key.WithKeys("l"),
		key.WithHelp("l", "view logs"),
	),
	ToggleTimestamp: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "toggle timestamps"),
	),
	TogglePodName: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "toggle pod names"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

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
