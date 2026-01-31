// Package main is the entry point for the StormStack Dev Bot.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/ireland-samantha/stormstack-dev-bot/internal/config"
	"github.com/ireland-samantha/stormstack-dev-bot/internal/repo"
	"github.com/ireland-samantha/stormstack-dev-bot/internal/slack"
	"github.com/ireland-samantha/stormstack-dev-bot/internal/storage"
)

func main() {
	// Setup logger
	logLevel := slog.LevelInfo
	if os.Getenv("STORMSTACK_LOG_LEVEL") == "debug" {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	logger.Info("Starting StormStack Dev Bot...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}
	logger.Info("Configuration loaded",
		"mode", cfg.Mode,
		"log_level", cfg.LogLevel,
	)

	// Setup repository manager
	repoManager, err := repo.NewManager(cfg)
	if err != nil {
		logger.Error("Failed to create repository manager", "error", err)
		os.Exit(1)
	}

	// Ensure repository is ready
	logger.Info("Preparing repository...")
	if err := repoManager.EnsureReady(); err != nil {
		logger.Error("Failed to prepare repository", "error", err)
		os.Exit(1)
	}
	logger.Info("Repository ready", "path", repoManager.GetRepoPath())

	// Create conversation store
	store := storage.NewMemoryStore()

	// Create message handler
	handler := slack.NewHandler(cfg, repoManager.GetRepoPath(), store, logger)

	// Create Slack bot
	bot, err := slack.NewBot(cfg, handler.HandleMessage, logger)
	if err != nil {
		logger.Error("Failed to create Slack bot", "error", err)
		os.Exit(1)
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		logger.Info("Received shutdown signal", "signal", sig)
		cancel()
	}()

	// Run the bot
	logger.Info("StormStack Dev Bot is running. Press Ctrl+C to stop.")
	if err := bot.Run(ctx); err != nil && ctx.Err() == nil {
		logger.Error("Bot error", "error", err)
		os.Exit(1)
	}

	logger.Info("StormStack Dev Bot stopped.")
}
