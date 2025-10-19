package ui

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type item struct {
	title, desc, command string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

type menuModel struct {
	list     list.Model
	quitting bool
	err      error
}

func (m menuModel) Init() tea.Cmd {
	return nil
}

func (m menuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			// Get selected item and execute command
			i, ok := m.list.SelectedItem().(item)
			if ok {
				m.quitting = true
				return m, tea.Sequence(
					tea.Quit,
					executeCommand(i.command),
				)
			}
		}

	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m menuModel) View() string {
	if m.quitting {
		return ""
	}
	return docStyle.Render(m.list.View())
}

func executeCommand(command string) tea.Cmd {
	return func() tea.Msg {
		// Execute the plat command
		cmd := exec.Command("plat", command)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
		}

		return nil
	}
}

// RunMenu launches the interactive TUI menu
func RunMenu() error {
	items := []list.Item{
		item{
			title:   "🚀 Start Environment",
			desc:    "Start all services with k3d and Helm",
			command: "up",
		},
		item{
			title:   "⏸️  Stop Services",
			desc:    "Stop services (keep cluster running)",
			command: "down",
		},
		item{
			title:   "🗑️  Stop & Delete Cluster",
			desc:    "Stop services and delete k3d cluster",
			command: "down --cluster --confirm",
		},
		item{
			title:   "📊 Status",
			desc:    "View environment and service status",
			command: "status",
		},
		item{
			title:   "📋 Logs",
			desc:    "View service logs (will prompt for service)",
			command: "logs",
		},
		item{
			title:   "🔧 Config",
			desc:    "View configuration",
			command: "config show",
		},
		item{
			title:   "🩺 Doctor",
			desc:    "Check system prerequisites",
			command: "doctor",
		},
	}

	m := menuModel{
		list: list.New(items, list.NewDefaultDelegate(), 0, 0),
	}
	m.list.Title = "🎯 Plat - Local Development Environment"
	m.list.Styles.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("99")).
		MarginLeft(2)

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}
