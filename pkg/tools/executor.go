package tools

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// DefaultProcessExecutor implements ProcessExecutor using Go's os/exec
type DefaultProcessExecutor struct{}

// NewProcessExecutor creates a new process executor
func NewProcessExecutor() ProcessExecutor {
	return &DefaultProcessExecutor{}
}

// Execute runs a command and captures all output
func (e *DefaultProcessExecutor) Execute(ctx context.Context, cmd Command) (*ExecuteResult, error) {
	execCmd := exec.CommandContext(ctx, cmd.Name, cmd.Args...)
	
	// Set working directory if specified
	if cmd.Dir != "" {
		execCmd.Dir = cmd.Dir
	}
	
	// Set environment variables
	if len(cmd.Env) > 0 {
		execCmd.Env = os.Environ()
		for key, value := range cmd.Env {
			execCmd.Env = append(execCmd.Env, fmt.Sprintf("%s=%s", key, value))
		}
	}
	
	var stdout, stderr bytes.Buffer
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr
	
	err := execCmd.Run()
	
	result := &ExecuteResult{
		ExitCode: 0,
		Stdout:   strings.TrimSpace(stdout.String()),
		Stderr:   strings.TrimSpace(stderr.String()),
	}
	
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
		} else {
			result.ExitCode = 1
		}
		// Include stderr in error message for better debugging
		if result.Stderr != "" {
			return result, fmt.Errorf("command failed: %w\nStderr: %s", err, result.Stderr)
		}
		return result, fmt.Errorf("command failed: %w", err)
	}
	
	return result, nil
}

// Stream runs a command with real-time output streaming
func (e *DefaultProcessExecutor) Stream(ctx context.Context, cmd Command, output io.Writer) error {
	execCmd := exec.CommandContext(ctx, cmd.Name, cmd.Args...)
	
	// Set working directory if specified
	if cmd.Dir != "" {
		execCmd.Dir = cmd.Dir
	}
	
	// Set environment variables
	if len(cmd.Env) > 0 {
		execCmd.Env = os.Environ()
		for key, value := range cmd.Env {
			execCmd.Env = append(execCmd.Env, fmt.Sprintf("%s=%s", key, value))
		}
	}
	
	// Stream output to provided writer
	execCmd.Stdout = output
	execCmd.Stderr = output
	
	err := execCmd.Run()
	if err != nil {
		return fmt.Errorf("streaming command failed: %w", err)
	}
	
	return nil
}

// ValidateCommand checks if a command is available in PATH
func ValidateCommand(name string) error {
	_, err := exec.LookPath(name)
	if err != nil {
		return fmt.Errorf("command '%s' not found in PATH: %w", name, err)
	}
	return nil
}

// GetCommandVersion attempts to get version information from a command
func GetCommandVersion(ctx context.Context, name string, versionArgs ...string) (string, error) {
	if len(versionArgs) == 0 {
		versionArgs = []string{"--version"}
	}
	
	cmd := Command{
		Name: name,
		Args: versionArgs,
	}
	
	executor := NewProcessExecutor()
	result, err := executor.Execute(ctx, cmd)
	if err != nil {
		return "", fmt.Errorf("failed to get version for %s: %w", name, err)
	}
	
	// Return first line of output, which usually contains version info
	lines := strings.Split(result.Stdout, "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0]), nil
	}
	
	return result.Stdout, nil
}