// Package repo provides sandbox repository access with clone capability.
package repo

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ireland-samantha/stormstack-dev-bot/internal/config"
)

// SandboxRepo provides access to a cloned repository in a sandboxed workspace.
type SandboxRepo struct {
	githubRepo    string
	githubToken   string
	workspacePath string
	repoPath      string
}

// NewSandboxRepo creates a new sandbox repository manager.
func NewSandboxRepo(githubRepo, githubToken, workspacePath string) (*SandboxRepo, error) {
	absWorkspace, err := filepath.Abs(workspacePath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve workspace path: %w", err)
	}

	// Extract repo name from github repo URL
	repoName := extractRepoName(githubRepo)
	repoPath := filepath.Join(absWorkspace, repoName)

	return &SandboxRepo{
		githubRepo:    githubRepo,
		githubToken:   githubToken,
		workspacePath: absWorkspace,
		repoPath:      repoPath,
	}, nil
}

// GetRepoPath returns the repository path.
func (r *SandboxRepo) GetRepoPath() string {
	return r.repoPath
}

// EnsureReady clones the repository if it doesn't exist.
func (r *SandboxRepo) EnsureReady() error {
	// Create workspace directory if needed
	if err := os.MkdirAll(r.workspacePath, 0755); err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	// Check if repo already exists
	gitDir := filepath.Join(r.repoPath, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		// Repository exists, just fetch latest
		return r.Sync()
	}

	// Clone the repository
	cloneURL := r.buildCloneURL()
	cmd := exec.Command("git", "clone", cloneURL, r.repoPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone failed: %w\n%s", err, string(output))
	}

	return nil
}

// Sync fetches the latest changes and resets to origin/main.
func (r *SandboxRepo) Sync() error {
	// Fetch all remotes
	fetchCmd := exec.Command("git", "fetch", "--all")
	fetchCmd.Dir = r.repoPath
	if output, err := fetchCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git fetch failed: %w\n%s", err, string(output))
	}

	// Get the default branch
	defaultBranch, err := r.getDefaultBranch()
	if err != nil {
		return err
	}

	// Reset to origin/default branch
	resetCmd := exec.Command("git", "checkout", defaultBranch)
	resetCmd.Dir = r.repoPath
	if output, err := resetCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git checkout failed: %w\n%s", err, string(output))
	}

	pullCmd := exec.Command("git", "pull", "origin", defaultBranch)
	pullCmd.Dir = r.repoPath
	if output, err := pullCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git pull failed: %w\n%s", err, string(output))
	}

	return nil
}

// GetMode returns the repository access mode.
func (r *SandboxRepo) GetMode() config.Mode {
	return config.ModeSandbox
}

// buildCloneURL constructs the authenticated clone URL.
func (r *SandboxRepo) buildCloneURL() string {
	// Remove protocol prefix if present
	repo := r.githubRepo
	repo = strings.TrimPrefix(repo, "https://")
	repo = strings.TrimPrefix(repo, "http://")
	repo = strings.TrimPrefix(repo, "git@")

	// Build authenticated HTTPS URL
	return fmt.Sprintf("https://%s@%s", r.githubToken, repo)
}

// getDefaultBranch determines the default branch (main or master).
func (r *SandboxRepo) getDefaultBranch() (string, error) {
	// Try to get the default branch from remote HEAD
	cmd := exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD", "--short")
	cmd.Dir = r.repoPath
	output, err := cmd.Output()
	if err == nil {
		branch := strings.TrimSpace(string(output))
		// Remove "origin/" prefix
		branch = strings.TrimPrefix(branch, "origin/")
		return branch, nil
	}

	// Fallback: check if main exists
	mainCmd := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/remotes/origin/main")
	mainCmd.Dir = r.repoPath
	if mainCmd.Run() == nil {
		return "main", nil
	}

	// Fallback: check if master exists
	masterCmd := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/remotes/origin/master")
	masterCmd.Dir = r.repoPath
	if masterCmd.Run() == nil {
		return "master", nil
	}

	return "main", nil // Default to main
}

// extractRepoName extracts the repository name from a GitHub URL.
func extractRepoName(url string) string {
	// Remove protocol and domain
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "git@")
	url = strings.TrimPrefix(url, "github.com/")
	url = strings.TrimPrefix(url, "github.com:")
	url = strings.TrimSuffix(url, ".git")

	// Get the last part (repo name)
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return url
}
