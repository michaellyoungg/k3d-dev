package ui

import (
	"bufio"
	"io"
	"os/exec"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"plat/pkg/config"
	"plat/pkg/orchestrator"
)

// NavItem represents an item in the left navigation panel
type NavItem struct {
	Type        NavItemType
	Name        string
	ServiceName string // Only populated for service items
}

// NavItemType identifies the type of navigation item
type NavItemType int

const (
	NavItemCluster NavItemType = iota
	NavItemService
)

type Model struct {
	// Shared state
	runtime *config.RuntimeConfig
	orch    *orchestrator.Orchestrator
	status  *orchestrator.EnvironmentStatus

	// UI state
	view        ViewMode
	selectedNav int // Index in navItems slice
	navItems    []NavItem
	loading     bool
	operation   string // Current operation being performed
	message     string
	error       error

	// Shared components
	spinner spinner.Model
	help    help.Model
	keys    keyMap

	// View-specific components
	viewport viewport.Model

	// Log viewer state
	logService      string
	logs            []string
	rawLogs         []string // Original logs before filtering
	logsInitialized bool
	showTimestamps  bool
	showPodNames    bool
	logStreaming    bool          // Whether logs are actively streaming
	userScrolled    bool          // Whether user has scrolled away from bottom
	unseenLogCount  int           // Number of new logs arrived while user is scrolled up
	logStreamCmd    *exec.Cmd     // The running kubectl logs command
	logStreamReader io.ReadCloser // The stdout reader for the stream
	logBufioReader  *bufio.Reader // Buffered reader for efficient line reading

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
		showTimestamps: false, // Hide timestamps by default to save space
		showPodNames:   false, // Hide pod names by default to save space
	}

	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}
