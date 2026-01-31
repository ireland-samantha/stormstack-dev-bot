// Package executor provides command execution with sandboxing.
package executor

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const (
	// DefaultTimeout is the default command timeout.
	DefaultTimeout = 5 * time.Minute
	// MaxOutputSize is the maximum output size in bytes.
	MaxOutputSize = 100 * 1024 // 100KB
)

// Runner executes commands in the repository directory.
type Runner struct {
	repoPath string
	buildCmd string
	testCmd  string
}

// NewRunner creates a new command runner.
func NewRunner(repoPath, buildCmd, testCmd string) *Runner {
	return &Runner{
		repoPath: repoPath,
		buildCmd: buildCmd,
		testCmd:  testCmd,
	}
}

// CommandResult represents the result of a command execution.
type CommandResult struct {
	Command  string
	ExitCode int
	Stdout   string
	Stderr   string
	Duration time.Duration
	TimedOut bool
}

// RunCommand runs a command with safety checks.
func (r *Runner) RunCommand(ctx context.Context, command string) (*CommandResult, error) {
	// Validate command
	if err := ValidateCommand(command); err != nil {
		return nil, err
	}

	return r.executeCommand(ctx, command, DefaultTimeout)
}

// RunBuild runs the configured build command.
func (r *Runner) RunBuild(ctx context.Context, args string) (*CommandResult, error) {
	command := r.buildCmd
	if args != "" {
		command = command + " " + args
	}
	return r.executeCommand(ctx, command, DefaultTimeout)
}

// RunTests runs the configured test command.
func (r *Runner) RunTests(ctx context.Context, args string) (*CommandResult, error) {
	command := r.testCmd
	if args != "" {
		command = command + " " + args
	}
	return r.executeCommand(ctx, command, DefaultTimeout)
}

// executeCommand executes a shell command.
func (r *Runner) executeCommand(ctx context.Context, command string, timeout time.Duration) (*CommandResult, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Create command
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = r.repoPath

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &limitedWriter{w: &stdout, limit: MaxOutputSize}
	cmd.Stderr = &limitedWriter{w: &stderr, limit: MaxOutputSize}

	// Run command
	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	// Build result
	result := &CommandResult{
		Command:  command,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: duration,
		TimedOut: ctx.Err() == context.DeadlineExceeded,
	}

	// Get exit code
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else if result.TimedOut {
			result.ExitCode = -1
		} else {
			return nil, fmt.Errorf("command failed: %w", err)
		}
	}

	return result, nil
}

// FormatResult formats a command result for display.
func (r *CommandResult) FormatResult() string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("$ %s\n", r.Command))

	if r.Stdout != "" {
		builder.WriteString(r.Stdout)
		if !strings.HasSuffix(r.Stdout, "\n") {
			builder.WriteString("\n")
		}
	}

	if r.Stderr != "" {
		builder.WriteString("STDERR:\n")
		builder.WriteString(r.Stderr)
		if !strings.HasSuffix(r.Stderr, "\n") {
			builder.WriteString("\n")
		}
	}

	if r.TimedOut {
		builder.WriteString("\n[Command timed out]\n")
	} else if r.ExitCode != 0 {
		builder.WriteString(fmt.Sprintf("\n[Exit code: %d]\n", r.ExitCode))
	}

	builder.WriteString(fmt.Sprintf("[Duration: %s]\n", r.Duration.Round(time.Millisecond)))

	return builder.String()
}

// IsSuccess returns true if the command succeeded.
func (r *CommandResult) IsSuccess() bool {
	return r.ExitCode == 0 && !r.TimedOut
}

// CombinedOutput returns stdout and stderr combined.
func (r *CommandResult) CombinedOutput() string {
	if r.Stderr == "" {
		return r.Stdout
	}
	return r.Stdout + "\n" + r.Stderr
}

// limitedWriter limits the amount of data written.
type limitedWriter struct {
	w       *bytes.Buffer
	limit   int
	written int
}

func (lw *limitedWriter) Write(p []byte) (n int, err error) {
	remaining := lw.limit - lw.written
	if remaining <= 0 {
		return len(p), nil // Discard but don't error
	}

	if len(p) > remaining {
		p = p[:remaining]
	}

	n, err = lw.w.Write(p)
	lw.written += n
	return len(p), err // Report full length to avoid writer errors
}
