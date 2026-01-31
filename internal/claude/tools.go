// Package claude provides tool definitions for Claude.
package claude

import (
	"github.com/anthropics/anthropic-sdk-go"
)

// GetAllTools returns all available tools for Claude.
func GetAllTools() []anthropic.ToolUnionParam {
	return []anthropic.ToolUnionParam{
		// Code Understanding
		ReadFileTool(),
		ListFilesTool(),
		SearchCodeTool(),
		GetTreeTool(),

		// Code Modification
		WriteFileTool(),
		EditFileTool(),

		// Build & Test
		RunCommandTool(),
		RunBuildTool(),
		RunTestsTool(),

		// Git Operations
		GitStatusTool(),
		GitDiffTool(),
		GitLogTool(),
		CreateBranchTool(),
		CommitTool(),
		PushTool(),
		CreatePRTool(),
		GetPRTool(),

		// Project Intelligence
		GetGuidelinesTool(),
		FindTestsTool(),
		AnalyzeFailuresTool(),
	}
}

// helper creates a tool with the given name, description and schema
func makeTool(name, description string, properties map[string]any, required []string) anthropic.ToolUnionParam {
	schema := anthropic.ToolInputSchemaParam{
		Properties: properties,
	}
	if len(required) > 0 {
		schema.ExtraFields = map[string]any{
			"required": required,
		}
	}

	return anthropic.ToolUnionParam{
		OfTool: &anthropic.ToolParam{
			Name:        name,
			Description: anthropic.String(description),
			InputSchema: schema,
		},
	}
}

// Code Understanding Tools

// ReadFileTool returns the read_file tool definition.
func ReadFileTool() anthropic.ToolUnionParam {
	return makeTool(
		"read_file",
		"Read the contents of a file at the given path. Returns the file content as text.",
		map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "The relative path to the file from the repository root",
			},
			"start_line": map[string]any{
				"type":        "integer",
				"description": "Optional start line number (1-indexed). If provided, only returns lines from this point.",
			},
			"end_line": map[string]any{
				"type":        "integer",
				"description": "Optional end line number (1-indexed). If provided, only returns lines up to this point.",
			},
		},
		[]string{"path"},
	)
}

// ListFilesTool returns the list_files tool definition.
func ListFilesTool() anthropic.ToolUnionParam {
	return makeTool(
		"list_files",
		"List files matching a glob pattern. Returns a list of file paths.",
		map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "Glob pattern to match files (e.g., '**/*.java', 'src/**/*.go')",
			},
		},
		[]string{"pattern"},
	)
}

// SearchCodeTool returns the search_code tool definition.
func SearchCodeTool() anthropic.ToolUnionParam {
	return makeTool(
		"search_code",
		"Search for a pattern in the codebase using grep-like syntax. Returns matching lines with file paths and line numbers.",
		map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "The search pattern (supports regex)",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "Optional path to limit search scope (can be a directory or glob pattern)",
			},
			"case_sensitive": map[string]any{
				"type":        "boolean",
				"description": "Whether the search should be case-sensitive (default: false)",
			},
			"max_results": map[string]any{
				"type":        "integer",
				"description": "Maximum number of results to return (default: 50)",
			},
		},
		[]string{"pattern"},
	)
}

// GetTreeTool returns the get_tree tool definition.
func GetTreeTool() anthropic.ToolUnionParam {
	return makeTool(
		"get_tree",
		"Get the directory structure of the repository or a subdirectory.",
		map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "The path to get the tree for (default: repository root)",
			},
			"max_depth": map[string]any{
				"type":        "integer",
				"description": "Maximum depth to traverse (default: 3)",
			},
		},
		nil,
	)
}

// Code Modification Tools

// WriteFileTool returns the write_file tool definition.
func WriteFileTool() anthropic.ToolUnionParam {
	return makeTool(
		"write_file",
		"Write content to a file. Creates the file if it doesn't exist, or overwrites if it does.",
		map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "The relative path to the file from the repository root",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "The content to write to the file",
			},
		},
		[]string{"path", "content"},
	)
}

// EditFileTool returns the edit_file tool definition.
func EditFileTool() anthropic.ToolUnionParam {
	return makeTool(
		"edit_file",
		"Make a targeted edit to a file by finding and replacing specific text. Use this for surgical changes rather than rewriting entire files.",
		map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "The relative path to the file from the repository root",
			},
			"old_text": map[string]any{
				"type":        "string",
				"description": "The exact text to find and replace (must be unique in the file)",
			},
			"new_text": map[string]any{
				"type":        "string",
				"description": "The text to replace old_text with",
			},
		},
		[]string{"path", "old_text", "new_text"},
	)
}

// Build & Test Tools

// RunCommandTool returns the run_command tool definition.
func RunCommandTool() anthropic.ToolUnionParam {
	return makeTool(
		"run_command",
		"Run a shell command in the repository directory. Only allowed commands: git, gh, ls, cat, head, tail, find, grep, wc, diff, echo, pwd, date, which.",
		map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "The command to run",
			},
		},
		[]string{"command"},
	)
}

// RunBuildTool returns the run_build tool definition.
func RunBuildTool() anthropic.ToolUnionParam {
	return makeTool(
		"run_build",
		"Run the project's build command (configured via STORMSTACK_BUILD_CMD).",
		map[string]any{
			"args": map[string]any{
				"type":        "string",
				"description": "Optional additional arguments to pass to the build command",
			},
		},
		nil,
	)
}

// RunTestsTool returns the run_tests tool definition.
func RunTestsTool() anthropic.ToolUnionParam {
	return makeTool(
		"run_tests",
		"Run the project's test command (configured via STORMSTACK_TEST_CMD).",
		map[string]any{
			"args": map[string]any{
				"type":        "string",
				"description": "Optional additional arguments (e.g., specific test file or pattern)",
			},
		},
		nil,
	)
}

// Git Operations Tools

// GitStatusTool returns the git_status tool definition.
func GitStatusTool() anthropic.ToolUnionParam {
	return makeTool(
		"git_status",
		"Show the current git status including modified, staged, and untracked files.",
		map[string]any{},
		nil,
	)
}

// GitDiffTool returns the git_diff tool definition.
func GitDiffTool() anthropic.ToolUnionParam {
	return makeTool(
		"git_diff",
		"Show git diff of changes. Can show staged, unstaged, or between commits.",
		map[string]any{
			"staged": map[string]any{
				"type":        "boolean",
				"description": "If true, show staged changes only (--cached)",
			},
			"ref": map[string]any{
				"type":        "string",
				"description": "Optional commit/branch reference to diff against",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "Optional file path to limit diff to",
			},
		},
		nil,
	)
}

// GitLogTool returns the git_log tool definition.
func GitLogTool() anthropic.ToolUnionParam {
	return makeTool(
		"git_log",
		"Show git commit history.",
		map[string]any{
			"count": map[string]any{
				"type":        "integer",
				"description": "Number of commits to show (default: 10)",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "Optional file path to show history for",
			},
			"format": map[string]any{
				"type":        "string",
				"description": "Output format: 'oneline', 'short', 'medium', 'full' (default: 'oneline')",
			},
		},
		nil,
	)
}

// CreateBranchTool returns the create_branch tool definition.
func CreateBranchTool() anthropic.ToolUnionParam {
	return makeTool(
		"create_branch",
		"Create a new git branch and switch to it.",
		map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "The branch name (will be sanitized)",
			},
			"from": map[string]any{
				"type":        "string",
				"description": "Optional base branch/commit to create from (default: current HEAD)",
			},
		},
		[]string{"name"},
	)
}

// CommitTool returns the commit tool definition.
func CommitTool() anthropic.ToolUnionParam {
	return makeTool(
		"commit",
		"Stage files and create a git commit.",
		map[string]any{
			"message": map[string]any{
				"type":        "string",
				"description": "The commit message",
			},
			"files": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "List of files to stage (default: all modified files)",
			},
		},
		[]string{"message"},
	)
}

// PushTool returns the push tool definition.
func PushTool() anthropic.ToolUnionParam {
	return makeTool(
		"push",
		"Push the current branch to the remote repository.",
		map[string]any{
			"set_upstream": map[string]any{
				"type":        "boolean",
				"description": "Whether to set upstream tracking (-u flag, default: true for new branches)",
			},
		},
		nil,
	)
}

// CreatePRTool returns the create_pr tool definition.
func CreatePRTool() anthropic.ToolUnionParam {
	return makeTool(
		"create_pr",
		"Create a GitHub pull request using the gh CLI.",
		map[string]any{
			"title": map[string]any{
				"type":        "string",
				"description": "The PR title",
			},
			"body": map[string]any{
				"type":        "string",
				"description": "The PR description/body",
			},
			"base": map[string]any{
				"type":        "string",
				"description": "The base branch to merge into (default: main)",
			},
			"draft": map[string]any{
				"type":        "boolean",
				"description": "Whether to create as draft PR (default: false)",
			},
		},
		[]string{"title", "body"},
	)
}

// GetPRTool returns the get_pr tool definition.
func GetPRTool() anthropic.ToolUnionParam {
	return makeTool(
		"get_pr",
		"Get details about a GitHub pull request including title, description, and diff. Use this to review PRs when given a PR URL or number.",
		map[string]any{
			"url": map[string]any{
				"type":        "string",
				"description": "The PR URL (e.g., https://github.com/owner/repo/pull/123) or just the PR number if in the same repo",
			},
		},
		[]string{"url"},
	)
}

// Project Intelligence Tools

// GetGuidelinesTool returns the get_guidelines tool definition.
func GetGuidelinesTool() anthropic.ToolUnionParam {
	return makeTool(
		"get_guidelines",
		"Load project guidelines from CLAUDE.md or a custom guidelines file. Use this to understand project conventions and coding standards.",
		map[string]any{},
		nil,
	)
}

// FindTestsTool returns the find_tests tool definition.
func FindTestsTool() anthropic.ToolUnionParam {
	return makeTool(
		"find_tests",
		"Find the test file(s) associated with a source file.",
		map[string]any{
			"source_file": map[string]any{
				"type":        "string",
				"description": "The source file path to find tests for",
			},
		},
		[]string{"source_file"},
	)
}

// AnalyzeFailuresTool returns the analyze_failures tool definition.
func AnalyzeFailuresTool() anthropic.ToolUnionParam {
	return makeTool(
		"analyze_failures",
		"Analyze test or build output to identify and summarize failures.",
		map[string]any{
			"output": map[string]any{
				"type":        "string",
				"description": "The build/test output to analyze",
			},
		},
		[]string{"output"},
	)
}
