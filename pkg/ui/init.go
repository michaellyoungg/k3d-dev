package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Init initializes the model and returns initial commands
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.refreshStatus(),
		tickEvery(5*time.Second),
	)
}
