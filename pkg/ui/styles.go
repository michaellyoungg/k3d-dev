package ui

import "github.com/charmbracelet/lipgloss"

// Shared styles for the TUI
var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("99"))

	sectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	activeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
)
