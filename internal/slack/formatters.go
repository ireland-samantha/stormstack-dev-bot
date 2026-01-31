// Package slack provides Slack message formatting utilities.
package slack

import (
	"fmt"
	"strings"

	"github.com/slack-go/slack"
)

// FormatCodeBlock wraps text in a Slack code block.
func FormatCodeBlock(text string) string {
	return fmt.Sprintf("```\n%s\n```", text)
}

// FormatCodeBlockWithLang wraps text in a Slack code block with language hint.
func FormatCodeBlockWithLang(text, lang string) string {
	return fmt.Sprintf("```%s\n%s\n```", lang, text)
}

// FormatInlineCode wraps text in inline code markers.
func FormatInlineCode(text string) string {
	return fmt.Sprintf("`%s`", text)
}

// FormatBold wraps text in bold markers.
func FormatBold(text string) string {
	return fmt.Sprintf("*%s*", text)
}

// FormatItalic wraps text in italic markers.
func FormatItalic(text string) string {
	return fmt.Sprintf("_%s_", text)
}

// FormatLink creates a Slack link.
func FormatLink(url, text string) string {
	return fmt.Sprintf("<%s|%s>", url, text)
}

// FormatUserMention creates a user mention.
func FormatUserMention(userID string) string {
	return fmt.Sprintf("<@%s>", userID)
}

// FormatChannelMention creates a channel mention.
func FormatChannelMention(channelID string) string {
	return fmt.Sprintf("<#%s>", channelID)
}

// TruncateText truncates text to a maximum length with ellipsis.
func TruncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	if maxLen <= 3 {
		return text[:maxLen]
	}
	return text[:maxLen-3] + "..."
}

// BuildHeaderBlock creates a header block.
func BuildHeaderBlock(text string) *slack.HeaderBlock {
	return slack.NewHeaderBlock(
		slack.NewTextBlockObject(slack.PlainTextType, text, false, false),
	)
}

// BuildSectionBlock creates a section block with markdown text.
func BuildSectionBlock(text string) *slack.SectionBlock {
	return slack.NewSectionBlock(
		slack.NewTextBlockObject(slack.MarkdownType, text, false, false),
		nil, nil,
	)
}

// BuildDividerBlock creates a divider block.
func BuildDividerBlock() *slack.DividerBlock {
	return slack.NewDividerBlock()
}

// BuildContextBlock creates a context block with text elements.
func BuildContextBlock(texts ...string) *slack.ContextBlock {
	elements := make([]slack.MixedElement, len(texts))
	for i, text := range texts {
		elements[i] = slack.NewTextBlockObject(slack.MarkdownType, text, false, false)
	}
	return slack.NewContextBlock("", elements...)
}

// FormatError formats an error message for display.
func FormatError(err error) string {
	return fmt.Sprintf(":x: *Error:* %s", err.Error())
}

// FormatSuccess formats a success message.
func FormatSuccess(msg string) string {
	return fmt.Sprintf(":white_check_mark: %s", msg)
}

// FormatWarning formats a warning message.
func FormatWarning(msg string) string {
	return fmt.Sprintf(":warning: %s", msg)
}

// FormatInfo formats an info message.
func FormatInfo(msg string) string {
	return fmt.Sprintf(":information_source: %s", msg)
}

// FormatProgress formats a progress message.
func FormatProgress(msg string) string {
	return fmt.Sprintf(":hourglass_flowing_sand: %s", msg)
}

// FormatFileContent formats file content for display.
func FormatFileContent(path, content, lang string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("*%s*\n", path))
	if lang != "" {
		sb.WriteString(FormatCodeBlockWithLang(content, lang))
	} else {
		sb.WriteString(FormatCodeBlock(content))
	}
	return sb.String()
}

// DetectLanguage attempts to detect the language from a file extension.
func DetectLanguage(path string) string {
	path = strings.ToLower(path)
	switch {
	case strings.HasSuffix(path, ".go"):
		return "go"
	case strings.HasSuffix(path, ".java"):
		return "java"
	case strings.HasSuffix(path, ".js"):
		return "javascript"
	case strings.HasSuffix(path, ".ts"):
		return "typescript"
	case strings.HasSuffix(path, ".tsx"):
		return "typescript"
	case strings.HasSuffix(path, ".jsx"):
		return "javascript"
	case strings.HasSuffix(path, ".py"):
		return "python"
	case strings.HasSuffix(path, ".rb"):
		return "ruby"
	case strings.HasSuffix(path, ".rs"):
		return "rust"
	case strings.HasSuffix(path, ".sh"), strings.HasSuffix(path, ".bash"):
		return "bash"
	case strings.HasSuffix(path, ".yaml"), strings.HasSuffix(path, ".yml"):
		return "yaml"
	case strings.HasSuffix(path, ".json"):
		return "json"
	case strings.HasSuffix(path, ".xml"):
		return "xml"
	case strings.HasSuffix(path, ".html"):
		return "html"
	case strings.HasSuffix(path, ".css"):
		return "css"
	case strings.HasSuffix(path, ".sql"):
		return "sql"
	case strings.HasSuffix(path, ".md"):
		return "markdown"
	case strings.HasSuffix(path, ".toml"):
		return "toml"
	case strings.HasSuffix(path, ".c"):
		return "c"
	case strings.HasSuffix(path, ".cpp"), strings.HasSuffix(path, ".cc"):
		return "cpp"
	case strings.HasSuffix(path, ".h"), strings.HasSuffix(path, ".hpp"):
		return "cpp"
	default:
		return ""
	}
}
