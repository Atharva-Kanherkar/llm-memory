# Mnemosyne

**Your Personal Memory Assistant** — A cognitive prosthesis that captures and recalls your computer activity.

Mnemosyne continuously captures what you're doing on your computer (windows, screenshots, clipboard, git activity, stress levels) and lets you query it using natural language. Built for people who need help remembering what they were working on.

## Features

- **Window Tracking** — Captures active window titles and applications
- **Screenshot OCR** — Takes periodic screenshots and extracts text using vision AI (pre-computed for instant queries)
- **Clipboard History** — Tracks everything you copy
- **Git Activity** — Monitors your repositories, branches, and commits
- **Stress Detection** — Analyzes mouse jitter, typing patterns, and window switching to detect anxiety
- **External Integrations** — Connect to Gmail, Slack, and Google Calendar for comprehensive memory
- **Natural Language Queries** — Ask questions like "What was I working on this morning?"
- **Streaming Responses** — Real-time AI responses with animated loading
- **Privacy Controls** — Block sensitive apps, URLs, and keywords from capture
- **Encrypted OAuth Storage** — AES-256-GCM encrypted token storage with secure key management

## Installation

### Requirements

- Go 1.21+
- Linux with Hyprland (for window/cursor tracking)
- SQLite3
- OpenRouter API key (for AI features)

### Build

```bash
git clone https://github.com/Atharva-Kanherkar/mnemosyne.git
cd mnemosyne
go build ./cmd/mnemosyne/
```

## Usage

### Start the Daemon

The daemon runs in the background, capturing your activity:

```bash
export OPENROUTER_API_KEY="your-key-here"
./mnemosyne daemon
```

Or run in background:

```bash
./mnemosyne daemon &
```

### Query Your Memory

Start the interactive TUI:

```bash
./mnemosyne query
```

#### Example Queries

```
> What was I working on today?
> Was I stressed while coding?
> What did I copy to clipboard in the last hour?
> Summarize my morning
```

#### Commands

| Command | Description |
|---------|-------------|
| `/stats` | Show capture statistics |
| `/recent [n]` | Show recent captures |
| `/search <text>` | Search captures by text |
| `/summary [today\|hour\|day]` | AI summary of activity |
| `/stress` | Show stress/anxiety patterns |
| `/model [id]` | List or change AI model |
| `/privacy` | View privacy settings |
| `/exclude <app>` | Block an app from capture |
| `/clear [all\|today\|screen]` | Delete captured data |
| `/auth` | Show connected integrations |
| `/connect <provider>` | Connect Gmail, Slack, or Calendar |
| `/logout <provider>` | Disconnect a service |
| `/debug` | Toggle debug logging |
| `/help` | Show help |
| `/quit` | Exit |

### Quick Question

Ask a single question without entering the TUI:

```bash
./mnemosyne ask "What was I doing at 2pm?"
```

## Privacy & Security

Mnemosyne takes privacy seriously:

### Data Storage

- All data stored locally in `~/.local/share/mnemosyne/`
- Directory permissions set to `0700` (owner-only access)
- No data sent to external servers except for AI queries

### Blocked by Default

**Apps** (never captured):
- 1Password, KeePassXC, Bitwarden, LastPass
- GNOME Keyring, Seahorse, Wallet apps

**URL Patterns**:
- `*bank*`, `*banking*`, `*paypal*`, `*venmo*`
- `*password*`, `*login*`, `*signin*`

**Keywords** (filtered from clipboard):
- password, secret, api_key, token, private_key, credential

### Privacy Commands

```bash
# View current privacy settings
/privacy

# Block an app from capture
/exclude slack
/exclude discord

# Delete all captured data
/clear all

# Delete today's data only
/clear today

# Delete all screenshots
/clear screen
```

### Configuration

Create `~/.config/mnemosyne/config.yaml`:

```yaml
capture_interval_seconds: 10
screen_capture_enabled: true
window_capture_enabled: true
git_capture_enabled: true
clipboard_capture_enabled: true

blocked_apps:
  - 1password
  - keepassxc
  - bitwarden
  - slack  # add your own

blocked_urls:
  - "*bank*"
  - "*healthcare*"

blocked_keywords:
  - password
  - secret
  - ssn

llm:
  provider: openrouter
  chat_model: openai/gpt-4o-mini
```

## External Integrations

Mnemosyne can connect to external services to capture more context about your activities.

### Gmail

Captures recent emails and unread count to help you remember communications.

```bash
# Set up Google OAuth
export GOOGLE_CLIENT_ID="your-client-id"
export GOOGLE_CLIENT_SECRET="your-client-secret"

# In the TUI
/connect gmail
```

### Slack

Captures recent messages from channels you're a member of.

```bash
# Set up Slack OAuth
export SLACK_CLIENT_ID="your-client-id"
export SLACK_CLIENT_SECRET="your-client-secret"

# In the TUI
/connect slack
```

### Google Calendar

Captures today's events and upcoming meetings.

```bash
# Uses same Google OAuth credentials as Gmail
/connect calendar
```

### Security Notes

- **All OAuth tokens are encrypted** with AES-256-GCM before storage
- Encryption keys are generated using cryptographically secure random and stored with `0600` permissions
- Tokens are **never logged** anywhere
- OAuth callback server only listens on `127.0.0.1` (localhost)
- CSRF protection via cryptographic state parameter

### Setting Up OAuth Credentials

**Google (Gmail/Calendar):**
1. Go to [Google Cloud Console](https://console.cloud.google.com)
2. Create a new project or select existing
3. Enable Gmail API and Google Calendar API
4. Create OAuth 2.0 credentials (Desktop application)
5. Add `http://localhost:8085/callback` and `http://localhost:8086/callback` as redirect URIs

**Slack:**
1. Go to [Slack API](https://api.slack.com/apps)
2. Create a new app
3. Add OAuth scopes: `channels:history`, `channels:read`, `users:read`
4. Add `http://localhost:8087/callback` as a redirect URL

## Architecture

```
mnemosyne/
├── cmd/mnemosyne/       # Main application
│   ├── main.go          # Entry point
│   └── tui.go           # Terminal UI
├── internal/
│   ├── capture/         # Data capture modules
│   │   ├── window/      # Window tracking (Hyprland)
│   │   ├── screen/      # Screenshot capture + OCR
│   │   ├── clipboard/   # Clipboard monitoring
│   │   ├── git/         # Git repository tracking
│   │   ├── activity/    # Idle detection
│   │   └── biometrics/  # Stress detection
│   ├── daemon/          # Background daemon manager
│   ├── storage/         # SQLite storage
│   ├── query/           # Query engine
│   ├── llm/             # OpenRouter LLM client
│   ├── ocr/             # Vision-based OCR
│   ├── oauth/           # Secure OAuth 2.0 (encrypted tokens)
│   ├── integrations/    # Gmail, Slack, Calendar clients
│   └── config/          # Configuration
```

## Stress Detection

Mnemosyne analyzes behavioral patterns to detect stress:

| Metric | Normal | Stressed |
|--------|--------|----------|
| Mouse jitter | < 0.3 | > 0.3 |
| Typing pauses | < 10 | > 10 |
| Error rate (backspace) | < 15% | > 15% |
| Window switches/min | < 3 | > 3 |
| Rapid switches (<5s) | < 10 | > 10 |

Based on research from CMU and IEEE studies on keystroke dynamics.

## Models

Mnemosyne uses OpenRouter to access various AI models:

**Recommended:**
- `openai/gpt-4o-mini` (default, fast & cheap)
- `anthropic/claude-3.5-haiku` (fast)
- `deepseek/deepseek-chat` (very cheap)

**Best Quality:**
- `anthropic/claude-3.5-sonnet`
- `openai/gpt-4o`

**Free:**
- `qwen/qwen3-coder-next`

Change model in TUI:
```
/model deepseek/deepseek-chat
```

## License

MIT

## Author

Atharva Kanherkar
