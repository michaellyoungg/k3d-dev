package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"plat/pkg/orchestrator"
)

func (m *Model) renderHomeView() string {
	var b strings.Builder

	// Header
	b.WriteString(m.renderHeader())
	b.WriteString("\n\n")

	// Main home content
	b.WriteString(m.renderHome())

	// Footer
	b.WriteString("\n\n")
	b.WriteString(m.renderFooter())

	return b.String()
}

func (m *Model) renderHome() string {
	var b strings.Builder

	if m.status == nil {
		b.WriteString(dimStyle.Render("Loading status..."))
		return b.String()
	}

	// Build navigation items if not already built
	if len(m.navItems) == 0 {
		m.navItems = m.buildNavItems()
	}

	// Render split pane: navigation on left, detail on right
	navPanel := m.renderNavPanel()
	detailPanel := m.renderDetailPanel()

	// Join panels side by side
	splitView := lipgloss.JoinHorizontal(lipgloss.Top, navPanel, detailPanel)

	b.WriteString(splitView)

	return b.String()
}

// Key handling for home view

func (m *Model) handleHomeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		if len(m.navItems) > 0 {
			m.selectedNav = max(0, m.selectedNav-1)
		}
		return m, nil

	case key.Matches(msg, m.keys.Down):
		if len(m.navItems) > 0 {
			m.selectedNav = min(len(m.navItems)-1, m.selectedNav+1)
		}
		return m, nil

	case key.Matches(msg, m.keys.Refresh):
		m.loading = true
		return m, m.refreshStatus()

	case key.Matches(msg, m.keys.Start):
		m.loading = true
		m.operation = "Starting environment"
		m.message = ""
		m.error = nil
		return m, m.startEnvironment()

	case key.Matches(msg, m.keys.Stop):
		m.loading = true
		m.operation = "Stopping services"
		m.message = ""
		m.error = nil
		return m, m.stopServices(false)

	case key.Matches(msg, m.keys.StopAll):
		m.loading = true
		m.operation = "Stopping services and deleting cluster"
		m.message = ""
		m.error = nil
		return m, m.stopServices(true)

	case key.Matches(msg, m.keys.Logs):
		// Get selected navigation item
		item := m.getSelectedNavItem()
		if item != nil && item.Type == NavItemService {
			m.logService = item.ServiceName
			m.view = ServiceLogsView
			return m, m.fetchLogs(item.ServiceName)
		}
		return m, nil

	case key.Matches(msg, m.keys.StartService):
		// Get selected navigation item
		item := m.getSelectedNavItem()
		if item != nil && item.Type == NavItemService {
			m.loading = true
			m.operation = fmt.Sprintf("Starting service: %s", item.ServiceName)
			m.message = ""
			m.error = nil
			return m, m.startService(item.ServiceName)
		}
		return m, nil

	case key.Matches(msg, m.keys.StopService):
		// Get selected navigation item
		item := m.getSelectedNavItem()
		if item != nil && item.Type == NavItemService {
			m.loading = true
			m.operation = fmt.Sprintf("Stopping service: %s", item.ServiceName)
			m.message = ""
			m.error = nil
			return m, m.stopService(item.ServiceName)
		}
		return m, nil

	case key.Matches(msg, m.keys.RestartService):
		// Get selected navigation item
		item := m.getSelectedNavItem()
		if item != nil && item.Type == NavItemService {
			m.loading = true
			m.operation = fmt.Sprintf("Restarting service: %s", item.ServiceName)
			m.message = ""
			m.error = nil
			return m, m.restartService(item.ServiceName)
		}
		return m, nil
	}

	return m, nil
}

// Home commands

func (m *Model) refreshStatus() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var status *orchestrator.EnvironmentStatus
		var err error

		// Suppress output during status check
		suppressOutput(func() error {
			status, err = m.orch.Status(ctx, m.runtime)
			return nil
		})

		return statusRefreshMsg{status: status, err: err}
	}
}

func (m *Model) startEnvironment() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		var err error
		suppressOutput(func() error {
			err = m.orch.Up(ctx, m.runtime)
			return nil
		})

		if err != nil {
			return actionCompleteMsg{err: err}
		}

		return actionCompleteMsg{message: "Environment started successfully"}
	}
}

func (m *Model) stopServices(deleteCluster bool) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		var err error
		suppressOutput(func() error {
			err = m.orch.Down(ctx, m.runtime, deleteCluster)
			return nil
		})

		if err != nil {
			return actionCompleteMsg{err: err}
		}

		msg := "Services stopped"
		if deleteCluster {
			msg += " and cluster deleted"
		}

		return actionCompleteMsg{message: msg}
	}
}

func (m *Model) startService(serviceName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		var err error
		suppressOutput(func() error {
			err = m.orch.StartService(ctx, m.runtime, serviceName)
			return nil
		})

		if err != nil {
			return actionCompleteMsg{err: err}
		}

		return actionCompleteMsg{message: fmt.Sprintf("Service %s started successfully", serviceName)}
	}
}

func (m *Model) stopService(serviceName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		var err error
		suppressOutput(func() error {
			err = m.orch.StopService(ctx, m.runtime, serviceName)
			return nil
		})

		if err != nil {
			return actionCompleteMsg{err: err}
		}

		return actionCompleteMsg{message: fmt.Sprintf("Service %s stopped successfully", serviceName)}
	}
}

func (m *Model) restartService(serviceName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		var err error
		suppressOutput(func() error {
			err = m.orch.RestartService(ctx, m.runtime, serviceName)
			return nil
		})

		if err != nil {
			return actionCompleteMsg{err: err}
		}

		return actionCompleteMsg{message: fmt.Sprintf("Service %s restarted successfully", serviceName)}
	}
}
