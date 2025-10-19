package ui

import (
	"context"
	"fmt"
	"os"
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

// ViewMode represents different views in the TUI
type ViewMode int

const (
	ViewDashboard ViewMode = iota
	ViewLogs
	ViewHelp
)

// Model is the root TUI model containing shared state
type Model struct {
	// Shared state
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

	// Shared components
	spinner spinner.Model
	help    help.Model
	keys    keyMap

	// View-specific components
	viewport viewport.Model

	// Dashboard state
	lastRefresh time.Time

	// Log viewer state
	logService      string
	logs            []string
	rawLogs         []string // Original logs before filtering
	logsInitialized bool
	showTimestamps  bool
	showPodNames    bool

	// Dimensions
	width  int
	height int
}

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
	case ViewLogs:
		return []key.Binding{m.keys.Up, m.keys.Down, m.keys.ToggleTimestamp, m.keys.TogglePodName, m.keys.Logs, m.keys.Back, m.keys.Quit}
	default: // ViewDashboard
		return []key.Binding{m.keys.Start, m.keys.Stop, m.keys.Logs, m.keys.Refresh, m.keys.Help, m.keys.Quit}
	}
}

// FullHelp returns context-aware full help based on current view
func (m *Model) FullHelp() [][]key.Binding {
	switch m.view {
	case ViewLogs:
		return [][]key.Binding{
			{m.keys.Up, m.keys.Down},
			{m.keys.ToggleTimestamp, m.keys.TogglePodName},
			{m.keys.Logs, m.keys.Back, m.keys.Help, m.keys.Quit},
		}
	default: // ViewDashboard
		return [][]key.Binding{
			{m.keys.Up, m.keys.Down},
			{m.keys.Start, m.keys.Stop, m.keys.StopAll},
			{m.keys.Logs, m.keys.Refresh},
			{m.keys.Help, m.keys.Quit},
		}
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

type tickMsg time.Time
type clearMsg struct{}

// New creates a new TUI model
func New(runtime *config.RuntimeConfig) *Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return &Model{
		runtime:        runtime,
		orch:           orchestrator.NewOrchestrator(false),
		ctx:            context.Background(),
		view:           ViewDashboard,
		spinner:        s,
		help:           help.New(),
		keys:           keys,
		lastRefresh:    time.Now(),
		showTimestamps: false, // Hide timestamps by default to save space
		showPodNames:   false, // Hide pod names by default to save space
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.refreshStatus(),
		tickEvery(5*time.Second),
	)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
	}

	return m, nil
}

func (m *Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Delegate to view-specific rendering
	switch m.view {
	case ViewLogs:
		return m.renderLogsView()
	case ViewHelp:
		return m.renderHelpView()
	case ViewDashboard:
		return m.renderDashboardView()
	default:
		return m.renderDashboardView()
	}
}

func (m *Model) renderHelpView() string {
	return m.help.View(m)
}

func (m *Model) renderHeader() string {
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

func (m *Model) renderFooter() string {
	return m.help.View(m)
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

// RunDashboard starts the interactive dashboard
func RunDashboard(runtime *config.RuntimeConfig) error {
	m := New(runtime)
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}
