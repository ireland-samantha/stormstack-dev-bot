# StormStack Dev Bot

A Slack-based AI pair programming bot powered by Claude Opus. Works with any Git repository - either your local checkout or a sandboxed clone.
I asked Claude to build this so I can chat with it over Slack and mentor it while I'm not physically at my laptop. I run this on my computer and 
chat to it in a private Slack server.

## Features

- **Slack Integration**: Responds to @mentions, DMs, and slash commands
- **Claude Opus 4.5**: Powered by Anthropic's most capable model
- **Code Understanding**: Read, search, and explore any codebase
- **Code Modification**: Write and edit files with surgical precision
- **Build & Test**: Run your project's build and test commands
- **Git Operations**: Create branches, commits, and pull requests
- **Project Intelligence**: Automatically loads project guidelines (CLAUDE.md)

## Quick Start

### 1. Create a Slack App

1. Go to [api.slack.com/apps](https://api.slack.com/apps) and create a new app
2. Enable **Socket Mode** in Settings
3. Add OAuth scopes:
   - `app_mentions:read`
   - `chat:write`
   - `im:history`
   - `im:read`
   - `im:write`
   - `commands`
4. Subscribe to bot events:
   - `app_mention`
   - `message.im`
5. Create a slash command: `/stormstack-dev`
6. Install to your workspace
7. Copy the Bot Token (`xoxb-...`) and App Token (`xapp-...`)

### 2. Configure Environment

```bash
# Required
export STORMSTACK_SLACK_BOT_TOKEN=xoxb-your-bot-token
export STORMSTACK_SLACK_APP_TOKEN=xapp-your-app-token
export STORMSTACK_ANTHROPIC_API_KEY=sk-ant-your-api-key

# Mode: local or sandbox
export STORMSTACK_MODE=local

# For local mode: path to your repo
export STORMSTACK_REPO_PATH=/path/to/your/repo

# For sandbox mode: GitHub repo to clone
# export STORMSTACK_MODE=sandbox
# export STORMSTACK_GITHUB_REPO=github.com/org/repo
# export STORMSTACK_GITHUB_TOKEN=ghp_your-token
# export STORMSTACK_WORKSPACE_PATH=./workspace

# Optional: customize build/test commands
export STORMSTACK_BUILD_CMD="./build.sh build"
export STORMSTACK_TEST_CMD="./build.sh test"

# Optional: custom guidelines file (defaults to CLAUDE.md)
export STORMSTACK_GUIDELINES_FILE="CLAUDE.md"
```

### 3. Run the Bot

```bash
go run main.go
```

Or build and run:

```bash
go build -o stormstack-dev-bot
./stormstack-dev-bot
```

## Usage

### In Slack

**Mention the bot:**
```
@StormStack what files handle authentication?
```

**Direct message:**
```
Show me the README.md file
```

**Slash command:**
```
/stormstack-dev run the tests
```

### Example Interactions

**Explore the codebase:**
```
@StormStack how is the database connection handled?
```

**Make changes:**
```
@StormStack add a validateEmail() method to UserService and write a test for it
```

**Create a PR:**
```
@StormStack create a branch, commit these changes, and open a PR
```

**Debug failures:**
```
@StormStack the UserServiceTest is failing, help me fix it
```

## Architecture

```
stormstack-dev-bot/
├── main.go                    # Entry point
├── internal/
│   ├── config/                # Configuration loading
│   ├── slack/                 # Slack bot and handlers
│   ├── claude/                # Anthropic API client
│   ├── storage/               # Conversation storage
│   ├── repo/                  # Repository access
│   ├── codebase/              # File operations
│   ├── executor/              # Command execution
│   └── git/                   # Git and GitHub operations
└── configs/
    └── default-prompt.md      # Default system prompt
```

## Available Tools

The bot has access to these capabilities:

| Category | Tools |
|----------|-------|
| **Code Understanding** | `read_file`, `list_files`, `search_code`, `get_tree` |
| **Code Modification** | `write_file`, `edit_file` |
| **Build & Test** | `run_command`, `run_build`, `run_tests` |
| **Git Operations** | `git_status`, `git_diff`, `git_log`, `create_branch`, `commit`, `push`, `create_pr` |
| **Project Intelligence** | `get_guidelines`, `find_tests`, `analyze_failures` |

## Security

The bot includes several security measures:

- **Path Sandboxing**: All file operations are confined to the repository
- **Command Allowlist**: Only safe commands can be executed
- **Git Safety**: No force pushes, no direct pushes to main/master
- **Secret Protection**: Sensitive files are never exposed

## Configuration Reference

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `STORMSTACK_MODE` | Yes | `local` | `local` or `sandbox` |
| `STORMSTACK_REPO_PATH` | For local | - | Path to local repository |
| `STORMSTACK_GITHUB_REPO` | For sandbox | - | GitHub repo URL |
| `STORMSTACK_GITHUB_TOKEN` | For sandbox | - | GitHub access token |
| `STORMSTACK_WORKSPACE_PATH` | For sandbox | `./workspace` | Clone destination |
| `STORMSTACK_SLACK_BOT_TOKEN` | Yes | - | Slack bot OAuth token |
| `STORMSTACK_SLACK_APP_TOKEN` | Yes | - | Slack app-level token |
| `STORMSTACK_ANTHROPIC_API_KEY` | Yes | - | Anthropic API key |
| `STORMSTACK_BUILD_CMD` | No | `./build.sh build` | Build command |
| `STORMSTACK_TEST_CMD` | No | `./build.sh test` | Test command |
| `STORMSTACK_GUIDELINES_FILE` | No | `CLAUDE.md` | Project guidelines file |
| `STORMSTACK_LOG_LEVEL` | No | `info` | Log level (info/debug) |

## Development

```bash
# Install dependencies
go mod download

# Run in development
go run main.go

# Build
go build -o stormstack-dev-bot

# Run tests
go test ./...
```

## Troubleshooting

**Bot not responding?**
- Check that Socket Mode is enabled in Slack
- Verify the bot is installed to your workspace
- Check the logs for connection errors

**"Command not allowed" errors?**
- Only allowlisted commands can run
- Check `internal/executor/sandbox.go` for the allowlist

**"Path escapes repository" errors?**
- All paths must be relative to the repository root
- Use relative paths like `src/main.go`, not absolute paths

## License

MIT
