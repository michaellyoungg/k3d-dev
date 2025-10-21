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

func (m *Model) renderClusterStatus() string {
	if m.status == nil || m.status.Cluster == nil {
		return dimStyle.Render("No cluster information")
	}

	cluster := m.status.Cluster
	icon := getStatusIcon(cluster.Status)

	var parts []string
	parts = append(parts, fmt.Sprintf("%s Cluster: %s", icon, cluster.Status))

	if cluster.Name != "" {
		parts = append(parts, dimStyle.Render(fmt.Sprintf("(%s)", cluster.Name)))
	}

	if cluster.Servers > 0 {
		parts = append(parts, dimStyle.Render(fmt.Sprintf("%d servers, %d agents", cluster.Servers, cluster.Agents)))
	}

	if cluster.Error != "" {
		parts = append(parts, errorStyle.Render(cluster.Error))
	}

	return strings.Join(parts, " ")
}

func (m *Model) renderServices() string {
	if m.status == nil || len(m.status.Services) == 0 {
		return dimStyle.Render("No services configured")
	}

	var b strings.Builder
	b.WriteString(sectionStyle.Render("Services:"))
	b.WriteString("\n\n")

	// Get sorted service names for stable display order
	serviceNames := m.getSortedServiceNames()

	for i, name := range serviceNames {
		svc := m.status.Services[name]
		icon := getStatusIcon(svc.Status)

		// Highlight selected service
		var style lipgloss.Style
		if i == m.selectedService {
			style = lipgloss.NewStyle().Background(lipgloss.Color("236")).Padding(0, 1)
		} else {
			style = lipgloss.NewStyle().Padding(0, 1)
		}

		line := fmt.Sprintf("%s %-20s", icon, name)

		// Add version
		if svc.Version != "" {
			line += dimStyle.Render(fmt.Sprintf(" v%s", svc.Version))
		}

		// Add local indicator
		if svc.IsLocal {
			line += " " + activeStyle.Render("local")
		}

		// Add ports
		if len(svc.Ports) > 0 {
			line += dimStyle.Render(fmt.Sprintf(" :%d", svc.Ports[0]))
		}

		b.WriteString(style.Render(line))
		b.WriteString("\n")
	}

	return b.String()
}

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
