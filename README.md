# Mnemosyne

**Your Personal Memory Assistant** — A cognitive prosthesis that captures and recalls your computer activity.

Mnemosyne continuously captures what you're doing on your computer (windows, screenshots, clipboard, git activity, stress levels) and lets you query it using natural language. Built for people who need help remembering what they were working on.

## Features

- **Window Tracking** — Captures active window titles and applications
- **Screenshot OCR** — Two-stage AI pipeline: vision model extracts text, cheap model compresses to minimal tokens
- **Clipboard History** — Tracks everything you copy
- **Git Activity** — Monitors your repositories, branches, and commits
- **Stress Detection** — Analyzes mouse jitter, typing patterns, and window switching to detect anxiety
- **Proactive Assistant** — Desktop notifications for stress spikes, context reminders, and periodic AI insights (~$2.50/month)
- **Focus Mode** — AI-powered distraction blocker with visual window borders and smart browser tab detection
- **Focus Behavior Tracking** — Stores every window switch, block event, and quit reason to analyze your productivity patterns
- **Focus Widget** — Beautiful floating timer widget with waybar integration, color-coded timer, and animated icons
- **Persistent Memory** — Hierarchical summarization compresses activity into hourly/daily summaries for full-day recall
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
| `/alerts` | View proactive insights |
| `/trigger` | Generate insights now |
| `/mode` | Create a new focus mode (AI conversation) |
| `/modes` | List saved focus modes |
| `/start <name>` | Start a focus session |
| `/stop` | End focus session |
| `/status` | Show current focus mode status |
| `/model [id]` | List or change AI model |
| `/privacy` | View privacy settings |
| `/exclude <app>` | Block an app from capture |
| `/clear [all\|today\|screen]` | Delete captured data |
| `/auth` | Show connected integrations |
| `/connect <provider>` | Connect Gmail, Slack, or Calendar |
| `/logout <provider>` | Disconnect a service |
| `/setup <provider>` | CLI wizard for OAuth setup |
| `/backfill` | Process old screenshots with OCR |
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

**Option 1: Using gcloud CLI (Recommended for Google)**

```bash
# In the TUI, run:
/setup google

# This will:
# - Check/login to gcloud
# - Create or select a GCP project
# - Enable Gmail and Calendar APIs
# - Guide you through credential creation
```

**Option 2: Manual Setup**

**Google (Gmail/Calendar):**
```bash
# 1. Go to console.cloud.google.com/apis/credentials
# 2. Create OAuth client ID (Desktop app)
# 3. Enable Gmail API and Calendar API
# 4. Set environment variables:
export GOOGLE_CLIENT_ID="your-client-id"
export GOOGLE_CLIENT_SECRET="your-client-secret"

# 5. Connect:
/connect gmail
/connect calendar
```

**Slack:**
```bash
# 1. Go to api.slack.com/apps
# 2. Create New App > From scratch
# 3. Add OAuth scopes: channels:history, channels:read, users:read
# 4. Add redirect URL: http://localhost:8087/callback
# 5. Install to workspace
# 6. Set environment variables:
export SLACK_CLIENT_ID="your-client-id"
export SLACK_CLIENT_SECRET="your-client-secret"

# 7. Connect:
/connect slack
```

## Architecture

### High-Level System Design

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              MNEMOSYNE DAEMON                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐│
│  │   Window    │  │   Screen    │  │    Git      │  │     Clipboard       ││
│  │  Capturer   │  │  Capturer   │  │  Capturer   │  │     Capturer        ││
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └──────────┬──────────┘│
│         │                │                │                     │          │
│         └────────────────┴────────────────┴─────────────────────┘          │
│                                   │                                         │
│                          ┌────────▼────────┐                               │
│                          │   Daemon Mgr    │                               │
│                          │   (Orchestrator)│                               │
│                          └────────┬────────┘                               │
│                                   │                                         │
│         ┌─────────────────────────┼─────────────────────────┐              │
│         ▼                         ▼                         ▼              │
│  ┌─────────────┐         ┌─────────────────┐        ┌─────────────┐       │
│  │   Storage   │         │   Summarizer    │        │   Focus     │       │
│  │  (SQLite)   │◄────────│   (Compressor)  │        │  Enforcer   │       │
│  └──────┬──────┘         └────────┬────────┘        └──────┬──────┘       │
│         │                         │                        │               │
└─────────┼─────────────────────────┼────────────────────────┼───────────────┘
          │                         │                        │
          │                         │                        │
┌─────────┼─────────────────────────┼────────────────────────┼───────────────┐
│         │                 QUERY ENGINE                     │               │
│         ▼                         ▼                        ▼               │
│  ┌─────────────┐         ┌─────────────────┐        ┌─────────────┐       │
│  │   Raw       │         │    Summaries    │        │   Focus     │       │
│  │  Captures   │         │ (hourly/daily)  │        │   Status    │       │
│  └──────┬──────┘         └────────┬────────┘        └──────┬──────┘       │
│         │                         │                        │               │
│         └────────────────────────┬┴────────────────────────┘               │
│                          ┌───────▼───────┐                                 │
│                          │  LLM Client   │ ◄─── OpenRouter API             │
│                          │  (Streaming)  │                                 │
│                          └───────┬───────┘                                 │
│                                  │                                         │
│                          ┌───────▼───────┐                                 │
│                          │      TUI      │                                 │
│                          │  (Bubbletea)  │                                 │
│                          └───────────────┘                                 │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Memory Hierarchy (Human-Like)

```
┌────────────────────────────────────────────────────────────────────────────┐
│                           MEMORY ARCHITECTURE                              │
├────────────────────────────────────────────────────────────────────────────┤
│                                                                            │
│  Recent (0-30 min)         Hourly (30 min - 1 day)      Daily (1+ days)   │
│  ┌─────────────────┐       ┌─────────────────────┐      ┌──────────────┐  │
│  │ Raw Captures    │       │ Hourly Summaries    │      │    Daily     │  │
│  │ - Full OCR text │  ───▶ │ - ~50 tokens each   │ ───▶ │   Summaries  │  │
│  │ - Window titles │       │ - Main task/goal    │      │ - ~100 tokens│  │
│  │ - Clipboard     │       │ - Apps used         │      │ - Key events │  │
│  │ - Git changes   │       │ - Notable events    │      │ - Patterns   │  │
│  └─────────────────┘       └─────────────────────┘      └──────────────┘  │
│                                                                            │
│  Query: "last 5 min"       Query: "today"               Query: "this week"│
│  → Uses raw captures       → Uses hourly summaries     → Uses daily sums  │
│                              + recent raw captures                         │
└────────────────────────────────────────────────────────────────────────────┘

Cost: ~$0.01/day for continuous summarization (DeepSeek)
```

### Focus Mode Architecture

```
┌────────────────────────────────────────────────────────────────────────────┐
│                          FOCUS MODE PIPELINE                               │
├────────────────────────────────────────────────────────────────────────────┤
│                                                                            │
│  ┌─────────────┐       ┌─────────────────┐       ┌─────────────────────┐  │
│  │ Window      │       │    Enforcer     │       │     Controller      │  │
│  │ Change      │  ───▶ │  (Check Rules)  │  ───▶ │   (Hyprland IPC)    │  │
│  │ Event       │       │                 │       │                     │  │
│  └─────────────┘       └────────┬────────┘       └──────────┬──────────┘  │
│                                 │                           │             │
│                    ┌────────────┼────────────┐              │             │
│                    ▼            ▼            ▼              ▼             │
│              ┌──────────┐ ┌──────────┐ ┌──────────┐   ┌───────────┐      │
│              │ Allowed  │ │ Blocked  │ │ Browser  │   │  Actions  │      │
│              │   App?   │ │ Pattern? │ │   Tab?   │   │           │      │
│              └────┬─────┘ └────┬─────┘ └────┬─────┘   │ - Green   │      │
│                   │            │            │         │   Border  │      │
│                   │       ┌────▼─────┐      │         │ - Warning │      │
│                   │       │ LLM Ask  │◄─────┘         │ - Close   │      │
│                   │       │"Relevant?"│               │   Tab     │      │
│                   │       └────┬─────┘               │ - Close   │      │
│                   │            │                      │   Window  │      │
│                   └────────────┴──────────────────────┴───────────┘      │
└────────────────────────────────────────────────────────────────────────────┘
```

### Directory Structure

```
mnemosyne/
├── cmd/mnemosyne/         # Main application
│   ├── main.go            # Entry point
│   ├── tui.go             # Terminal UI (Bubbletea)
│   └── widget.go          # Focus mode floating widget
├── internal/
│   ├── capture/           # Data capture modules
│   │   ├── window/        # Window tracking (Hyprland IPC)
│   │   ├── screen/        # Screenshot capture
│   │   ├── clipboard/     # Clipboard monitoring (wl-paste)
│   │   ├── git/           # Git repository tracking
│   │   ├── activity/      # Idle detection (hypridle)
│   │   ├── audio/         # Audio capture (opt-in)
│   │   └── biometrics/    # Stress detection (mouse/keyboard)
│   ├── daemon/            # Background daemon orchestrator
│   ├── focus/             # Focus mode system
│   │   ├── mode.go        # Mode definition & rules
│   │   ├── builder.go     # AI conversation to build modes
│   │   ├── enforcer.go    # Window monitoring & blocking
│   │   ├── controller.go  # Hyprland window control
│   │   └── widget.go      # Widget state broadcaster
│   ├── memory/            # Persistent memory
│   │   └── summarizer.go  # Hourly/daily summarization
│   ├── storage/           # SQLite persistence
│   ├── query/             # Query engine
│   ├── llm/               # OpenRouter LLM client
│   ├── ocr/               # Vision-based OCR (two-stage)
│   ├── insights/          # Proactive assistant
│   ├── oauth/             # Secure OAuth 2.0
│   ├── integrations/      # Gmail, Slack, Calendar
│   └── config/            # Configuration
```

### Two-Stage OCR Pipeline

Screenshots are processed through a two-stage AI pipeline that extracts and compresses text for efficient storage:

```
┌─────────────┐     ┌──────────────────┐     ┌─────────────────┐     ┌──────────────┐
│ Screenshot  │────▶│  GPT-4o-mini     │────▶│  DeepSeek Chat  │────▶│   Storage    │
│   (PNG)     │     │  (Vision Model)  │     │  (Compressor)   │     │   (SQLite)   │
└─────────────┘     └──────────────────┘     └─────────────────┘     └──────────────┘
                           │                         │
                           ▼                         ▼
                    Raw extraction            Compressed output
                    (~500 tokens)             (~50 tokens)
```

**Stage 1: Vision Extraction**
- Model: `openai/gpt-4o-mini` (vision-capable)
- Extracts: app name, window title, user activity, visible text, errors
- Output: ~500 tokens of detailed description

**Stage 2: Text Compression**
- Model: `deepseek/deepseek-chat` ($0.07/M tokens)
- Compresses to 1-2 sentences preserving key information
- Output: ~50 tokens

**Token Efficiency**

| Scenario | Traditional | Two-Stage Pipeline |
|----------|-------------|-------------------|
| 1 screenshot | ~500 tokens | ~50 tokens |
| 100 screenshots/day | ~50,000 tokens | ~5,000 tokens |
| Cost (DeepSeek) | N/A | ~$0.0004/day |

This allows capturing every 10 seconds while staying within context limits for queries.

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

## Proactive Assistant

Mnemosyne can proactively notify you about patterns in your activity:

### Automatic Alerts

| Alert Type | Trigger | Notification |
|------------|---------|--------------|
| Stress Spike | Score jumps >20 points in 2min | Desktop (urgent) |
| Sustained Stress | High stress for 10+ minutes | Desktop (warning) |
| Context Reminder | Return from 5+ min break | TUI (info) |
| Deep Work | Focused on one app 30+ min | TUI (info) |

### Periodic LLM Analysis

Every 30 minutes, a cheap model (DeepSeek) analyzes your activity to detect:
- Work patterns and fragmentation
- Stress correlations with tasks/apps
- Productivity observations

**Cost**: ~$2.50/month for continuous analysis

### Commands

```bash
/alerts     # View recent insights
/trigger    # Generate insights now (manual)
```

### Configuration

```yaml
# ~/.config/mnemosyne/config.yaml
insights:
  enabled: true
  desktop_notifications: true
  batch_interval_minutes: 30
  stress_alerts_enabled: true
  context_reminders: true
  llm_model: deepseek/deepseek-chat
```

## Focus Mode

AI-powered distraction blocker that helps you stay focused by automatically closing distracting apps and browser tabs.

### How It Works

1. **Create a mode** with `/mode` — Have a conversation with the AI about what you're doing
2. **AI builds rules** — Based on your description, it determines allowed/blocked apps and sites
3. **Visual feedback** — Window borders glow green (allowed) or red (blocked)
4. **Smart enforcement** — Distracting windows get a 5-second warning with pulsing red border, then close

### Creating a Focus Mode

```
/mode

AI: What will you be doing in this focus session?
You: Studying algorithms using textbooks, Claude, and VS Code

AI: Which apps do you need?
You: VS Code, Firefox for Claude and docs, Zathura for PDFs

AI: What distractions should I block?
You: YouTube, Reddit, Twitter, Discord

AI: How long will this session be?
You: 2 hours

✓ Focus mode "Study Mode" created!
```

### Visual Indicators

| State | Border Color | Description |
|-------|--------------|-------------|
| Allowed | 🟢 Green | Window is allowed in focus mode |
| Warning | 🟠 Orange | Distraction detected, 5s countdown |
| Blocking | 🔴 Red pulse | About to close |

### Browser Tab Handling

Focus mode is smart about browsers:
- **Closes just the tab** (Ctrl+W), not the whole browser
- **Checks tab titles** against allowed sites and blocked patterns
- **Asks AI** for ambiguous tabs ("Is this aligned with studying algorithms?")
- **Caches decisions** so repeated tabs don't cost extra

### Requirements

```bash
# For closing browser tabs (Wayland)
sudo pacman -S wtype
```

### Commands

```bash
/mode          # Create new focus mode via AI conversation
/modes         # List saved modes
/start <name>  # Start a focus session
/stop          # End session (shows stats, asks why if early)
/status        # Current mode and blocks count
```

### Behavior Tracking

Mnemosyne now tracks your focus behavior to help you understand your productivity patterns:

| Tracked Data | Description |
|--------------|-------------|
| **Window switches** | Every app/tab you switch to during a session |
| **LLM decisions** | What the AI allowed/blocked and why |
| **Block events** | Which windows triggered distractions |
| **Quit reasons** | Why you ended early (if before planned time) |
| **Duration stats** | Planned vs actual time for each session |

Data is stored in `~/.local/share/mnemosyne/mnemosyne.db` in the `focus_session_events` table.

### Early Quit Detection

When you run `/stop` before your planned session time, the LLM asks why you're quitting. This helps you:
- Identify common distraction patterns
- Understand when you're underestimating task time
- Track completion rates per focus mode

The reason is summarized (e.g., "Task completed", "Got distracted", "Emergency") and stored with your session.

### Cost

| Operation | Cost |
|-----------|------|
| Tab check (DeepSeek) | ~$0.001 per unique tab |
| Mode creation | ~$0.01 |
| Heavy daily use | ~$0.10 |

Tab decisions are cached, so visiting the same site multiple times costs nothing extra.

## Persistent Memory

Mnemosyne uses hierarchical summarization to maintain persistent memory without exploding token costs. This mimics how human memory works — recent events are detailed, older events are compressed into gist.

### How It Works

```
Raw Captures ──▶ Hourly Summaries ──▶ Daily Summaries
(every 10s)      (every 30 min)       (once per day)
   │                    │                    │
   │                    │                    │
   ▼                    ▼                    ▼
"User clicked on      "10-11am: Worked    "Tuesday: Deep work
VS Code, opened        on query engine,    on memory system.
file main.go..."       fixed bug in        Morning stress spike
                       parsing. Used       during debugging."
                       VS Code, Firefox."
```

### Query Behavior

| Query | Data Source |
|-------|-------------|
| "What did I just do?" | Raw captures (last 30 min) |
| "How was my morning?" | Hourly summaries + recent raw |
| "What did I do today?" | Hourly summaries + recent raw |
| "What happened yesterday?" | Daily summary |
| "This week's highlights" | Daily summaries |

### Token Efficiency

| Time Span | Traditional | With Summaries |
|-----------|-------------|----------------|
| 1 hour | ~5,000 tokens | ~100 tokens |
| 1 day | ~60,000 tokens | ~1,200 tokens |
| 1 week | ~420,000 tokens | ~2,000 tokens |

**Cost**: ~$0.01/day for continuous summarization using DeepSeek.

## Focus Widget

A beautiful floating widget to track your focus sessions:

```bash
# Interactive terminal widget (full UI)
mnemosyne widget

# One-line output for waybar/polybar
mnemosyne widget line

# JSON output for eww
mnemosyne widget json
```

### Widget Display

```
  ╭────────────────────────────────────────╮
  │ ● FOCUSING                             │
  ├────────────────────────────────────────┤
  │             01:23:45                   │
  │            Study Mode                  │
  ├────────────────────────────────────────┤
  │  Blocked: 3                            │
  │  ✓ Allowed: claude.ai - Claude         │
  ╰────────────────────────────────────────╯

  Ctrl+C to close • /stop to end session
```

### Waybar Integration

Add to your `~/.config/waybar/config.jsonc`:

```json
"custom/mnemosyne": {
    "exec": "mnemosyne widget waybar",
    "interval": 1,
    "return-type": "json",
    "format": "{}",
    "class": "mnemosyne"
}
```

Add to your `~/.config/waybar/style.css`:

```css
/* Minimal rectangular focus widget */
#custom-mnemosyne {
  margin: 0 8px;
  padding: 0 4px;
  transition: all 0.3s ease;
}

#custom-mnemosyne.inactive {
  color: #6c7086;
}

#custom-mnemosyne.focus-active {
  color: #a6e3a1;
  border-bottom: 2px solid #a6e3a1;
}

#custom-mnemosyne.focus-deep {
  color: #f9e2af;
  border-bottom: 2px solid #f9e2af;
}
```

**Features:**
- 🎨 **Color-coded timer**: Green → Yellow → Orange → Red as time passes
- 🔄 **Animated icon**: Changes every 2 seconds while active
- 📊 **Rich tooltip**: Hover to see progress bar, blocks count, mode name
- 🎯 **CSS classes**: `inactive`, `focus-active`, `focus-deep` (30+ min)

### Legacy Waybar (simple text)

If you prefer simple text without colors:

```json
"custom/mnemosyne": {
    "exec": "mnemosyne widget line",
    "interval": 1,
    "format": "{}"
}
```

### EWW Integration

```lisp
(deflisten focus-state :initial "{}"
  `while true; do mnemosyne widget json; sleep 1; done`)

(defwidget focus []
  (box :class "focus-widget"
    (label :text {focus-state.active ?
      "● ${focus-state.mode_name} ${focus-state.elapsed}" :
      "○ Focus Off"})))
```

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
