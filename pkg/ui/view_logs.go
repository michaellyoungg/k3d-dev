package ui

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
)

// Logs view rendering and logic

func (m *Model) renderLogsView() string {
	var b strings.Builder

	// Header
	b.WriteString(m.renderHeader())
	b.WriteString("\n\n")

	// Logs content
	b.WriteString(m.renderLogs())

	// Footer with help
	b.WriteString("\n\n")
	b.WriteString(m.help.View(m.keys))

	return b.String()
}

func (m *Model) renderLogs() string {
	var b strings.Builder

	// Logs header
	title := sectionStyle.Render(fmt.Sprintf("📋 Logs: %s", m.logService))
	b.WriteString(title)
	b.WriteString("\n")

	// Show current toggle states and help
	var toggleInfo []string
	if m.showTimestamps {
		toggleInfo = append(toggleInfo, "timestamps: on")
	} else {
		toggleInfo = append(toggleInfo, "timestamps: off")
	}
	if m.showPodNames {
		toggleInfo = append(toggleInfo, "pod names: on")
	} else {
		toggleInfo = append(toggleInfo, "pod names: off")
	}

	b.WriteString(dimStyle.Render(fmt.Sprintf("Use ↑/↓ to scroll • t/p to toggle %s • ESC to go back", strings.Join(toggleInfo, " • "))))
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

// Logs-specific key handling

func (m *Model) handleLogsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back):
		// Go back to dashboard
		m.showingLogs = false
		m.logs = nil
		m.rawLogs = nil
		m.logsInitialized = false
		return m, nil

	case key.Matches(msg, m.keys.Up):
		if m.logsInitialized {
			m.viewport.ScrollUp(1)
		}
		return m, nil

	case key.Matches(msg, m.keys.Down):
		if m.logsInitialized {
			m.viewport.ScrollDown(1)
		}
		return m, nil

	case key.Matches(msg, m.keys.ToggleTimestamp):
		m.showTimestamps = !m.showTimestamps
		m.updateLogDisplay()
		return m, nil

	case key.Matches(msg, m.keys.TogglePodName):
		m.showPodNames = !m.showPodNames
		m.updateLogDisplay()
		return m, nil
	}

	return m, nil
}

// Logs message handling

func (m *Model) handleLogsMsg(msg logsMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.error = msg.err
		m.showingLogs = false
		return m, nil
	}

	m.rawLogs = msg.logs // Store original logs
	m.logService = msg.service

	// Initialize viewport if not done
	if !m.logsInitialized {
		m.viewport = m.createViewport(m.width, m.height-10)
		m.logsInitialized = true
	}

	// Apply filtering based on current toggle states
	m.updateLogDisplay()
	m.viewport.GotoBottom()

	return m, nil
}

func (m *Model) createViewport(width, height int) viewport.Model {
	vp := viewport.New(width, height)
	vp.Style = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62"))
	return vp
}

// Logs commands

func (m *Model) fetchLogs(serviceName string) tea.Cmd {
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

// updateLogDisplay reprocesses raw logs based on toggle states
func (m *Model) updateLogDisplay() {
	if !m.logsInitialized || len(m.rawLogs) == 0 {
		return
	}

	// Process rawLogs based on showTimestamps and showPodNames
	filtered := make([]string, 0, len(m.rawLogs))
	for _, line := range m.rawLogs {
		processed := line

		// Strip timestamp if disabled (kubectl --timestamps format: "2025-10-19T18:31:10.831Z message")
		if !m.showTimestamps {
			// Find first space after timestamp (timestamps are ISO8601 format)
			if len(processed) > 20 && processed[10] == 'T' {
				// Look for space after timestamp
				if idx := strings.Index(processed, " "); idx != -1 {
					processed = processed[idx+1:]
				}
			}
		}

		// Strip pod name if disabled (kubectl multi-pod format: "[pod-name] message" or "pod-name message")
		if !m.showPodNames {
			// Check for bracket format first
			if strings.HasPrefix(processed, "[") {
				if idx := strings.Index(processed, "] "); idx != -1 {
					processed = processed[idx+2:]
				}
			} else {
				// Some logs may have "pod-name " prefix without brackets
				// Only strip if it looks like a pod name (contains alphanumeric and dashes)
				parts := strings.SplitN(processed, " ", 2)
				if len(parts) == 2 {
					// Check if first part looks like a pod name (contains dash and alphanumeric)
					if strings.Contains(parts[0], "-") && len(parts[0]) > 5 {
						processed = parts[1]
					}
				}
			}
		}

		filtered = append(filtered, processed)
	}

	m.logs = filtered
	m.viewport.SetContent(strings.Join(m.logs, "\n"))
}
