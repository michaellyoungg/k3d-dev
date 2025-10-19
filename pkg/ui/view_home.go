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

	// Show operation progress if running
	if m.loading && m.operation != "" {
		b.WriteString(sectionStyle.Render("⚙️  Operation in Progress"))
		b.WriteString("\n\n")
		b.WriteString(activeStyle.Render(fmt.Sprintf("  %s %s...", m.spinner.View(), m.operation)))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("  This may take a few moments"))
		b.WriteString("\n\n")
	}

	if m.status == nil {
		b.WriteString(dimStyle.Render("Loading status..."))
		return b.String()
	}

	// Environment info
	b.WriteString(sectionStyle.Render(fmt.Sprintf("Environment: %s (%s mode)", m.status.Name, m.status.Mode)))
	b.WriteString("\n\n")

	// Cluster status
	b.WriteString(m.renderClusterStatus())
	b.WriteString("\n\n")

	// Services
	b.WriteString(m.renderServices())

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
		if m.status != nil && len(m.status.Services) > 0 {
			m.selectedService = max(0, m.selectedService-1)
		}
		return m, nil

	case key.Matches(msg, m.keys.Down):
		if m.status != nil && len(m.status.Services) > 0 {
			m.selectedService = min(len(m.status.Services)-1, m.selectedService+1)
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
		// Get selected service name using sorted list
		if m.status != nil && len(m.status.Services) > 0 {
			serviceNames := m.getSortedServiceNames()
			if m.selectedService < len(serviceNames) {
				name := serviceNames[m.selectedService]
				m.logService = name
				m.view = ServiceLogsView
				return m, m.fetchLogs(name)
			}
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
