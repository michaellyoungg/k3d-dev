package ui

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/key"
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

	// Footer
	b.WriteString("\n\n")
	b.WriteString(m.renderFooter())

	return b.String()
}

func (m *Model) renderLogs() string {
	var b strings.Builder

	// Logs header with streaming indicator
	title := sectionStyle.Render(fmt.Sprintf("📋 Logs: %s", m.logService))
	if m.logStreaming {
		title += " " + successStyle.Render("● streaming")
	}
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

	b.WriteString(dimStyle.Render(fmt.Sprintf("Use ↑/↓ to scroll • t/p to toggle %s • l/ESC to go back", strings.Join(toggleInfo, " • "))))
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
	case key.Matches(msg, m.keys.Back), key.Matches(msg, m.keys.Logs):
		// Stop streaming and go back to home (ESC or L key to toggle)
		m.stopLogStream()
		m.view = HomeView
		m.logs = nil
		m.rawLogs = nil
		m.logsInitialized = false
		return m, nil

	case key.Matches(msg, m.keys.Up):
		if m.logsInitialized {
			// Mark that user has scrolled
			m.userScrolled = true
			m.viewport.ScrollUp(1)
		}
		return m, nil

	case key.Matches(msg, m.keys.Down):
		if m.logsInitialized {
			m.viewport.ScrollDown(1)
			// Check if we're at the bottom after scrolling down
			if m.viewport.AtBottom() {
				m.userScrolled = false
			}
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
		m.view = HomeView
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

	// Start streaming logs
	cmd, reader, err := m.startLogStream(msg.service)
	if err != nil {
		// If streaming fails, just show the initial logs
		m.error = err
		return m, nil
	}

	m.logStreamCmd = cmd
	m.logStreamReader = reader
	m.logBufioReader = bufio.NewReader(reader)
	m.logStreaming = true

	// Start waiting for the first log line
	return m, m.waitForLogLine()
}

func (m *Model) handleLogStreamMsg(msg logStreamMsg) (tea.Model, tea.Cmd) {
	// Append new log line to raw logs
	m.rawLogs = append(m.rawLogs, msg.line)

	// Update the display with the new line
	m.updateLogDisplay()

	// Auto-scroll to bottom if user hasn't scrolled up
	if !m.userScrolled {
		m.viewport.GotoBottom()
	}

	// Wait for the next line
	return m, m.waitForLogLine()
}

func (m *Model) handleLogStreamErrorMsg(msg logStreamErrorMsg) (tea.Model, tea.Cmd) {
	// Stream ended or error occurred
	m.stopLogStream()

	// Only show error if it's not EOF (normal end of stream)
	if msg.err != nil && msg.err != io.EOF {
		m.error = msg.err
	}

	return m, nil
}

// Logs commands

func (m *Model) fetchLogs(serviceName string) tea.Cmd {
	return func() tea.Msg {
		// Build kubectl command to get initial logs
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

// startLogStream initializes the kubectl log stream process
func (m *Model) startLogStream(serviceName string) (*exec.Cmd, io.ReadCloser, error) {
	namespace := m.runtime.Base.Defaults.Namespace
	selector := fmt.Sprintf("app.kubernetes.io/instance=%s", serviceName)

	cmd := exec.Command("kubectl", "logs",
		"-l", selector,
		"-n", namespace,
		"--follow",
		"--timestamps")

	// Get stdout pipe
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("failed to start log stream: %w", err)
	}

	return cmd, stdout, nil
}

// waitForLogLine reads a single line from the stream using the buffered reader
func (m *Model) waitForLogLine() tea.Cmd {
	return func() tea.Msg {
		if m.logBufioReader == nil {
			return logStreamErrorMsg{err: io.EOF}
		}

		// Read one line from the buffered reader
		line, err := m.logBufioReader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return logStreamErrorMsg{err: io.EOF}
			}
			return logStreamErrorMsg{err: err}
		}

		// Trim the newline character
		if len(line) > 0 && line[len(line)-1] == '\n' {
			line = line[:len(line)-1]
		}
		// Also trim carriage return if present (for Windows line endings)
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}

		return logStreamMsg{line: line}
	}
}

// stopLogStream stops the running log stream
func (m *Model) stopLogStream() {
	if m.logStreamCmd != nil && m.logStreamCmd.Process != nil {
		m.logStreamCmd.Process.Kill()
		m.logStreamCmd = nil
	}
	if m.logStreamReader != nil {
		m.logStreamReader.Close()
		m.logStreamReader = nil
	}
	m.logBufioReader = nil
	m.logStreaming = false
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
