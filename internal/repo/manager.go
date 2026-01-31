// Package repo provides repository access abstraction for local and sandbox modes.
package repo

import (
	"fmt"

	"github.com/ireland-samantha/stormstack-dev-bot/internal/config"
)

// Manager provides access to a git repository.
type Manager interface {
	// GetRepoPath returns the absolute path to the repository root.
	GetRepoPath() string

	// EnsureReady prepares the repository for use.
	// For local mode, this validates the path.
	// For sandbox mode, this clones the repository.
	EnsureReady() error

	// Sync fetches the latest changes from the remote.
	Sync() error

	// GetMode returns the repository access mode.
	GetMode() config.Mode
}

// NewManager creates a repository manager based on configuration.
func NewManager(cfg *config.Config) (Manager, error) {
	switch cfg.Mode {
	case config.ModeLocal:
		return NewLocalRepo(cfg.RepoPath)
	case config.ModeSandbox:
		return NewSandboxRepo(cfg.GitHubRepo, cfg.GitHubToken, cfg.WorkspacePath)
	default:
		return nil, fmt.Errorf("unknown mode: %s", cfg.Mode)
	}
}
