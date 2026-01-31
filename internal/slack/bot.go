// Package slack provides Slack bot integration using Socket Mode.
package slack

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/ireland-samantha/stormstack-dev-bot/internal/config"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

// MessageHandler is called when the bot receives a message to process.
type MessageHandler func(ctx context.Context, msg *IncomingMessage) (*OutgoingMessage, error)

// IncomingMessage represents a message received by the bot.
type IncomingMessage struct {
	// Text is the message content (with bot mention stripped)
	Text string
	// UserID is the Slack user ID of the sender
	UserID string
	// ChannelID is the channel where the message was sent
	ChannelID string
	// ThreadTS is the thread timestamp (for threading replies)
	ThreadTS string
	// IsDM indicates if this is a direct message
	IsDM bool
}

// OutgoingMessage represents a message to send.
type OutgoingMessage struct {
	// Text is the message content
	Text string
	// ThreadTS is the thread timestamp to reply in
	ThreadTS string
	// Blocks are optional Slack blocks for rich formatting
	Blocks []slack.Block
}

// Bot manages the Slack connection and event handling.
type Bot struct {
	client       *slack.Client
	socketClient *socketmode.Client
	handler      MessageHandler
	botUserID    string
	logger       *slog.Logger
}

// NewBot creates a new Slack bot instance.
func NewBot(cfg *config.Config, handler MessageHandler, logger *slog.Logger) (*Bot, error) {
	client := slack.New(
		cfg.SlackBotToken,
		slack.OptionAppLevelToken(cfg.SlackAppToken),
	)

	socketClient := socketmode.New(
		client,
		socketmode.OptionDebug(cfg.LogLevel == "debug"),
	)

	// Get bot user ID for mention detection
	authTest, err := client.AuthTest()
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate with Slack: %w", err)
	}

	return &Bot{
		client:       client,
		socketClient: socketClient,
		handler:      handler,
		botUserID:    authTest.UserID,
		logger:       logger,
	}, nil
}

// Run starts the bot and blocks until the context is cancelled.
func (b *Bot) Run(ctx context.Context) error {
	go b.handleEvents(ctx)

	b.logger.Info("starting Slack bot", "bot_user_id", b.botUserID)
	return b.socketClient.RunContext(ctx)
}

// handleEvents processes incoming Socket Mode events.
func (b *Bot) handleEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case evt := <-b.socketClient.Events:
			b.handleEvent(ctx, evt)
		}
	}
}

// handleEvent routes a single event to the appropriate handler.
func (b *Bot) handleEvent(ctx context.Context, evt socketmode.Event) {
	switch evt.Type {
	case socketmode.EventTypeEventsAPI:
		b.handleEventsAPI(ctx, evt)
	case socketmode.EventTypeSlashCommand:
		b.handleSlashCommand(ctx, evt)
	case socketmode.EventTypeConnecting:
		b.logger.Info("connecting to Slack...")
	case socketmode.EventTypeConnected:
		b.logger.Info("connected to Slack")
	case socketmode.EventTypeConnectionError:
		b.logger.Error("connection error", "error", evt.Data)
	}
}

// handleEventsAPI processes Events API events (mentions, DMs).
func (b *Bot) handleEventsAPI(ctx context.Context, evt socketmode.Event) {
	eventsAPIEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
	if !ok {
		return
	}

	b.socketClient.Ack(*evt.Request)

	switch eventsAPIEvent.Type {
	case slackevents.CallbackEvent:
		b.handleCallbackEvent(ctx, eventsAPIEvent)
	}
}

// handleCallbackEvent processes callback events.
func (b *Bot) handleCallbackEvent(ctx context.Context, evt slackevents.EventsAPIEvent) {
	switch innerEvent := evt.InnerEvent.Data.(type) {
	case *slackevents.AppMentionEvent:
		b.handleAppMention(ctx, innerEvent)
	case *slackevents.MessageEvent:
		b.handleMessageEvent(ctx, innerEvent)
	}
}

// handleAppMention processes @bot mentions.
func (b *Bot) handleAppMention(ctx context.Context, evt *slackevents.AppMentionEvent) {
	// Strip the bot mention from the text
	text := b.stripBotMention(evt.Text)

	msg := &IncomingMessage{
		Text:      text,
		UserID:    evt.User,
		ChannelID: evt.Channel,
		ThreadTS:  evt.ThreadTimeStamp,
		IsDM:      false,
	}

	// Use the event timestamp for threading if no thread exists
	if msg.ThreadTS == "" {
		msg.ThreadTS = evt.TimeStamp
	}

	b.processMessage(ctx, msg)
}

// handleMessageEvent processes direct messages.
func (b *Bot) handleMessageEvent(ctx context.Context, evt *slackevents.MessageEvent) {
	// Ignore bot messages and message changes
	if evt.BotID != "" || evt.SubType != "" {
		return
	}

	// Only handle DMs (channel type "im")
	if evt.ChannelType != "im" {
		return
	}

	msg := &IncomingMessage{
		Text:      evt.Text,
		UserID:    evt.User,
		ChannelID: evt.Channel,
		ThreadTS:  evt.ThreadTimeStamp,
		IsDM:      true,
	}

	// Use the event timestamp for threading if no thread exists
	if msg.ThreadTS == "" {
		msg.ThreadTS = evt.TimeStamp
	}

	b.processMessage(ctx, msg)
}

// handleSlashCommand processes /stormstack-dev commands.
func (b *Bot) handleSlashCommand(ctx context.Context, evt socketmode.Event) {
	cmd, ok := evt.Data.(slack.SlashCommand)
	if !ok {
		return
	}

	b.socketClient.Ack(*evt.Request)

	// Only handle our command
	if cmd.Command != "/stormstack-dev" {
		return
	}

	msg := &IncomingMessage{
		Text:      cmd.Text,
		UserID:    cmd.UserID,
		ChannelID: cmd.ChannelID,
		ThreadTS:  "", // Slash commands don't have threads
		IsDM:      false,
	}

	b.processMessage(ctx, msg)
}

// processMessage sends a message to the handler and posts the response.
func (b *Bot) processMessage(ctx context.Context, msg *IncomingMessage) {
	b.logger.Debug("processing message",
		"user", msg.UserID,
		"channel", msg.ChannelID,
		"text", msg.Text,
	)

	// Show typing indicator
	b.showTyping(msg.ChannelID)

	// Call the handler
	response, err := b.handler(ctx, msg)
	if err != nil {
		b.logger.Error("handler error", "error", err)
		response = &OutgoingMessage{
			Text:     fmt.Sprintf("Sorry, I encountered an error: %v", err),
			ThreadTS: msg.ThreadTS,
		}
	}

	// Send the response
	if err := b.sendMessage(msg.ChannelID, response); err != nil {
		b.logger.Error("failed to send message", "error", err)
	}
}

// sendMessage posts a message to a channel.
func (b *Bot) sendMessage(channelID string, msg *OutgoingMessage) error {
	options := []slack.MsgOption{
		slack.MsgOptionText(msg.Text, false),
	}

	if msg.ThreadTS != "" {
		options = append(options, slack.MsgOptionTS(msg.ThreadTS))
	}

	if len(msg.Blocks) > 0 {
		options = append(options, slack.MsgOptionBlocks(msg.Blocks...))
	}

	_, _, err := b.client.PostMessage(channelID, options...)
	return err
}

// SendMessage allows external callers to send messages (for streaming updates).
func (b *Bot) SendMessage(channelID string, msg *OutgoingMessage) error {
	return b.sendMessage(channelID, msg)
}

// UpdateMessage updates an existing message.
func (b *Bot) UpdateMessage(channelID, timestamp, text string) error {
	_, _, _, err := b.client.UpdateMessage(channelID, timestamp, slack.MsgOptionText(text, false))
	return err
}

// showTyping sends a typing indicator to a channel.
func (b *Bot) showTyping(channelID string) {
	// Note: Slack doesn't have a direct typing indicator API for bots
	// The typing indicator is shown automatically when the bot is processing
}

// stripBotMention removes the bot mention from message text.
func (b *Bot) stripBotMention(text string) string {
	mention := fmt.Sprintf("<@%s>", b.botUserID)
	text = strings.Replace(text, mention, "", 1)
	return strings.TrimSpace(text)
}

// GetBotUserID returns the bot's Slack user ID.
func (b *Bot) GetBotUserID() string {
	return b.botUserID
}
