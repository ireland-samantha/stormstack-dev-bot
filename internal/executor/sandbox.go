// Package executor provides command sandboxing and validation.
package executor

import (
	"fmt"
	"strings"
)

// AllowedCommands is the list of commands allowed to be executed.
var AllowedCommands = []string{
	// Git commands
	"git",
	// GitHub CLI
	"gh",
	// Read-only file operations
	"ls", "cat", "head", "tail", "find", "grep", "wc", "diff",
	// Utilities
	"echo", "pwd", "date", "which", "file", "stat",
	// Build tools (these are typically safe read operations)
	"make", "mvn", "gradle", "npm", "yarn", "pnpm", "cargo", "go",
}

// DangerousPatterns are patterns that indicate dangerous commands.
var DangerousPatterns = []string{
	// Destructive operations
	"rm -rf /",
	"rm -rf ~",
	"rm -rf $HOME",
	"> /dev/",
	"mkfs",
	"dd if=",
	":(){:|:&};:",
	// Network exfiltration
	"curl.*|.*sh",
	"wget.*|.*sh",
	// Privilege escalation
	"sudo",
	"su -",
	"chmod 777",
	"chown root",
	// Sensitive file access
	"/etc/passwd",
	"/etc/shadow",
	"~/.ssh",
	".ssh/",
	// Environment variable exposure
	"printenv",
	"export.*=",
}

// BlockedGitCommands are git subcommands that should not be allowed.
var BlockedGitCommands = []string{
	"push --force",
	"push -f",
	"reset --hard",
	"clean -f",
	"clean -fd",
	"checkout .",
	"restore .",
}

// ValidateCommand checks if a command is safe to execute.
func ValidateCommand(command string) error {
	// Trim and normalize
	command = strings.TrimSpace(command)
	if command == "" {
		return fmt.Errorf("empty command")
	}

	// Check for dangerous patterns
	lowerCmd := strings.ToLower(command)
	for _, pattern := range DangerousPatterns {
		if strings.Contains(lowerCmd, strings.ToLower(pattern)) {
			return fmt.Errorf("command contains dangerous pattern: %s", pattern)
		}
	}

	// Extract the base command
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	baseCmd := parts[0]

	// Handle pipe chains - check each command
	if strings.Contains(command, "|") {
		return validatePipeChain(command)
	}

	// Handle command chaining with && or ;
	if strings.Contains(command, "&&") || strings.Contains(command, ";") {
		return validateChainedCommands(command)
	}

	// Check if base command is allowed
	if !isAllowedCommand(baseCmd) {
		return fmt.Errorf("command not allowed: %s", baseCmd)
	}

	// Special validation for git commands
	if baseCmd == "git" {
		if err := validateGitCommand(command); err != nil {
			return err
		}
	}

	return nil
}

// validatePipeChain validates each command in a pipe chain.
func validatePipeChain(command string) error {
	pipes := strings.Split(command, "|")
	for _, pipe := range pipes {
		pipe = strings.TrimSpace(pipe)
		if pipe == "" {
			continue
		}

		parts := strings.Fields(pipe)
		if len(parts) == 0 {
			continue
		}

		if !isAllowedCommand(parts[0]) {
			return fmt.Errorf("command not allowed in pipe: %s", parts[0])
		}
	}
	return nil
}

// validateChainedCommands validates each command in a chain.
func validateChainedCommands(command string) error {
	// Split by && and ;
	command = strings.ReplaceAll(command, "&&", "\n")
	command = strings.ReplaceAll(command, ";", "\n")

	for _, cmd := range strings.Split(command, "\n") {
		cmd = strings.TrimSpace(cmd)
		if cmd == "" {
			continue
		}

		if err := ValidateCommand(cmd); err != nil {
			return err
		}
	}
	return nil
}

// isAllowedCommand checks if a command is in the allowed list.
func isAllowedCommand(cmd string) bool {
	// Handle path-qualified commands
	if strings.Contains(cmd, "/") {
		parts := strings.Split(cmd, "/")
		cmd = parts[len(parts)-1]
	}

	for _, allowed := range AllowedCommands {
		if cmd == allowed {
			return true
		}
	}
	return false
}

// validateGitCommand performs additional validation for git commands.
func validateGitCommand(command string) error {
	lowerCmd := strings.ToLower(command)

	// Check for blocked git operations
	for _, blocked := range BlockedGitCommands {
		if strings.Contains(lowerCmd, strings.ToLower(blocked)) {
			return fmt.Errorf("git operation not allowed: %s", blocked)
		}
	}

	// Prevent pushing to main/master
	if strings.Contains(lowerCmd, "push") {
		if strings.Contains(lowerCmd, "main") || strings.Contains(lowerCmd, "master") {
			// Allow if it's setting upstream
			if !strings.Contains(lowerCmd, "-u") && !strings.Contains(lowerCmd, "--set-upstream") {
				return fmt.Errorf("direct push to main/master not allowed")
			}
		}
	}

	return nil
}

// SanitizeBranchName sanitizes a branch name for safe use.
func SanitizeBranchName(name string) string {
	// Remove or replace unsafe characters
	replacer := strings.NewReplacer(
		" ", "-",
		"..", "-",
		"~", "-",
		"^", "-",
		":", "-",
		"?", "-",
		"*", "-",
		"[", "-",
		"]", "-",
		"\\", "-",
		"@{", "-",
	)

	name = replacer.Replace(name)

	// Remove leading/trailing dashes and slashes
	name = strings.Trim(name, "-/")

	// Ensure reasonable length
	if len(name) > 100 {
		name = name[:100]
	}

	return name
}

// SanitizeCommitMessage sanitizes a commit message.
func SanitizeCommitMessage(message string) string {
	// Remove any potential command injection
	message = strings.ReplaceAll(message, "`", "'")
	message = strings.ReplaceAll(message, "$", "")
	message = strings.ReplaceAll(message, "\\", "")

	return message
}
