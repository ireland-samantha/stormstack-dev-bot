// Package git provides GitHub operations via gh CLI.
package git

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// GitHub provides GitHub operations using the gh CLI.
type GitHub struct {
	repoPath string
	token    string
}

// NewGitHub creates a new GitHub operations instance.
func NewGitHub(repoPath, token string) *GitHub {
	return &GitHub{
		repoPath: repoPath,
		token:    token,
	}
}

// PRInfo contains information about a pull request.
type PRInfo struct {
	Number    int    `json:"number"`
	Title     string `json:"title"`
	URL       string `json:"url"`
	State     string `json:"state"`
	HeadRef   string `json:"headRefName"`
	BaseRef   string `json:"baseRefName"`
	Body      string `json:"body"`
	CreatedAt string `json:"createdAt"`
	Author    string `json:"author"`
}

// CreatePR creates a new pull request.
func (g *GitHub) CreatePR(ctx context.Context, title, body, base string, draft bool) (*PRInfo, error) {
	args := []string{"pr", "create", "--title", title, "--body", body}

	if base != "" {
		args = append(args, "--base", base)
	}

	if draft {
		args = append(args, "--draft")
	}

	output, err := g.runGH(ctx, args...)
	if err != nil {
		return nil, err
	}

	// gh pr create returns the PR URL
	url := strings.TrimSpace(output)

	// Get PR details
	return g.GetPRByURL(ctx, url)
}

// GetPR gets information about a pull request.
func (g *GitHub) GetPR(ctx context.Context, number int) (*PRInfo, error) {
	output, err := g.runGH(ctx, "pr", "view", fmt.Sprintf("%d", number), "--json",
		"number,title,url,state,headRefName,baseRefName,body,createdAt,author")
	if err != nil {
		return nil, err
	}

	var pr PRInfo
	if err := json.Unmarshal([]byte(output), &pr); err != nil {
		return nil, fmt.Errorf("failed to parse PR info: %w", err)
	}

	return &pr, nil
}

// GetPRByURL gets information about a PR by its URL.
func (g *GitHub) GetPRByURL(ctx context.Context, url string) (*PRInfo, error) {
	// Extract PR number from URL
	parts := strings.Split(url, "/")
	if len(parts) == 0 {
		return nil, fmt.Errorf("invalid PR URL: %s", url)
	}

	number := parts[len(parts)-1]

	output, err := g.runGH(ctx, "pr", "view", number, "--json",
		"number,title,url,state,headRefName,baseRefName,body,createdAt")
	if err != nil {
		// If we can't get details, return basic info
		return &PRInfo{URL: url, Title: "PR Created"}, nil
	}

	var pr PRInfo
	if err := json.Unmarshal([]byte(output), &pr); err != nil {
		return &PRInfo{URL: url}, nil
	}

	return &pr, nil
}

// ListPRs lists open pull requests.
func (g *GitHub) ListPRs(ctx context.Context, state string, limit int) ([]PRInfo, error) {
	if limit <= 0 {
		limit = 10
	}
	if state == "" {
		state = "open"
	}

	output, err := g.runGH(ctx, "pr", "list", "--state", state, "--limit", fmt.Sprintf("%d", limit),
		"--json", "number,title,url,state,headRefName,baseRefName,createdAt")
	if err != nil {
		return nil, err
	}

	var prs []PRInfo
	if err := json.Unmarshal([]byte(output), &prs); err != nil {
		return nil, fmt.Errorf("failed to parse PR list: %w", err)
	}

	return prs, nil
}

// IssueInfo contains information about an issue.
type IssueInfo struct {
	Number    int      `json:"number"`
	Title     string   `json:"title"`
	URL       string   `json:"url"`
	State     string   `json:"state"`
	Body      string   `json:"body"`
	Labels    []string `json:"labels"`
	CreatedAt string   `json:"createdAt"`
}

// GetIssue gets information about an issue.
func (g *GitHub) GetIssue(ctx context.Context, number int) (*IssueInfo, error) {
	output, err := g.runGH(ctx, "issue", "view", fmt.Sprintf("%d", number), "--json",
		"number,title,url,state,body,labels,createdAt")
	if err != nil {
		return nil, err
	}

	var issue IssueInfo
	if err := json.Unmarshal([]byte(output), &issue); err != nil {
		return nil, fmt.Errorf("failed to parse issue info: %w", err)
	}

	return &issue, nil
}

// ListIssues lists open issues.
func (g *GitHub) ListIssues(ctx context.Context, state string, limit int) ([]IssueInfo, error) {
	if limit <= 0 {
		limit = 10
	}
	if state == "" {
		state = "open"
	}

	output, err := g.runGH(ctx, "issue", "list", "--state", state, "--limit", fmt.Sprintf("%d", limit),
		"--json", "number,title,url,state,createdAt")
	if err != nil {
		return nil, err
	}

	var issues []IssueInfo
	if err := json.Unmarshal([]byte(output), &issues); err != nil {
		return nil, fmt.Errorf("failed to parse issue list: %w", err)
	}

	return issues, nil
}

// CheckGHInstalled verifies that gh CLI is installed and authenticated.
func (g *GitHub) CheckGHInstalled(ctx context.Context) error {
	_, err := g.runGH(ctx, "auth", "status")
	if err != nil {
		return fmt.Errorf("gh CLI not installed or not authenticated: %w", err)
	}
	return nil
}

// runGH executes a gh CLI command.
func (g *GitHub) runGH(ctx context.Context, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, CommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "gh", args...)
	cmd.Dir = g.repoPath

	// Set token if provided
	if g.token != "" {
		cmd.Env = append(cmd.Environ(), "GH_TOKEN="+g.token)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("gh command timed out")
		}
		return "", fmt.Errorf("gh %s failed: %s", strings.Join(args, " "), stderr.String())
	}

	return stdout.String(), nil
}

// FormatPR formats a PR for display.
func FormatPR(pr *PRInfo) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("#%d: %s\n", pr.Number, pr.Title))
	sb.WriteString(fmt.Sprintf("URL: %s\n", pr.URL))
	sb.WriteString(fmt.Sprintf("State: %s\n", pr.State))
	sb.WriteString(fmt.Sprintf("Branch: %s → %s\n", pr.HeadRef, pr.BaseRef))
	return sb.String()
}
