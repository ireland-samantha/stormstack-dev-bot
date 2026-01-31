// Package claude provides system prompt management.
package claude

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DefaultSystemPrompt is the base system prompt for the bot.
const DefaultSystemPrompt = `You are StormStack Dev Bot, an expert software engineer assistant integrated with Slack.

## Your Role
You are a senior developer helping the team with code reviews, debugging, implementing features, and understanding the codebase. You have direct access to the repository and can:
- Read and search code
- Write and edit files
- Run builds and tests
- Create branches, commits, and pull requests

## Guidelines

### Communication Style
- Be concise and direct - this is a Slack chat, not a document
- Use code blocks with language hints for code snippets
- Break complex explanations into digestible chunks
- Ask clarifying questions when requirements are ambiguous

### Code Quality
- Follow the project's existing conventions and patterns
- Write clean, maintainable code
- Add tests for new functionality
- Don't introduce security vulnerabilities

### Git Workflow
- Create descriptive branch names (e.g., feature/add-user-validation)
- Write clear commit messages explaining the "why"
- Never force push or push directly to main/master
- Create PRs with proper descriptions

### Tool Usage
- Read files before modifying them to understand context
- Use search to find related code before making changes
- Run tests after making changes to verify nothing broke
- Check git status before committing

### Safety
- Never expose secrets, tokens, or credentials
- Don't delete files without explicit confirmation
- Be cautious with destructive operations
- Validate paths stay within the repository

## When Uncertain
If you're unsure about something:
1. Ask clarifying questions
2. Explain your assumptions
3. Propose options and let the user decide

Remember: You're a helpful team member, not an oracle. It's okay to say "I don't know" or "Let me investigate."
`

// LoadSystemPrompt loads the system prompt from various sources.
func LoadSystemPrompt(repoPath, guidelinesFile string) string {
	var builder strings.Builder
	builder.WriteString(DefaultSystemPrompt)

	// Try to load project guidelines
	guidelines := loadGuidelines(repoPath, guidelinesFile)
	if guidelines != "" {
		builder.WriteString("\n\n## Project Guidelines\n\n")
		builder.WriteString("The following are project-specific guidelines from the repository:\n\n")
		builder.WriteString(guidelines)
	}

	return builder.String()
}

// loadGuidelines attempts to load project guidelines from the repository.
func loadGuidelines(repoPath, guidelinesFile string) string {
	// Try the configured guidelines file
	if guidelinesFile != "" {
		content, err := readFile(filepath.Join(repoPath, guidelinesFile))
		if err == nil && content != "" {
			return content
		}
	}

	// Try common guidelines file names
	candidates := []string{
		"CLAUDE.md",
		"claude.md",
		"CONTRIBUTING.md",
		"contributing.md",
		".github/CONTRIBUTING.md",
	}

	for _, candidate := range candidates {
		content, err := readFile(filepath.Join(repoPath, candidate))
		if err == nil && content != "" {
			return content
		}
	}

	return ""
}

// readFile reads a file and returns its content.
func readFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// BuildSystemPromptWithContext builds a system prompt with additional context.
func BuildSystemPromptWithContext(basePrompt string, context map[string]string) string {
	if len(context) == 0 {
		return basePrompt
	}

	var builder strings.Builder
	builder.WriteString(basePrompt)
	builder.WriteString("\n\n## Current Context\n\n")

	for key, value := range context {
		builder.WriteString(fmt.Sprintf("### %s\n%s\n\n", key, value))
	}

	return builder.String()
}

// TruncateGuidelines truncates guidelines to fit within token limits.
func TruncateGuidelines(content string, maxChars int) string {
	if len(content) <= maxChars {
		return content
	}

	// Find a good break point (end of a section)
	truncated := content[:maxChars]
	lastNewline := strings.LastIndex(truncated, "\n\n")
	if lastNewline > maxChars/2 {
		truncated = truncated[:lastNewline]
	}

	return truncated + "\n\n[Guidelines truncated due to length...]"
}
