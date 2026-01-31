// Package slack provides message handlers for the bot.
package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/ireland-samantha/stormstack-dev-bot/internal/claude"
	"github.com/ireland-samantha/stormstack-dev-bot/internal/codebase"
	"github.com/ireland-samantha/stormstack-dev-bot/internal/config"
	"github.com/ireland-samantha/stormstack-dev-bot/internal/executor"
	"github.com/ireland-samantha/stormstack-dev-bot/internal/git"
	"github.com/ireland-samantha/stormstack-dev-bot/internal/storage"
)

// Handler handles incoming messages and coordinates with Claude.
type Handler struct {
	conversation *claude.ConversationManager
	toolExecutor *ToolExecutor
	logger       *slog.Logger
}

// NewHandler creates a new message handler.
func NewHandler(
	cfg *config.Config,
	repoPath string,
	store storage.ConversationStore,
	logger *slog.Logger,
) *Handler {
	// Create Claude client
	claudeClient := claude.NewClient(cfg.AnthropicAPIKey)

	// Create tool executor
	toolExecutor := NewToolExecutor(repoPath, cfg, logger)

	// Load system prompt
	systemPrompt := claude.LoadSystemPrompt(repoPath, cfg.GuidelinesFile)

	// Create conversation manager
	conversation := claude.NewConversationManager(
		claudeClient,
		store,
		systemPrompt,
		toolExecutor.Execute,
		logger,
	)

	return &Handler{
		conversation: conversation,
		toolExecutor: toolExecutor,
		logger:       logger,
	}
}

// HandleMessage processes an incoming message.
func (h *Handler) HandleMessage(ctx context.Context, msg *IncomingMessage) (*OutgoingMessage, error) {
	h.logger.Info("handling message",
		"user", msg.UserID,
		"channel", msg.ChannelID,
		"thread", msg.ThreadTS,
	)

	// Use thread timestamp as conversation ID
	conversationID := msg.ThreadTS
	if conversationID == "" {
		conversationID = msg.ChannelID + "-" + msg.UserID
	}

	// Process with Claude
	response, err := h.conversation.ProcessMessage(ctx, conversationID, msg.ChannelID, msg.Text)
	if err != nil {
		h.logger.Error("failed to process message", "error", err)
		return &OutgoingMessage{
			Text:     fmt.Sprintf("Sorry, I encountered an error: %v", err),
			ThreadTS: msg.ThreadTS,
		}, nil
	}

	return &OutgoingMessage{
		Text:     response,
		ThreadTS: msg.ThreadTS,
	}, nil
}

// ToolExecutor executes tools for Claude.
type ToolExecutor struct {
	reader   *codebase.Reader
	writer   *codebase.Writer
	searcher *codebase.Searcher
	runner   *executor.Runner
	gitOps   *git.Operations
	github   *git.GitHub
	cfg      *config.Config
	logger   *slog.Logger
}

// NewToolExecutor creates a new tool executor.
func NewToolExecutor(repoPath string, cfg *config.Config, logger *slog.Logger) *ToolExecutor {
	return &ToolExecutor{
		reader:   codebase.NewReader(repoPath),
		writer:   codebase.NewWriter(repoPath),
		searcher: codebase.NewSearcher(repoPath),
		runner:   executor.NewRunner(repoPath, cfg.BuildCmd, cfg.TestCmd),
		gitOps:   git.NewOperations(repoPath),
		github:   git.NewGitHub(repoPath, cfg.GitHubToken),
		cfg:      cfg,
		logger:   logger,
	}
}

// Execute executes a tool and returns the result.
func (e *ToolExecutor) Execute(ctx context.Context, name string, input json.RawMessage) (string, error) {
	e.logger.Debug("executing tool", "name", name)

	switch name {
	// Code Understanding
	case "read_file":
		return e.readFile(input)
	case "list_files":
		return e.listFiles(input)
	case "search_code":
		return e.searchCode(input)
	case "get_tree":
		return e.getTree(input)

	// Code Modification
	case "write_file":
		return e.writeFile(input)
	case "edit_file":
		return e.editFile(input)

	// Build & Test
	case "run_command":
		return e.runCommand(ctx, input)
	case "run_build":
		return e.runBuild(ctx, input)
	case "run_tests":
		return e.runTests(ctx, input)

	// Git Operations
	case "git_status":
		return e.gitStatus(ctx)
	case "git_diff":
		return e.gitDiff(ctx, input)
	case "git_log":
		return e.gitLog(ctx, input)
	case "create_branch":
		return e.createBranch(ctx, input)
	case "commit":
		return e.commit(ctx, input)
	case "push":
		return e.push(ctx, input)
	case "create_pr":
		return e.createPR(ctx, input)

	// Project Intelligence
	case "get_guidelines":
		return e.getGuidelines()
	case "find_tests":
		return e.findTests(input)
	case "analyze_failures":
		return e.analyzeFailures(input)

	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}

// Tool implementations

func (e *ToolExecutor) readFile(input json.RawMessage) (string, error) {
	var params struct {
		Path      string `json:"path"`
		StartLine int    `json:"start_line"`
		EndLine   int    `json:"end_line"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", err
	}

	if params.StartLine > 0 || params.EndLine > 0 {
		return e.reader.ReadFileLines(params.Path, params.StartLine, params.EndLine)
	}
	return e.reader.ReadFile(params.Path)
}

func (e *ToolExecutor) listFiles(input json.RawMessage) (string, error) {
	var params struct {
		Pattern string `json:"pattern"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", err
	}

	files, err := e.searcher.ListFiles(params.Pattern)
	if err != nil {
		return "", err
	}

	if len(files) == 0 {
		return "No files found matching pattern: " + params.Pattern, nil
	}

	return fmt.Sprintf("Found %d files:\n%s", len(files), joinLines(files)), nil
}

func (e *ToolExecutor) searchCode(input json.RawMessage) (string, error) {
	var params struct {
		Pattern       string `json:"pattern"`
		Path          string `json:"path"`
		CaseSensitive bool   `json:"case_sensitive"`
		MaxResults    int    `json:"max_results"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", err
	}

	results, err := e.searcher.SearchCode(params.Pattern, params.Path, params.CaseSensitive, params.MaxResults)
	if err != nil {
		return "", err
	}

	if len(results) == 0 {
		return "No matches found for pattern: " + params.Pattern, nil
	}

	return fmt.Sprintf("Found %d matches:\n%s", len(results), codebase.FormatSearchResults(results)), nil
}

func (e *ToolExecutor) getTree(input json.RawMessage) (string, error) {
	var params struct {
		Path     string `json:"path"`
		MaxDepth int    `json:"max_depth"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", err
	}

	return e.searcher.GetTree(params.Path, params.MaxDepth)
}

func (e *ToolExecutor) writeFile(input json.RawMessage) (string, error) {
	var params struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", err
	}

	if err := e.writer.WriteFile(params.Path, params.Content); err != nil {
		return "", err
	}

	return fmt.Sprintf("Successfully wrote %d bytes to %s", len(params.Content), params.Path), nil
}

func (e *ToolExecutor) editFile(input json.RawMessage) (string, error) {
	var params struct {
		Path    string `json:"path"`
		OldText string `json:"old_text"`
		NewText string `json:"new_text"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", err
	}

	if err := e.writer.EditFile(params.Path, params.OldText, params.NewText); err != nil {
		return "", err
	}

	return fmt.Sprintf("Successfully edited %s", params.Path), nil
}

func (e *ToolExecutor) runCommand(ctx context.Context, input json.RawMessage) (string, error) {
	var params struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", err
	}

	result, err := e.runner.RunCommand(ctx, params.Command)
	if err != nil {
		return "", err
	}

	return result.FormatResult(), nil
}

func (e *ToolExecutor) runBuild(ctx context.Context, input json.RawMessage) (string, error) {
	var params struct {
		Args string `json:"args"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", err
	}

	result, err := e.runner.RunBuild(ctx, params.Args)
	if err != nil {
		return "", err
	}

	return result.FormatResult(), nil
}

func (e *ToolExecutor) runTests(ctx context.Context, input json.RawMessage) (string, error) {
	var params struct {
		Args string `json:"args"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", err
	}

	result, err := e.runner.RunTests(ctx, params.Args)
	if err != nil {
		return "", err
	}

	return result.FormatResult(), nil
}

func (e *ToolExecutor) gitStatus(ctx context.Context) (string, error) {
	return e.gitOps.Status(ctx)
}

func (e *ToolExecutor) gitDiff(ctx context.Context, input json.RawMessage) (string, error) {
	var params struct {
		Staged bool   `json:"staged"`
		Ref    string `json:"ref"`
		Path   string `json:"path"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", err
	}

	return e.gitOps.Diff(ctx, params.Staged, params.Ref, params.Path)
}

func (e *ToolExecutor) gitLog(ctx context.Context, input json.RawMessage) (string, error) {
	var params struct {
		Count  int    `json:"count"`
		Path   string `json:"path"`
		Format string `json:"format"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", err
	}

	return e.gitOps.Log(ctx, params.Count, params.Path, params.Format)
}

func (e *ToolExecutor) createBranch(ctx context.Context, input json.RawMessage) (string, error) {
	var params struct {
		Name string `json:"name"`
		From string `json:"from"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", err
	}

	if err := e.gitOps.CreateBranch(ctx, params.Name, params.From); err != nil {
		return "", err
	}

	return fmt.Sprintf("Created and switched to branch: %s", params.Name), nil
}

func (e *ToolExecutor) commit(ctx context.Context, input json.RawMessage) (string, error) {
	var params struct {
		Message string   `json:"message"`
		Files   []string `json:"files"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", err
	}

	if err := e.gitOps.Commit(ctx, params.Message, params.Files); err != nil {
		return "", err
	}

	return fmt.Sprintf("Committed: %s", params.Message), nil
}

func (e *ToolExecutor) push(ctx context.Context, input json.RawMessage) (string, error) {
	var params struct {
		SetUpstream bool `json:"set_upstream"`
	}
	// Default to true for set_upstream
	params.SetUpstream = true
	if err := json.Unmarshal(input, &params); err != nil {
		return "", err
	}

	if err := e.gitOps.Push(ctx, params.SetUpstream); err != nil {
		return "", err
	}

	branch, _ := e.gitOps.CurrentBranch(ctx)
	return fmt.Sprintf("Pushed branch: %s", branch), nil
}

func (e *ToolExecutor) createPR(ctx context.Context, input json.RawMessage) (string, error) {
	var params struct {
		Title string `json:"title"`
		Body  string `json:"body"`
		Base  string `json:"base"`
		Draft bool   `json:"draft"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", err
	}

	pr, err := e.github.CreatePR(ctx, params.Title, params.Body, params.Base, params.Draft)
	if err != nil {
		return "", err
	}

	return git.FormatPR(pr), nil
}

func (e *ToolExecutor) getGuidelines() (string, error) {
	content, err := e.reader.ReadFile(e.cfg.GuidelinesFile)
	if err != nil {
		// Try CLAUDE.md as fallback
		content, err = e.reader.ReadFile("CLAUDE.md")
		if err != nil {
			return "No guidelines file found in repository.", nil
		}
	}
	return content, nil
}

func (e *ToolExecutor) findTests(input json.RawMessage) (string, error) {
	var params struct {
		SourceFile string `json:"source_file"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", err
	}

	tests, err := e.searcher.FindTests(params.SourceFile)
	if err != nil {
		return "", err
	}

	if len(tests) == 0 {
		return fmt.Sprintf("No test files found for: %s", params.SourceFile), nil
	}

	return fmt.Sprintf("Found test files:\n%s", joinLines(tests)), nil
}

func (e *ToolExecutor) analyzeFailures(input json.RawMessage) (string, error) {
	var params struct {
		Output string `json:"output"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", err
	}

	result := executor.AnalyzeOutput(params.Output)
	return result.Summary(), nil
}

// Helper functions

func joinLines(lines []string) string {
	result := ""
	for _, line := range lines {
		result += line + "\n"
	}
	return result
}
