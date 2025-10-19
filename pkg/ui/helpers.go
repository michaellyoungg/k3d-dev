package ui

import (
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Helper functions

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
