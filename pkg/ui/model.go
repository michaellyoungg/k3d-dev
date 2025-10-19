package ui

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"plat/pkg/config"
	"plat/pkg/orchestrator"
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

// New creates a new TUI model
func New(runtime *config.RuntimeConfig) *Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return &Model{
		runtime:        runtime,
		orch:           orchestrator.NewOrchestrator(false),
		ctx:            context.Background(),
		view:           HomeView,
		spinner:        s,
		help:           help.New(),
		keys:           keys,
		lastRefresh:    time.Now(),
		showTimestamps: false, // Hide timestamps by default to save space
		showPodNames:   false, // Hide pod names by default to save space
	}
}

func RunTUI(runtime *config.RuntimeConfig) error {
	m := New(runtime)
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}
