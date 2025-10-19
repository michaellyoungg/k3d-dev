package ui

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"plat/pkg/config"
	"plat/pkg/orchestrator"
)

// ViewMode represents different views in the dashboard
type ViewMode int

const (
	ViewDashboard ViewMode = iota
	ViewLogs
	ViewHelp
)

// DashboardModel is the main TUI dashboard
type DashboardModel struct {
	// State
	runtime *config.RuntimeConfig
	orch    *orchestrator.Orchestrator
	ctx     context.Context
	status  *orchestrator.EnvironmentStatus

	// UI state
	view            ViewMode
	selectedService int
	loading         bool
	operation       string // Current operation being performed
	message         string
	error           error

	// Components
	spinner  spinner.Model
	help     help.Model
	keys     keyMap
	viewport viewport.Model

	// Log viewer state
	showingLogs     bool
	logService      string
	logs            []string
	logsInitialized bool

	// Dimensions
	width  int
	height int

	// Auto-refresh
	lastRefresh time.Time
}

type keyMap struct {
	Up         key.Binding
	Down       key.Binding
	Start      key.Binding
	Stop       key.Binding
	StopAll    key.Binding
	Refresh    key.Binding
	Logs       key.Binding
	Back       key.Binding
	Help       key.Binding
	Quit       key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Start, k.Stop, k.Logs, k.Refresh, k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down},
		{k.Start, k.Stop, k.StopAll},
		{k.Logs, k.Refresh},
		{k.Help, k.Quit},
	}
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

// Messages
type statusRefreshMsg struct {
	status *orchestrator.EnvironmentStatus
	err    error
}

type actionCompleteMsg struct {
	message string
	err     error
}

type logsMsg struct {
	service string
	logs    []string
	err     error
}

// NewDashboard creates a new dashboard TUI
func NewDashboard(runtime *config.RuntimeConfig) *DashboardModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return &DashboardModel{
		runtime:     runtime,
		orch:        orchestrator.NewOrchestrator(false),
		ctx:         context.Background(),
		view:        ViewDashboard,
		spinner:     s,
		help:        help.New(),
		keys:        keys,
		lastRefresh: time.Now(),
	}
}

func (m *DashboardModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.refreshStatus(),
		tickEvery(5 * time.Second),
	)
}

func (m *DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
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
			clearMessageAfter(3 * time.Second),
		)

	case tickMsg:
		// Auto-refresh every 5 seconds
		return m, tea.Batch(
			m.refreshStatus(),
			tickEvery(5 * time.Second),
		)

	case clearMsg:
		m.message = ""
		return m, nil

	case logsMsg:
		if msg.err != nil {
			m.error = msg.err
			m.showingLogs = false
			return m, nil
		}

		m.logs = msg.logs
		m.logService = msg.service

		// Initialize viewport if not done
		if !m.logsInitialized {
			m.viewport = viewport.New(m.width, m.height-10)
			m.viewport.Style = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("62"))
			m.logsInitialized = true
		}

		// Set content
		m.viewport.SetContent(strings.Join(m.logs, "\n"))
		m.viewport.GotoBottom()

		return m, nil
	}

	return m, nil
}

func (m *DashboardModel) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Help):
		m.help.ShowAll = !m.help.ShowAll
		return m, nil

	case key.Matches(msg, m.keys.Back):
		// Go back to dashboard from logs
		if m.showingLogs {
			m.showingLogs = false
			m.logs = nil
			m.logsInitialized = false
			return m, nil
		}
		return m, nil

	case key.Matches(msg, m.keys.Up):
		if m.showingLogs && m.logsInitialized {
			// Scroll logs up
			m.viewport.ScrollUp(1)
			return m, nil
		} else if m.status != nil && len(m.status.Services) > 0 {
			m.selectedService = max(0, m.selectedService-1)
		}
		return m, nil

	case key.Matches(msg, m.keys.Down):
		if m.showingLogs && m.logsInitialized {
			// Scroll logs down
			m.viewport.ScrollDown(1)
			return m, nil
		} else if m.status != nil && len(m.status.Services) > 0 {
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
		// Toggle log viewer for selected service
		if m.showingLogs {
			// Already showing logs, go back
			m.showingLogs = false
			m.logs = nil
			m.logsInitialized = false
			return m, nil
		}

		// Get selected service name
		if m.status != nil && len(m.status.Services) > 0 {
			i := 0
			for name := range m.status.Services {
				if i == m.selectedService {
					m.logService = name
					m.showingLogs = true
					return m, m.fetchLogs(name)
				}
				i++
			}
		}
		return m, nil
	}

	return m, nil
}

func (m *DashboardModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var b strings.Builder

	// Header
	b.WriteString(m.renderHeader())
	b.WriteString("\n\n")

	// Main content based on view
	if m.showingLogs {
		b.WriteString(m.renderLogs())
	} else {
		switch m.view {
		case ViewDashboard:
			b.WriteString(m.renderDashboard())
		case ViewHelp:
			b.WriteString(m.renderHelpView())
		}
	}

	// Footer with help
	b.WriteString("\n\n")
	b.WriteString(m.help.View(m.keys))

	return b.String()
}

func (m *DashboardModel) renderHeader() string {
	title := headerStyle.Render("🎯 Plat Dashboard")

	var status string
	if m.loading && m.operation != "" {
		// Show active operation with spinner
		status = activeStyle.Render(m.spinner.View() + " " + m.operation + "...")
	} else if m.message != "" {
		// Show success message
		status = successStyle.Render("✓ " + m.message)
	} else if m.error != nil {
		// Show error
		status = errorStyle.Render("✗ " + m.error.Error())
	} else if m.status != nil {
		// Show last refresh time
		elapsed := time.Since(m.lastRefresh).Round(time.Second)
		status = dimStyle.Render(fmt.Sprintf("Last updated: %s ago", elapsed))
	}

	// Pad to fill width
	padding := m.width - lipgloss.Width(title) - lipgloss.Width(status) - 4
	if padding < 0 {
		padding = 0
	}

	return title + strings.Repeat(" ", padding) + status
}

func (m *DashboardModel) renderDashboard() string {
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

func (m *DashboardModel) renderClusterStatus() string {
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

func (m *DashboardModel) renderServices() string {
	if m.status == nil || len(m.status.Services) == 0 {
		return dimStyle.Render("No services configured")
	}

	var b strings.Builder
	b.WriteString(sectionStyle.Render("Services:"))
	b.WriteString("\n\n")

	i := 0
	for name, svc := range m.status.Services {
		icon := getStatusIcon(svc.Status)

		// Highlight selected service
		style := lipgloss.NewStyle()
		if i == m.selectedService {
			style = style.Background(lipgloss.Color("236")).Padding(0, 1)
		} else {
			style = style.Padding(0, 1)
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
		i++
	}

	return b.String()
}

func (m *DashboardModel) renderLogs() string {
	var b strings.Builder

	// Logs header
	title := sectionStyle.Render(fmt.Sprintf("📋 Logs: %s", m.logService))
	b.WriteString(title)
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("Use ↑/↓ to scroll, ESC to go back"))
	b.WriteString("\n\n")

	// Show viewport if logs are loaded
	if m.logsInitialized && len(m.logs) > 0 {
		b.WriteString(m.viewport.View())
	} else if len(m.logs) == 0 {
		b.WriteString(dimStyle.Render("No logs available"))
	} else {
		b.WriteString(fmt.Sprintf("%s Loading logs...", m.spinner.View()))
	}

	return b.String()
}

func (m *DashboardModel) renderHelpView() string {
	return m.help.View(m.keys)
}

// Commands

func (m *DashboardModel) refreshStatus() tea.Cmd {
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

func (m *DashboardModel) startEnvironment() tea.Cmd {
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

func (m *DashboardModel) stopServices(deleteCluster bool) tea.Cmd {
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

func (m *DashboardModel) fetchLogs(serviceName string) tea.Cmd {
	return func() tea.Msg {
		// Build kubectl command to get logs
		namespace := m.runtime.Base.Defaults.Namespace
		selector := fmt.Sprintf("app.kubernetes.io/instance=%s", serviceName)

		cmd := exec.Command("kubectl", "logs",
			"-l", selector,
			"-n", namespace,
			"--tail=100",
			"--timestamps")

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err != nil {
			errorMsg := stderr.String()
			if errorMsg == "" {
				errorMsg = err.Error()
			}
			return logsMsg{
				service: serviceName,
				err:     fmt.Errorf("failed to get logs: %s", errorMsg),
			}
		}

		// Split logs into lines
		output := stdout.String()
		var logs []string
		scanner := bufio.NewScanner(strings.NewReader(output))
		for scanner.Scan() {
			logs = append(logs, scanner.Text())
		}

		if len(logs) == 0 {
			logs = []string{"No logs available for this service"}
		}

		return logsMsg{
			service: serviceName,
			logs:    logs,
		}
	}
}

// Helper messages and functions

type tickMsg time.Time
type clearMsg struct{}

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

// suppressOutput redirects stdout/stderr to null during execution
func suppressOutput(fn func() error) error {
	// Save original stdout/stderr
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	// Redirect to null (open for writing)
	null, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0666)
	if err != nil {
		return fn() // If we can't open null, just run normally
	}
	defer null.Close()

	os.Stdout = null
	os.Stderr = null

	// Restore after execution
	defer func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	}()

	return fn()
}

// Styles
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

// RunDashboard starts the interactive dashboard
func RunDashboard(runtime *config.RuntimeConfig) error {
	m := NewDashboard(runtime)
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}
