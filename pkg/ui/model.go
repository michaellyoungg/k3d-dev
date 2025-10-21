package ui

import (
	"bufio"
	"io"
	"os/exec"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"plat/pkg/config"
	"plat/pkg/orchestrator"
)

type Model struct {
	// Shared state
	runtime     *config.RuntimeConfig
	orch        *orchestrator.Orchestrator
	status      *orchestrator.EnvironmentStatus
	lastRefresh time.Time

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

	// Log viewer state
	logService       string
	logs             []string
	rawLogs          []string // Original logs before filtering
	logsInitialized  bool
	showTimestamps   bool
	showPodNames     bool
	logStreaming     bool          // Whether logs are actively streaming
	userScrolled     bool          // Whether user has scrolled away from bottom
	unseenLogCount   int           // Number of new logs arrived while user is scrolled up
	logStreamCmd     *exec.Cmd     // The running kubectl logs command
	logStreamReader  io.ReadCloser // The stdout reader for the stream
	logBufioReader   *bufio.Reader // Buffered reader for efficient line reading

	// Dimensions
	width  int
	height int
}

func RunTUI(runtime *config.RuntimeConfig) error {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	m := &Model{
		runtime:        runtime,
		orch:           orchestrator.NewOrchestrator(false),
		view:           HomeView,
		spinner:        s,
		help:           help.New(),
		keys:           keys,
		lastRefresh:    time.Now(),
		showTimestamps: false, // Hide timestamps by default to save space
		showPodNames:   false, // Hide pod names by default to save space
	}

	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}
