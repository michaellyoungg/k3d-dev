package ui

import (
	"time"

	"plat/pkg/orchestrator"
)

// Messages define all the messages that can be sent to the Update function

// statusRefreshMsg is sent when status data is refreshed
type statusRefreshMsg struct {
	status *orchestrator.EnvironmentStatus
	err    error
}

// actionCompleteMsg is sent when an action (up/down) completes
type actionCompleteMsg struct {
	message string
	err     error
}

// logsMsg is sent when logs are fetched for a service
type logsMsg struct {
	service string
	logs    []string
	err     error
}

// tickMsg is sent periodically for auto-refresh
type tickMsg time.Time

// clearMsg is sent to clear temporary messages
type clearMsg struct{}
