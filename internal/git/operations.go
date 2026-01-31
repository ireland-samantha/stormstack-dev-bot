// Package git provides git operations.
package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/ireland-samantha/stormstack-dev-bot/internal/executor"
)

const (
	// CommandTimeout is the timeout for git commands.
	CommandTimeout = 2 * time.Minute
)

// Operations provides git operations for a repository.
type Operations struct {
	repoPath string
}

// NewOperations creates a new git operations instance.
func NewOperations(repoPath string) *Operations {
	return &Operations{repoPath: repoPath}
}

// Status returns the current git status.
func (g *Operations) Status(ctx context.Context) (string, error) {
	return g.runGit(ctx, "status", "--short", "--branch")
}

// Diff returns git diff output.
func (g *Operations) Diff(ctx context.Context, staged bool, ref, path string) (string, error) {
	args := []string{"diff"}

	if staged {
		args = append(args, "--cached")
	}

	if ref != "" {
		args = append(args, ref)
	}

	if path != "" {
		args = append(args, "--", path)
	}

	return g.runGit(ctx, args...)
}

// Log returns git log output.
func (g *Operations) Log(ctx context.Context, count int, path, format string) (string, error) {
	if count <= 0 {
		count = 10
	}

	args := []string{"log", fmt.Sprintf("-n%d", count)}

	switch format {
	case "oneline":
		args = append(args, "--oneline")
	case "short":
		args = append(args, "--format=short")
	case "medium":
		args = append(args, "--format=medium")
	case "full":
		args = append(args, "--format=full")
	default:
		args = append(args, "--oneline")
	}

	if path != "" {
		args = append(args, "--", path)
	}

	return g.runGit(ctx, args...)
}

// CreateBranch creates a new branch and switches to it.
func (g *Operations) CreateBranch(ctx context.Context, name, from string) error {
	// Sanitize branch name
	name = executor.SanitizeBranchName(name)
	if name == "" {
		return fmt.Errorf("invalid branch name")
	}

	args := []string{"checkout", "-b", name}
	if from != "" {
		args = append(args, from)
	}

	_, err := g.runGit(ctx, args...)
	return err
}

// Commit stages files and creates a commit.
func (g *Operations) Commit(ctx context.Context, message string, files []string) error {
	// Sanitize commit message
	message = executor.SanitizeCommitMessage(message)
	if message == "" {
		return fmt.Errorf("empty commit message")
	}

	// Stage files
	if len(files) == 0 {
		// Stage all modified files
		if _, err := g.runGit(ctx, "add", "-A"); err != nil {
			return fmt.Errorf("failed to stage files: %w", err)
		}
	} else {
		// Stage specific files
		args := append([]string{"add"}, files...)
		if _, err := g.runGit(ctx, args...); err != nil {
			return fmt.Errorf("failed to stage files: %w", err)
		}
	}

	// Create commit
	// Add co-author attribution
	message = message + "\n\nCo-Authored-By: StormStack Dev Bot <bot@stormstack.dev>"

	if _, err := g.runGit(ctx, "commit", "-m", message); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	return nil
}

// Push pushes the current branch to the remote.
func (g *Operations) Push(ctx context.Context, setUpstream bool) error {
	args := []string{"push"}

	if setUpstream {
		branch, err := g.CurrentBranch(ctx)
		if err != nil {
			return err
		}
		args = append(args, "-u", "origin", branch)
	}

	_, err := g.runGit(ctx, args...)
	return err
}

// CurrentBranch returns the current branch name.
func (g *Operations) CurrentBranch(ctx context.Context) (string, error) {
	output, err := g.runGit(ctx, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

// GetRemoteURL returns the remote URL.
func (g *Operations) GetRemoteURL(ctx context.Context) (string, error) {
	output, err := g.runGit(ctx, "remote", "get-url", "origin")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

// HasUncommittedChanges checks if there are uncommitted changes.
func (g *Operations) HasUncommittedChanges(ctx context.Context) (bool, error) {
	output, err := g.runGit(ctx, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(output) != "", nil
}

// GetDefaultBranch returns the default branch (main or master).
func (g *Operations) GetDefaultBranch(ctx context.Context) (string, error) {
	// Try to get from remote HEAD
	output, err := g.runGit(ctx, "symbolic-ref", "refs/remotes/origin/HEAD", "--short")
	if err == nil {
		branch := strings.TrimSpace(output)
		branch = strings.TrimPrefix(branch, "origin/")
		return branch, nil
	}

	// Check if main exists
	if _, err := g.runGit(ctx, "show-ref", "--verify", "--quiet", "refs/remotes/origin/main"); err == nil {
		return "main", nil
	}

	// Check if master exists
	if _, err := g.runGit(ctx, "show-ref", "--verify", "--quiet", "refs/remotes/origin/master"); err == nil {
		return "master", nil
	}

	return "main", nil
}

// Fetch fetches from all remotes.
func (g *Operations) Fetch(ctx context.Context) error {
	_, err := g.runGit(ctx, "fetch", "--all")
	return err
}

// Stash stashes current changes.
func (g *Operations) Stash(ctx context.Context, message string) error {
	args := []string{"stash", "push"}
	if message != "" {
		args = append(args, "-m", message)
	}
	_, err := g.runGit(ctx, args...)
	return err
}

// StashPop pops the latest stash.
func (g *Operations) StashPop(ctx context.Context) error {
	_, err := g.runGit(ctx, "stash", "pop")
	return err
}

// runGit executes a git command.
func (g *Operations) runGit(ctx context.Context, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, CommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = g.repoPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("git command timed out")
		}
		return "", fmt.Errorf("git %s failed: %s", strings.Join(args, " "), stderr.String())
	}

	return stdout.String(), nil
}
