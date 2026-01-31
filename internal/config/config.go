// Package config provides configuration loading for the StormStack Dev Bot.
package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Mode represents the repository access mode.
type Mode string

const (
	ModeLocal   Mode = "local"
	ModeSandbox Mode = "sandbox"
)

// Config holds all configuration for the bot.
type Config struct {
	// Mode is either "local" or "sandbox"
	Mode Mode

	// Local mode settings
	RepoPath string

	// Sandbox mode settings
	GitHubRepo    string
	GitHubToken   string
	WorkspacePath string

	// Slack settings
	SlackBotToken string
	SlackAppToken string

	// Claude settings
	AnthropicAPIKey string

	// Build commands
	BuildCmd string
	TestCmd  string

	// Optional settings
	GuidelinesFile string
	LogLevel       string
}

// Load loads configuration from environment variables.
func Load() (*Config, error) {
	v := viper.New()

	// Set prefix for environment variables
	v.SetEnvPrefix("STORMSTACK")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Set defaults
	v.SetDefault("MODE", "local")
	v.SetDefault("GUIDELINES_FILE", "CLAUDE.md")
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("BUILD_CMD", "./build.sh build")
	v.SetDefault("TEST_CMD", "./build.sh test")
	v.SetDefault("WORKSPACE_PATH", "./workspace")

	cfg := &Config{
		Mode:            Mode(v.GetString("MODE")),
		RepoPath:        v.GetString("REPO_PATH"),
		GitHubRepo:      v.GetString("GITHUB_REPO"),
		GitHubToken:     v.GetString("GITHUB_TOKEN"),
		WorkspacePath:   v.GetString("WORKSPACE_PATH"),
		SlackBotToken:   v.GetString("SLACK_BOT_TOKEN"),
		SlackAppToken:   v.GetString("SLACK_APP_TOKEN"),
		AnthropicAPIKey: v.GetString("ANTHROPIC_API_KEY"),
		BuildCmd:        v.GetString("BUILD_CMD"),
		TestCmd:         v.GetString("TEST_CMD"),
		GuidelinesFile:  v.GetString("GUIDELINES_FILE"),
		LogLevel:        v.GetString("LOG_LEVEL"),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks that all required configuration is present.
func (c *Config) Validate() error {
	var errs []string

	// Validate mode
	if c.Mode != ModeLocal && c.Mode != ModeSandbox {
		errs = append(errs, fmt.Sprintf("invalid mode %q, must be 'local' or 'sandbox'", c.Mode))
	}

	// Mode-specific validation
	switch c.Mode {
	case ModeLocal:
		if c.RepoPath == "" {
			errs = append(errs, "STORMSTACK_REPO_PATH is required in local mode")
		} else if !isDirectory(c.RepoPath) {
			errs = append(errs, fmt.Sprintf("STORMSTACK_REPO_PATH %q does not exist or is not a directory", c.RepoPath))
		}
	case ModeSandbox:
		if c.GitHubRepo == "" {
			errs = append(errs, "STORMSTACK_GITHUB_REPO is required in sandbox mode")
		}
		if c.GitHubToken == "" {
			errs = append(errs, "STORMSTACK_GITHUB_TOKEN is required in sandbox mode")
		}
	}

	// Required for all modes
	if c.SlackBotToken == "" {
		errs = append(errs, "STORMSTACK_SLACK_BOT_TOKEN is required")
	}
	if c.SlackAppToken == "" {
		errs = append(errs, "STORMSTACK_SLACK_APP_TOKEN is required")
	}
	if c.AnthropicAPIKey == "" {
		errs = append(errs, "STORMSTACK_ANTHROPIC_API_KEY is required")
	}

	if len(errs) > 0 {
		return errors.New("configuration errors:\n  - " + strings.Join(errs, "\n  - "))
	}

	return nil
}

// isDirectory checks if a path exists and is a directory.
func isDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
