// Package repo provides local repository access.
package repo

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ireland-samantha/stormstack-dev-bot/internal/config"
)

// LocalRepo provides access to an existing local repository.
type LocalRepo struct {
	path string
}

// NewLocalRepo creates a new local repository manager.
func NewLocalRepo(path string) (*LocalRepo, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	return &LocalRepo{path: absPath}, nil
}

// GetRepoPath returns the repository path.
func (r *LocalRepo) GetRepoPath() string {
	return r.path
}

// EnsureReady validates that the path exists and is a git repository.
func (r *LocalRepo) EnsureReady() error {
	// Check that the path exists
	info, err := os.Stat(r.path)
	if err != nil {
		return fmt.Errorf("repository path does not exist: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("repository path is not a directory: %s", r.path)
	}

	// Check that it's a git repository
	gitDir := filepath.Join(r.path, ".git")
	if _, err := os.Stat(gitDir); err != nil {
		return fmt.Errorf("not a git repository (missing .git): %s", r.path)
	}

	return nil
}

// Sync fetches the latest changes from the remote.
func (r *LocalRepo) Sync() error {
	cmd := exec.Command("git", "fetch", "--all")
	cmd.Dir = r.path
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git fetch failed: %w\n%s", err, string(output))
	}
	return nil
}

// GetMode returns the repository access mode.
func (r *LocalRepo) GetMode() config.Mode {
	return config.ModeLocal
}
