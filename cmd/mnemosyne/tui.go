package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Atharva-Kanherkar/mnemosyne/internal/config"
	"github.com/Atharva-Kanherkar/mnemosyne/internal/focus"
	"github.com/Atharva-Kanherkar/mnemosyne/internal/integrations"
	"github.com/Atharva-Kanherkar/mnemosyne/internal/llm"
	"github.com/Atharva-Kanherkar/mnemosyne/internal/notify"
	"github.com/Atharva-Kanherkar/mnemosyne/internal/oauth"
	"github.com/Atharva-Kanherkar/mnemosyne/internal/ocr"
	"github.com/Atharva-Kanherkar/mnemosyne/internal/query"
	"github.com/Atharva-Kanherkar/mnemosyne/internal/storage"
)

// ANSI colors for easy access
const (
	reset      = "\033[0m"
	bold       = "\033[1m"
	dim        = "\033[2m"
	cyan       = "\033[36m"
	green      = "\033[32m"
	yellow     = "\033[33m"
	red        = "\033[31m"
	blue       = "\033[34m"
	magenta    = "\033[35m"
	brightCyan = "\033[96m"
)

// Spinner frames for loading animation
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Loading phrases that cycle during processing
var loadingPhrases = []string{
	"Reading your memories",
	"Scanning screenshots",
	"Processing images",
	"Extracting text",
	"Analyzing patterns",
	"Connecting the dots",
	"Building context",
	"Recalling moments",
	"Piecing together",
	"Understanding activity",
	"Examining captures",
	"Parsing clipboard",
	"Reviewing windows",
	"Checking git history",
	"Measuring stress levels",
	"Reconstructing timeline",
	"Gathering insights",
	"Synthesizing data",
	"Almost there",
	"Thinking deeply",
}

// Spinner represents an animated loading spinner
type Spinner struct {
	mu       sync.Mutex
	running  bool
	stopChan chan struct{}
	message  string
}

// NewSpinner creates a new spinner
func NewSpinner() *Spinner {
	return &Spinner{
		stopChan: make(chan struct{}),
	}
}

// Start begins the spinner animation
func (s *Spinner) Start(initialMessage string) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.message = initialMessage
	s.stopChan = make(chan struct{})
	s.mu.Unlock()

	go s.animate()
}

// SetMessage updates the spinner message
func (s *Spinner) SetMessage(msg string) {
	s.mu.Lock()
	s.message = msg
	s.mu.Unlock()
}

// Stop stops the spinner and clears the line
func (s *Spinner) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	close(s.stopChan)
	s.mu.Unlock()

	// Clear the spinner line
	fmt.Print("\r" + strings.Repeat(" ", 70) + "\r")
}

// animate runs the spinner animation
func (s *Spinner) animate() {
	frameIdx := 0
	phraseIdx := rand.Intn(len(loadingPhrases))
	lastPhraseChange := time.Now()
	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.mu.Lock()
			msg := s.message
			s.mu.Unlock()

			// Change phrase every 2 seconds
			if time.Since(lastPhraseChange) > 2*time.Second {
				phraseIdx = (phraseIdx + 1) % len(loadingPhrases)
				lastPhraseChange = time.Now()
			}

			frame := spinnerFrames[frameIdx%len(spinnerFrames)]
			phrase := loadingPhrases[phraseIdx]

			// Build the display line with colors
			line := fmt.Sprintf("\r%s%s%s %s%s%s %s...%s",
				cyan, frame, reset,
				dim, phrase, reset,
				dim, reset)

			if msg != "" {
				line = fmt.Sprintf("\r%s%s%s %s%s%s %s— %s%s",
					cyan, frame, reset,
					dim, phrase, reset,
					dim, msg, reset)
			}

			fmt.Print(line)
			frameIdx++
		}
	}
}

// TUI provides an interactive terminal interface for querying captures.
type TUI struct {
	engine       *query.Engine
	reader       *bufio.Reader
	llmClient    *llm.Client
	apiKey       string
	db           *sql.DB
	cfg          *config.Config
	debug        bool
	integrations *integrations.Manager
	store        *storage.Store
	socketClient *notify.SocketClient
	pendingAlert *storage.InsightRecord
	alertMu      sync.Mutex

	// Focus mode
	focusBuilder      *focus.Builder
	inFocusChat       bool
	activeSessionID   int64        // Currently active focus session (for heartbeat)
	heartbeatStop     chan struct{} // Stop signal for heartbeat goroutine
	heartbeatStopOnce sync.Once
}

// toggleDebug enables or disables debug logging.
func (t *TUI) toggleDebug() {
	t.debug = !t.debug
	if t.llmClient != nil {
		t.llmClient.Debug = t.debug
	}
	t.engine.Debug = t.debug

	if t.debug {
		fmt.Println(yellow + "Debug mode: ON" + reset)
	} else {
		fmt.Println(dim + "Debug mode: OFF" + reset)
	}
}

// NewTUI creates a new TUI instance.
func NewTUI(db *sql.DB, apiKey string) *TUI {
	var llmClient *llm.Client
	if apiKey != "" {
		llmClient = llm.NewClient(apiKey)
	}

	cfg, _ := config.Load()

	// Initialize integrations manager
	homeDir, _ := os.UserHomeDir()
	integrationsDir := filepath.Join(homeDir, ".local", "share", "mnemosyne")
	intMgr, _ := integrations.NewManager(integrationsDir)

	// Initialize storage for insights
	store, _ := storage.New(integrationsDir)

	// Try to connect to daemon's socket for real-time alerts
	socketPath := filepath.Join(integrationsDir, "mnemosyne.sock")
	socketClient := notify.NewSocketClient()
	socketClient.Connect(socketPath) // Best effort, may fail if daemon not running

	return &TUI{
		engine:       query.NewWithOCR(db, llmClient, apiKey),
		reader:       bufio.NewReader(os.Stdin),
		llmClient:    llmClient,
		apiKey:       apiKey,
		db:           db,
		cfg:          cfg,
		integrations: intMgr,
		store:        store,
		socketClient: socketClient,
	}
}

// Run starts the interactive TUI loop.
func (t *TUI) Run(ctx context.Context) error {
	t.printWelcome()

	for {
		t.printPrompt()
		input, err := t.reader.ReadString('\n')
		if err != nil {
			return err
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		// Handle commands
		if strings.HasPrefix(input, "/") {
			if t.handleCommand(ctx, input) {
				return nil // /quit was called
			}
			continue
		}

		// Natural language query
		t.handleQuery(ctx, input)
	}
}

// printPrompt prints the styled input prompt.
func (t *TUI) printPrompt() {
	fmt.Println()
	// Top border with label
	fmt.Print(magenta + "╭─" + reset + bold + " mnemosyne " + reset + magenta + strings.Repeat("─", 46) + "╮" + reset + "\n")
	// Input line with prompt
	fmt.Print(magenta + "│" + reset + " " + brightCyan + "❯" + reset + " ")
}

// printPromptClose closes the input box after receiving input (optional).
func (t *TUI) printPromptClose() {
	fmt.Println(magenta + "╰" + strings.Repeat("─", 58) + "╯" + reset)
}

// printWelcome prints the welcome message.
func (t *TUI) printWelcome() {
	fmt.Println()
	fmt.Println(magenta + "╭" + strings.Repeat("─", 58) + "╮" + reset)
	fmt.Println(magenta + "│" + reset + "                                                          " + magenta + "│" + reset)
	fmt.Println(magenta + "│" + reset + bold + brightCyan + "       ✦ MNEMOSYNE" + reset + dim + " — Your Memory Assistant" + reset + "           " + magenta + "│" + reset)
	fmt.Println(magenta + "│" + reset + "                                                          " + magenta + "│" + reset)
	fmt.Println(magenta + "╰" + strings.Repeat("─", 58) + "╯" + reset)

	fmt.Println()
	fmt.Println(bold + cyan + "  Ask me anything:" + reset)
	fmt.Println(dim + "    \"What was I working on today?\"" + reset)
	fmt.Println(dim + "    \"Was I stressed while coding?\"" + reset)
	fmt.Println(dim + "    \"What did I copy to clipboard?\"" + reset)

	fmt.Println()
	fmt.Println(bold + cyan + "  Commands:" + reset)
	fmt.Println(green + "    /stats" + reset + dim + "     capture statistics" + reset)
	fmt.Println(green + "    /recent" + reset + dim + "    recent captures" + reset)
	fmt.Println(green + "    /search" + reset + dim + "    search by text" + reset)
	fmt.Println(yellow + "    /summary" + reset + dim + "   AI summary of activity" + reset)
	fmt.Println(yellow + "    /stress" + reset + dim + "    stress/anxiety patterns" + reset)
	fmt.Println(yellow + "    /alerts" + reset + dim + "    proactive insights" + reset)
	fmt.Println(yellow + "    /trigger" + reset + dim + "   generate insights now" + reset)
	fmt.Println(blue + "    /model" + reset + dim + "     list or change AI model" + reset)
	fmt.Println()
	fmt.Println(bold + cyan + "  Integrations:" + reset)
	fmt.Println(blue + "    /auth" + reset + dim + "      show connected services" + reset)
	fmt.Println(blue + "    /connect" + reset + dim + "   connect Gmail, Slack, Calendar" + reset)
	fmt.Println(blue + "    /setup" + reset + dim + "     setup OAuth credentials (CLI)" + reset)
	fmt.Println(blue + "    /logout" + reset + dim + "    disconnect a service" + reset)
	fmt.Println()
	fmt.Println(bold + cyan + "  Focus Mode:" + reset)
	fmt.Println(green + "    /mode" + reset + dim + "      create a new focus mode" + reset)
	fmt.Println(green + "    /modes" + reset + dim + "     list saved focus modes" + reset)
	fmt.Println(green + "    /start" + reset + dim + "     start a focus session" + reset)
	fmt.Println(green + "    /stop" + reset + dim + "      stop focus mode" + reset)
	fmt.Println()
	fmt.Println(bold + cyan + "  Privacy:" + reset)
	fmt.Println(red + "    /privacy" + reset + dim + "   view privacy settings" + reset)
	fmt.Println(red + "    /exclude" + reset + dim + "   block an app from capture" + reset)
	fmt.Println(red + "    /clear" + reset + dim + "     delete captured data" + reset)
	fmt.Println()
	fmt.Println(magenta + "    /debug" + reset + dim + "     toggle debug logging" + reset)
	fmt.Println(dim + "    /help      show this help" + reset)
	fmt.Println(red + "    /quit" + reset + dim + "      exit" + reset)

	fmt.Println()
	if t.llmClient != nil {
		fmt.Printf(dim+"  Model: "+reset+cyan+"%s"+reset+"\n", t.llmClient.ChatModel)
	}
	fmt.Println(dim + "  " + strings.Repeat("─", 56) + reset)
}

// handleCommand handles slash commands. Returns true if should exit.
func (t *TUI) handleCommand(ctx context.Context, input string) bool {
	// Close the input box
	fmt.Println()
	fmt.Println(magenta + "╰" + strings.Repeat("─", 58) + "╯" + reset)

	parts := strings.Fields(input)
	cmd := parts[0]
	args := parts[1:]

	switch cmd {
	case "/quit", "/exit", "/q":
		fmt.Println("\n" + dim + "Goodbye!" + reset)
		return true

	case "/help", "/h":
		t.printWelcome()

	case "/stats":
		t.showStats(ctx)

	case "/recent":
		limit := 10
		if len(args) > 0 {
			fmt.Sscanf(args[0], "%d", &limit)
		}
		t.showRecent(ctx, limit)

	case "/search":
		if len(args) == 0 {
			fmt.Println("Usage: /search <text>")
			return false
		}
		searchText := strings.Join(args, " ")
		t.searchCaptures(ctx, searchText)

	case "/summary":
		duration := time.Hour
		if len(args) > 0 {
			switch args[0] {
			case "today":
				now := time.Now()
				start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
				duration = time.Since(start)
			case "hour":
				duration = time.Hour
			case "day":
				duration = 24 * time.Hour
			}
		}
		t.showSummary(ctx, duration)

	case "/window", "/windows":
		t.showBySource(ctx, "window", 10)

	case "/clipboard":
		t.showBySource(ctx, "clipboard", 10)

	case "/git":
		t.showBySource(ctx, "git", 10)

	case "/screen":
		t.showBySource(ctx, "screen", 5)

	case "/stress", "/anxiety", "/biometrics":
		t.showStress(ctx)

	case "/model", "/models":
		if len(args) == 0 {
			t.showModels(ctx)
		} else {
			t.setModel(args[0])
		}

	case "/debug":
		t.toggleDebug()

	case "/privacy":
		t.showPrivacy()

	case "/clear":
		t.clearData(ctx, args)

	case "/exclude":
		if len(args) == 0 {
			t.showExcluded()
		} else {
			t.excludeApp(args[0])
		}

	case "/auth", "/integrations":
		t.showIntegrations()

	case "/connect":
		if len(args) == 0 {
			t.showConnectHelp()
		} else {
			t.connectProvider(ctx, args[0])
		}

	case "/logout", "/disconnect":
		if len(args) == 0 {
			fmt.Println(yellow + "Usage: /logout <provider>" + reset)
			fmt.Println(dim + "Providers: gmail, slack, calendar" + reset)
		} else {
			t.logoutProvider(args[0])
		}

	case "/setup":
		if len(args) == 0 {
			t.showSetupHelp()
		} else {
			t.runSetup(ctx, args[0])
		}

	case "/backfill":
		t.backfillOCR(ctx)

	case "/alerts", "/insights":
		t.showAlerts(ctx)

	case "/trigger":
		t.triggerInsights(ctx)

	case "/mode", "/focus":
		t.startFocusMode(ctx)

	case "/modes":
		t.listFocusModes(ctx)

	case "/start":
		if len(args) == 0 {
			fmt.Println(yellow + "Usage: /start <mode-name-or-id>" + reset)
			fmt.Println(dim + "Use /modes to see available modes" + reset)
		} else {
			t.startFocusSession(ctx, strings.Join(args, " "))
		}

	case "/stop":
		t.stopFocusSession(ctx)

	case "/status":
		t.showFocusStatus(ctx)

	default:
		fmt.Printf(red+"Unknown command: %s\n"+reset, cmd)
		fmt.Println("Type " + cyan + "/help" + reset + " for available commands")
	}

	return false
}

// handleQuery processes a natural language query with streaming.
func (t *TUI) handleQuery(ctx context.Context, question string) {
	// Close the input box
	fmt.Println()
	fmt.Println(magenta + "╰" + strings.Repeat("─", 58) + "╯" + reset)
	fmt.Println()

	// Use a generous timeout for queries with vision OCR
	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	// Start the animated spinner
	spinner := NewSpinner()
	spinner.Start("")

	// Track if we've received the first chunk (means context is built)
	firstChunk := true

	// Stream the response token by token
	_, err := t.engine.AskStream(queryCtx, question, func(chunk string) {
		if firstChunk {
			// Stop spinner and show response box when first chunk arrives
			spinner.Stop()
			fmt.Println()
			fmt.Println(green + "╭─" + reset + bold + " response " + reset + green + strings.Repeat("─", 47) + "╮" + reset)
			fmt.Print(green + "│" + reset + " ")
			firstChunk = false
		}
		// Print each chunk immediately, handle newlines for box formatting
		formatted := strings.ReplaceAll(chunk, "\n", "\n"+green+"│"+reset+" ")
		fmt.Print(formatted)
	})

	// Make sure spinner is stopped in case of error before first chunk
	spinner.Stop()

	if err != nil {
		if queryCtx.Err() == context.DeadlineExceeded {
			fmt.Println("\n" + red + "Query timed out." + reset)
		} else {
			fmt.Printf("\n"+red+"Error: %v"+reset+"\n", err)
		}
		return
	}

	// Close the response box (only if we got chunks)
	if !firstChunk {
		fmt.Println()
		fmt.Println(green + "╰" + strings.Repeat("─", 58) + "╯" + reset)
	}
}

// showStats displays capture statistics.
func (t *TUI) showStats(ctx context.Context) {
	stats, err := t.engine.Stats(ctx)
	if err != nil {
		fmt.Printf(red+"Error getting stats: %v\n"+reset, err)
		return
	}

	fmt.Println()
	fmt.Println(blue + "╭─" + reset + bold + " statistics " + reset + blue + strings.Repeat("─", 45) + "╮" + reset)
	fmt.Printf(blue+"│"+reset+" "+bold+"Total captures:"+reset+" %s%d%s\n", cyan, stats["total_captures"], reset)
	fmt.Println(blue + "│" + reset)

	if bySource, ok := stats["by_source"].(map[string]int64); ok {
		fmt.Println(blue + "│" + reset + " " + bold + "By source:" + reset)
		sourceColors := map[string]string{
			"window":     cyan,
			"screen":     blue,
			"clipboard":  yellow,
			"git":        magenta,
			"activity":   dim,
			"biometrics": red,
		}
		for source, count := range bySource {
			color := sourceColors[source]
			if color == "" {
				color = dim
			}
			fmt.Printf(blue+"│"+reset+"   %s%-10s%s %s%d%s\n", color, source, reset, dim, count, reset)
		}
	}

	fmt.Println(blue + "│" + reset)
	if oldest, ok := stats["oldest_capture"].(time.Time); ok && !oldest.IsZero() {
		fmt.Printf(blue+"│"+reset+" "+dim+"Oldest: %s%s\n", oldest.Format("2006-01-02 15:04"), reset)
	}
	if newest, ok := stats["newest_capture"].(time.Time); ok && !newest.IsZero() {
		fmt.Printf(blue+"│"+reset+" "+dim+"Newest: %s%s\n", newest.Format("2006-01-02 15:04"), reset)
	}
	fmt.Println(blue + "╰" + strings.Repeat("─", 58) + "╯" + reset)
}

// showRecent displays recent captures.
func (t *TUI) showRecent(ctx context.Context, limit int) {
	records, err := t.engine.GetRecent(ctx, limit)
	if err != nil {
		fmt.Printf(red+"Error: %v\n"+reset, err)
		return
	}

	label := fmt.Sprintf(" recent %d captures ", len(records))
	padding := 56 - len(label)
	fmt.Println()
	fmt.Println(cyan + "╭─" + reset + bold + label + reset + cyan + strings.Repeat("─", padding) + "╮" + reset)
	for _, r := range records {
		fmt.Print(cyan + "│" + reset + " ")
		t.printRecord(r)
	}
	fmt.Println(cyan + "╰" + strings.Repeat("─", 58) + "╯" + reset)
}

// showBySource displays captures from a specific source.
func (t *TUI) showBySource(ctx context.Context, source string, limit int) {
	records, err := t.engine.GetBySource(ctx, source, limit)
	if err != nil {
		fmt.Printf(red+"Error: %v\n"+reset, err)
		return
	}

	sourceColors := map[string]string{
		"window":     cyan,
		"screen":     blue,
		"clipboard":  yellow,
		"git":        magenta,
		"activity":   dim,
		"biometrics": red,
	}
	color := sourceColors[source]
	if color == "" {
		color = cyan
	}

	label := fmt.Sprintf(" %s captures ", source)
	padding := 56 - len(label)
	fmt.Println()
	fmt.Println(color + "╭─" + reset + bold + label + reset + color + strings.Repeat("─", padding) + "╮" + reset)
	for _, r := range records {
		fmt.Print(color + "│" + reset + " ")
		t.printRecord(r)
	}
	fmt.Println(color + "╰" + strings.Repeat("─", 58) + "╯" + reset)
}

// searchCaptures searches and displays matching captures.
func (t *TUI) searchCaptures(ctx context.Context, searchText string) {
	records, err := t.engine.SearchText(ctx, searchText, 20)
	if err != nil {
		fmt.Printf(red+"Error: %v\n"+reset, err)
		return
	}

	if len(records) == 0 {
		fmt.Println(dim + "No matches found." + reset)
		return
	}

	label := fmt.Sprintf(" %d matches for '%s' ", len(records), searchText)
	if len(label) > 54 {
		label = label[:51] + "... "
	}
	padding := 56 - len(label)
	if padding < 0 {
		padding = 0
	}
	fmt.Println()
	fmt.Println(yellow + "╭─" + reset + bold + label + reset + yellow + strings.Repeat("─", padding) + "╮" + reset)
	for _, r := range records {
		fmt.Print(yellow + "│" + reset + " ")
		t.printRecord(r)
	}
	fmt.Println(yellow + "╰" + strings.Repeat("─", 58) + "╯" + reset)
}

// showSummary displays an LLM-generated summary with streaming.
func (t *TUI) showSummary(ctx context.Context, duration time.Duration) {
	fmt.Println()

	queryCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	// Start spinner
	spinner := NewSpinner()
	spinner.Start("")

	firstChunk := true
	label := fmt.Sprintf(" summary (last %s) ", duration)
	padding := 56 - len(label)

	// Stream the summary
	_, err := t.engine.SummarizeStream(queryCtx, duration, func(chunk string) {
		if firstChunk {
			spinner.Stop()
			fmt.Println()
			fmt.Println(yellow + "╭─" + reset + bold + label + reset + yellow + strings.Repeat("─", padding) + "╮" + reset)
			fmt.Print(yellow + "│" + reset + " ")
			firstChunk = false
		}
		formatted := strings.ReplaceAll(chunk, "\n", "\n"+yellow+"│"+reset+" ")
		fmt.Print(formatted)
	})

	spinner.Stop()

	if err != nil {
		fmt.Printf("\n"+red+"Error: %v\n"+reset, err)
		return
	}

	if !firstChunk {
		fmt.Println()
		fmt.Println(yellow + "╰" + strings.Repeat("─", 58) + "╯" + reset)
	}
}

// showStress displays stress/anxiety analysis.
func (t *TUI) showStress(ctx context.Context) {
	records, err := t.engine.GetBySource(ctx, "biometrics", 20)
	if err != nil {
		fmt.Printf(red+"Error: %v\n"+reset, err)
		return
	}

	if len(records) == 0 {
		fmt.Println("\n" + yellow + "No biometrics data captured yet." + reset)
		fmt.Println(dim + "Run the daemon to start tracking stress patterns." + reset)
		return
	}

	fmt.Println("\n" + cyan + "━━━ Stress/Anxiety Analysis ━━━" + reset)
	fmt.Println(dim + "Based on: mouse jitter, typing pauses, window switching patterns" + reset)
	fmt.Println()

	// Show stress timeline
	fmt.Println(bold + "Recent stress levels:" + reset)
	for _, r := range records {
		timestamp := r.Timestamp.Format("15:04:05")
		level := r.Metadata["stress_level"]
		score := r.Metadata["stress_score"]

		// Color-code based on level
		var levelColor, indicator string
		switch level {
		case "calm":
			levelColor = green
			indicator = green + "●" + reset
		case "normal":
			levelColor = blue
			indicator = blue + "●" + reset
		case "elevated":
			levelColor = yellow
			indicator = yellow + "●" + reset
		case "high":
			levelColor = red
			indicator = red + "●" + reset
		case "anxious":
			levelColor = red + bold
			indicator = red + bold + "◉" + reset
		default:
			levelColor = dim
			indicator = dim + "○" + reset
		}

		fmt.Printf("  "+dim+"[%s]"+reset+" %s %s%s"+reset+" "+dim+"(score: %s)"+reset+"\n",
			timestamp, indicator, levelColor, level, score)

		// Show indicators if present
		if r.TextData != "" {
			fmt.Printf("           "+dim+"%s"+reset+"\n", r.TextData)
		}
	}

	// Summarize patterns
	fmt.Println("\nKey metrics from latest capture:")
	if len(records) > 0 {
		latest := records[0]
		if jitter := latest.Metadata["mouse_jitter"]; jitter != "" {
			fmt.Printf("  Mouse jitter:      %s (>0.3 = stressed)\n", jitter)
		}
		if pauses := latest.Metadata["typing_pauses"]; pauses != "" {
			fmt.Printf("  Typing pauses:     %s (>10 = stressed)\n", pauses)
		}
		if switches := latest.Metadata["window_switches_pm"]; switches != "" {
			fmt.Printf("  Window switches/m: %s (>3 = fragmented)\n", switches)
		}
		if rapid := latest.Metadata["rapid_switches"]; rapid != "" {
			fmt.Printf("  Rapid switches:    %s (>10 = anxious)\n", rapid)
		}
	}
}

// showModels displays available models.
func (t *TUI) showModels(ctx context.Context) {
	fmt.Println()

	// Spinner while fetching
	spinner := NewSpinner()
	spinner.Start("fetching model catalog")

	models, err := llm.FetchModels(ctx)
	spinner.Stop()

	if err != nil {
		fmt.Printf(red+"Error fetching models: %v\n"+reset, err)
		fmt.Println("\n" + bold + "Recommended models:" + reset)
		for _, id := range llm.RecommendedModels {
			fmt.Printf("  %s%s%s\n", cyan, id, reset)
		}
		return
	}

	// Current model
	if t.llmClient != nil {
		fmt.Printf("\n"+green+"Current model: %s%s\n", t.llmClient.ChatModel, reset)
	}

	// Show recommended models
	fmt.Println("\n" + bold + "⭐ Recommended:" + reset)
	for _, recID := range llm.RecommendedModels[:10] {
		for _, m := range models {
			if m.ID == recID {
				ctx := formatCtx(m.ContextLength)
				price := formatPriceShort(m.Pricing)
				fmt.Printf("  %s%-40s%s %s%s%s %s\n", cyan, m.ID, reset, dim, ctx, reset, price)
				break
			}
		}
	}

	// Show free models
	fmt.Println("\n" + bold + "🆓 Free models:" + reset)
	free := llm.GetFreeModels(models)
	count := 0
	for _, m := range free {
		if count >= 8 {
			break
		}
		ctx := formatCtx(m.ContextLength)
		fmt.Printf("  %s%-40s%s %s%s%s %s\n", green, m.ID, reset, dim, ctx, reset, green+"FREE"+reset)
		count++
	}

	// Chinese/Asian models
	fmt.Println("\n" + bold + "🇨🇳 Chinese & Asian models:" + reset)
	providers := []string{"qwen", "deepseek", "zhipu", "baichuan", "minimax", "moonshotai", "z-ai", "01-ai"}
	count = 0
	for _, m := range models {
		if count >= 8 {
			break
		}
		for _, p := range providers {
			if strings.HasPrefix(m.ID, p+"/") {
				ctx := formatCtx(m.ContextLength)
				price := formatPriceShort(m.Pricing)
				fmt.Printf("  %s%-40s%s %s%s%s %s\n", magenta, m.ID, reset, dim, ctx, reset, price)
				count++
				break
			}
		}
	}

	fmt.Println("\n" + dim + "Use /model <model-id> to switch models" + reset)
	fmt.Println(dim + "Use /model search <query> to search all models" + reset)
}

// setModel changes the current model.
func (t *TUI) setModel(modelID string) {
	// Handle search
	if modelID == "search" {
		fmt.Print("Search query: ")
		query, _ := t.reader.ReadString('\n')
		query = strings.TrimSpace(query)
		t.searchModels(context.Background(), query)
		return
	}

	if t.llmClient == nil {
		fmt.Println(red + "No API key configured" + reset)
		return
	}

	// Set the model
	t.llmClient.ChatModel = modelID

	// Rebuild engine with new model
	t.engine = query.NewWithOCR(t.db, t.llmClient, t.apiKey)

	fmt.Printf(green+"✓ Model changed to: %s%s\n", modelID, reset)
}

// searchModels searches for models matching a query.
func (t *TUI) searchModels(ctx context.Context, query string) {
	models, err := llm.FetchModels(ctx)
	if err != nil {
		fmt.Printf(red+"Error: %v\n"+reset, err)
		return
	}

	filtered := llm.FilterModels(models, query)
	if len(filtered) == 0 {
		fmt.Println("No models found matching: " + query)
		return
	}

	fmt.Printf("\n"+bold+"Found %d models matching '%s':"+reset+"\n", len(filtered), query)
	for i, m := range filtered {
		if i >= 20 {
			fmt.Printf(dim+"  ... and %d more"+reset+"\n", len(filtered)-20)
			break
		}
		ctx := formatCtx(m.ContextLength)
		price := formatPriceShort(m.Pricing)
		fmt.Printf("  %s%-40s%s %s%s%s %s\n", cyan, m.ID, reset, dim, ctx, reset, price)
	}
}

func formatCtx(ctx int) string {
	if ctx >= 1000000 {
		return fmt.Sprintf("%.1fM ctx", float64(ctx)/1000000)
	}
	return fmt.Sprintf("%dk ctx", ctx/1000)
}

func formatPriceShort(p llm.Pricing) string {
	if p.Prompt == "0" && p.Completion == "0" {
		return green + "FREE" + reset
	}
	return dim + "$" + p.Prompt + "/" + p.Completion + reset
}

// printRecord prints a single capture record.
func (t *TUI) printRecord(r query.CaptureRecord) {
	ts := dim + "[" + r.Timestamp.Format("15:04:05") + "]" + reset

	switch r.Source {
	case "window":
		app := r.Metadata["app_class"]
		title := r.Metadata["window_title"]
		if len(title) > 50 {
			title = title[:47] + "..."
		}
		fmt.Printf("%s %swindow:%s %s%s%s - %s\n", ts, cyan, reset, bold, app, reset, title)

	case "clipboard":
		contentType := r.Metadata["content_type"]
		length := r.Metadata["length"]
		preview := r.TextData
		if len(preview) > 60 {
			preview = preview[:57] + "..."
		}
		preview = strings.ReplaceAll(preview, "\n", " ")
		fmt.Printf("%s %sclipboard:%s %s (%s chars): %s%s%s\n", ts, yellow, reset, contentType, length, dim, preview, reset)

	case "git":
		repo := r.Metadata["repo_name"]
		branch := r.Metadata["branch"]
		fmt.Printf("%s %sgit:%s %s%s%s @ %s\n", ts, magenta, reset, bold, repo, reset, branch)

	case "screen":
		size := r.Metadata["size_bytes"]
		fmt.Printf("%s %sscreen:%s %s bytes\n", ts, blue, reset, size)

	case "activity":
		state := r.Metadata["state"]
		idle := r.Metadata["idle_seconds"]
		stateColor := green
		if state == "idle" {
			stateColor = dim
		}
		fmt.Printf("%s %sactivity:%s %s%s%s (idle: %ss)\n", ts, dim, reset, stateColor, state, reset, idle)

	case "biometrics":
		level := r.Metadata["stress_level"]
		score := r.Metadata["stress_score"]
		levelColor := green
		switch level {
		case "elevated":
			levelColor = yellow
		case "high", "anxious":
			levelColor = red
		}
		fmt.Printf("%s %sstress:%s %s%s%s (score: %s)\n", ts, red, reset, levelColor, level, reset, score)

	default:
		fmt.Printf("%s %s\n", ts, r.Source)
	}
}

// showPrivacy displays current privacy settings.
func (t *TUI) showPrivacy() {
	fmt.Println()
	fmt.Println(red + "╭─" + reset + bold + " privacy & security " + reset + red + strings.Repeat("─", 37) + "╮" + reset)
	fmt.Println(red + "│" + reset)
	fmt.Println(red + "│" + reset + " " + bold + "Data Storage:" + reset)
	fmt.Printf(red+"│"+reset+"   Location: %s%s%s\n", dim, t.cfg.StoragePath, reset)
	fmt.Printf(red+"│"+reset+"   Permissions: %s0700 (owner only)%s\n", dim, reset)
	fmt.Println(red + "│" + reset)
	fmt.Println(red + "│" + reset + " " + bold + "Blocked Apps:" + reset + dim + " (never captured)" + reset)
	for _, app := range t.cfg.BlockedApps {
		fmt.Printf(red+"│"+reset+"   %s• %s%s\n", yellow, app, reset)
	}
	fmt.Println(red + "│" + reset)
	fmt.Println(red + "│" + reset + " " + bold + "Blocked URL Patterns:" + reset)
	for _, url := range t.cfg.BlockedURLs {
		fmt.Printf(red+"│"+reset+"   %s• %s%s\n", yellow, url, reset)
	}
	fmt.Println(red + "│" + reset)
	fmt.Println(red + "│" + reset + " " + bold + "Blocked Keywords:" + reset + dim + " (filtered from captures)" + reset)
	keywords := strings.Join(t.cfg.BlockedKeywords, ", ")
	if len(keywords) > 50 {
		keywords = keywords[:47] + "..."
	}
	fmt.Printf(red+"│"+reset+"   %s%s%s\n", dim, keywords, reset)
	fmt.Println(red + "│" + reset)
	fmt.Println(red + "│" + reset + " " + bold + "Commands:" + reset)
	fmt.Println(red + "│" + reset + "   /exclude <app>  - Add app to blocklist")
	fmt.Println(red + "│" + reset + "   /clear all      - Delete ALL captured data")
	fmt.Println(red + "│" + reset + "   /clear today    - Delete today's data")
	fmt.Println(red + "╰" + strings.Repeat("─", 58) + "╯" + reset)
}

// showExcluded shows currently excluded apps.
func (t *TUI) showExcluded() {
	fmt.Println()
	fmt.Println(yellow + "╭─" + reset + bold + " excluded apps " + reset + yellow + strings.Repeat("─", 42) + "╮" + reset)
	if len(t.cfg.BlockedApps) == 0 {
		fmt.Println(yellow + "│" + reset + " " + dim + "No apps excluded" + reset)
	} else {
		for _, app := range t.cfg.BlockedApps {
			fmt.Printf(yellow+"│"+reset+" • %s\n", app)
		}
	}
	fmt.Println(yellow + "│" + reset)
	fmt.Println(yellow + "│" + reset + " " + dim + "Use /exclude <appname> to add" + reset)
	fmt.Println(yellow + "╰" + strings.Repeat("─", 58) + "╯" + reset)
}

// excludeApp adds an app to the blocklist.
func (t *TUI) excludeApp(appName string) {
	appName = strings.ToLower(strings.TrimSpace(appName))
	if appName == "" {
		fmt.Println(red + "Error: Please provide an app name" + reset)
		return
	}

	// Check if already excluded
	for _, app := range t.cfg.BlockedApps {
		if strings.ToLower(app) == appName {
			fmt.Printf(yellow+"'%s' is already excluded%s\n", appName, reset)
			return
		}
	}

	t.cfg.BlockedApps = append(t.cfg.BlockedApps, appName)
	if err := t.cfg.Save(); err != nil {
		fmt.Printf(red+"Error saving config: %v%s\n", err, reset)
		return
	}

	fmt.Printf(green+"✓ '%s' added to blocklist%s\n", appName, reset)
	fmt.Println(dim + "The daemon will no longer capture this app" + reset)
}

// clearData clears captured data.
func (t *TUI) clearData(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Println(yellow + "Usage:" + reset)
		fmt.Println("  /clear all    - Delete ALL captured data")
		fmt.Println("  /clear today  - Delete today's data")
		fmt.Println("  /clear screen - Delete all screenshots")
		return
	}

	target := args[0]

	fmt.Print(red + bold + "⚠ WARNING: " + reset)
	switch target {
	case "all":
		fmt.Println("This will delete ALL your captured memories!")
	case "today":
		fmt.Println("This will delete all of today's captures!")
	case "screen", "screenshots":
		fmt.Println("This will delete all screenshots!")
	default:
		fmt.Printf("Unknown target: %s\n", target)
		return
	}

	fmt.Print(yellow + "Type 'yes' to confirm: " + reset)
	confirm, _ := t.reader.ReadString('\n')
	confirm = strings.TrimSpace(strings.ToLower(confirm))

	if confirm != "yes" {
		fmt.Println(dim + "Cancelled" + reset)
		return
	}

	var err error
	var deleted int64

	switch target {
	case "all":
		result, e := t.db.ExecContext(ctx, "DELETE FROM captures")
		err = e
		if err == nil {
			deleted, _ = result.RowsAffected()
		}
		// Also clear screenshot files
		screenshotDir := filepath.Join(t.cfg.StoragePath, "captures")
		os.RemoveAll(screenshotDir)

	case "today":
		today := time.Now().Format("2006-01-02")
		result, e := t.db.ExecContext(ctx, "DELETE FROM captures WHERE date(timestamp) = ?", today)
		err = e
		if err == nil {
			deleted, _ = result.RowsAffected()
		}

	case "screen", "screenshots":
		result, e := t.db.ExecContext(ctx, "DELETE FROM captures WHERE source = 'screen'")
		err = e
		if err == nil {
			deleted, _ = result.RowsAffected()
		}
		screenshotDir := filepath.Join(t.cfg.StoragePath, "captures")
		os.RemoveAll(screenshotDir)
	}

	if err != nil {
		fmt.Printf(red+"Error: %v%s\n", err, reset)
		return
	}

	fmt.Printf(green+"✓ Deleted %d records%s\n", deleted, reset)
}

// showIntegrations shows the status of external service connections.
func (t *TUI) showIntegrations() {
	fmt.Println()
	fmt.Println(blue + "╭─" + reset + bold + " integrations " + reset + blue + strings.Repeat("─", 43) + "╮" + reset)

	if t.integrations == nil {
		fmt.Println(blue + "│" + reset + " " + red + "Integrations not initialized" + reset)
		fmt.Println(blue + "╰" + strings.Repeat("─", 58) + "╯" + reset)
		return
	}

	status := t.integrations.GetProviderStatus()

	providers := []struct {
		name  string
		key   string
		icon  string
		color string
	}{
		{"Gmail", oauth.ProviderGmail, "📧", yellow},
		{"Slack", oauth.ProviderSlack, "💬", magenta},
		{"Google Calendar", oauth.ProviderCalendar, "📅", cyan},
	}

	fmt.Println(blue + "│" + reset)
	for _, p := range providers {
		s := status[p.key]
		statusIcon := red + "○" + reset
		statusText := dim + "not connected" + reset

		if !s["configured"] {
			statusText = dim + "not configured (set env vars)" + reset
		} else if s["authenticated"] {
			statusIcon = green + "●" + reset
			statusText = green + "connected" + reset
		} else {
			statusText = yellow + "ready to connect" + reset
		}

		fmt.Printf(blue+"│"+reset+" %s %s%-18s%s %s %s\n",
			p.icon, p.color, p.name, reset, statusIcon, statusText)
	}

	fmt.Println(blue + "│" + reset)
	fmt.Println(blue + "│" + reset + " " + bold + "Environment Variables:" + reset)
	fmt.Println(blue + "│" + reset + "   " + dim + "GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET" + reset)
	fmt.Println(blue + "│" + reset + "   " + dim + "SLACK_CLIENT_ID, SLACK_CLIENT_SECRET" + reset)
	fmt.Println(blue + "│" + reset)
	fmt.Println(blue + "│" + reset + " " + dim + "Use /connect <provider> to authenticate" + reset)
	fmt.Println(blue + "│" + reset + " " + dim + "Use /logout <provider> to disconnect" + reset)
	fmt.Println(blue + "╰" + strings.Repeat("─", 58) + "╯" + reset)
}

// showConnectHelp shows help for the connect command.
func (t *TUI) showConnectHelp() {
	fmt.Println()
	fmt.Println(blue + "╭─" + reset + bold + " connect " + reset + blue + strings.Repeat("─", 48) + "╮" + reset)
	fmt.Println(blue + "│" + reset)
	fmt.Println(blue + "│" + reset + " " + bold + "Usage:" + reset + " /connect <provider>")
	fmt.Println(blue + "│" + reset)
	fmt.Println(blue + "│" + reset + " " + bold + "Providers:" + reset)
	fmt.Println(blue + "│" + reset + "   " + yellow + "gmail" + reset + "     - Access your emails")
	fmt.Println(blue + "│" + reset + "   " + magenta + "slack" + reset + "     - Access your Slack messages")
	fmt.Println(blue + "│" + reset + "   " + cyan + "calendar" + reset + "  - Access your calendar events")
	fmt.Println(blue + "│" + reset)
	fmt.Println(blue + "│" + reset + " " + dim + "Example: /connect gmail" + reset)
	fmt.Println(blue + "╰" + strings.Repeat("─", 58) + "╯" + reset)
}

// connectProvider starts the OAuth flow for a provider.
func (t *TUI) connectProvider(ctx context.Context, provider string) {
	if t.integrations == nil {
		fmt.Println(red + "Error: Integrations not initialized" + reset)
		return
	}

	// Normalize provider name
	provider = strings.ToLower(provider)
	switch provider {
	case "gmail", "email", "mail":
		provider = oauth.ProviderGmail
	case "slack":
		provider = oauth.ProviderSlack
	case "calendar", "gcal", "google-calendar":
		provider = oauth.ProviderCalendar
	}

	// Check if credentials are configured
	if !oauth.IsProviderConfigured(provider) {
		t.showCredentialsHelp(provider)
		return
	}

	fmt.Println()
	spinner := NewSpinner()
	spinner.Start("Starting authentication")

	authURL, err := t.integrations.AuthenticateProvider(ctx, provider)
	spinner.Stop()

	if err != nil {
		fmt.Printf(red+"Error: %v%s\n", err, reset)
		return
	}

	fmt.Println()
	fmt.Println(blue + "╭─" + reset + bold + " authentication " + reset + blue + strings.Repeat("─", 41) + "╮" + reset)
	fmt.Println(blue + "│" + reset)
	fmt.Println(blue + "│" + reset + " " + bold + "Opening browser for authentication..." + reset)
	fmt.Println(blue + "│" + reset)
	fmt.Println(blue + "│" + reset + " " + dim + "If browser doesn't open, visit:" + reset)
	fmt.Println(blue + "│" + reset)

	// Try to open browser
	openBrowser(authURL)

	// Show URL (but truncate for display)
	displayURL := authURL
	if len(displayURL) > 54 {
		displayURL = displayURL[:51] + "..."
	}
	fmt.Println(blue + "│" + reset + " " + cyan + displayURL + reset)
	fmt.Println(blue + "│" + reset)
	fmt.Println(blue + "│" + reset + " " + yellow + "Waiting for authorization..." + reset)
	fmt.Println(blue + "│" + reset + " " + dim + "(This window will update when complete)" + reset)
	fmt.Println(blue + "╰" + strings.Repeat("─", 58) + "╯" + reset)
}

// logoutProvider removes authentication for a provider.
func (t *TUI) logoutProvider(provider string) {
	if t.integrations == nil {
		fmt.Println(red + "Error: Integrations not initialized" + reset)
		return
	}

	// Normalize provider name
	provider = strings.ToLower(provider)
	switch provider {
	case "gmail", "email", "mail":
		provider = oauth.ProviderGmail
	case "slack":
		provider = oauth.ProviderSlack
	case "calendar", "gcal", "google-calendar":
		provider = oauth.ProviderCalendar
	}

	fmt.Print(yellow + "Are you sure you want to disconnect? (y/N): " + reset)
	confirm, _ := t.reader.ReadString('\n')
	confirm = strings.TrimSpace(strings.ToLower(confirm))

	if confirm != "y" && confirm != "yes" {
		fmt.Println(dim + "Cancelled" + reset)
		return
	}

	if err := t.integrations.LogoutProvider(provider); err != nil {
		fmt.Printf(red+"Error: %v%s\n", err, reset)
		return
	}

	fmt.Printf(green+"✓ Disconnected from %s%s\n", provider, reset)
}

// openBrowser attempts to open a URL in the default browser.
func openBrowser(url string) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return
	}

	cmd.Start()
}

// showCredentialsHelp shows how to set up OAuth credentials for a provider.
func (t *TUI) showCredentialsHelp(provider string) {
	fmt.Println()
	fmt.Println(yellow + "╭─" + reset + bold + " credentials required " + reset + yellow + strings.Repeat("─", 35) + "╮" + reset)
	fmt.Println(yellow + "│" + reset)

	switch provider {
	case oauth.ProviderGmail, oauth.ProviderCalendar:
		fmt.Println(yellow + "│" + reset + " " + bold + "Google OAuth Setup" + reset)
		fmt.Println(yellow + "│" + reset)
		fmt.Println(yellow + "│" + reset + " " + cyan + "Option 1: Using gcloud CLI (recommended)" + reset)
		fmt.Println(yellow + "│" + reset + "   1. Install gcloud: https://cloud.google.com/sdk")
		fmt.Println(yellow + "│" + reset + "   2. Run: " + green + "/setup google" + reset)
		fmt.Println(yellow + "│" + reset)
		fmt.Println(yellow + "│" + reset + " " + cyan + "Option 2: Manual setup" + reset)
		fmt.Println(yellow + "│" + reset + "   1. Go to: console.cloud.google.com/apis/credentials")
		fmt.Println(yellow + "│" + reset + "   2. Create OAuth client ID (Desktop app)")
		fmt.Println(yellow + "│" + reset + "   3. Enable Gmail/Calendar APIs")
		fmt.Println(yellow + "│" + reset + "   4. Set environment variables:")
		fmt.Println(yellow + "│" + reset + "      " + dim + "export GOOGLE_CLIENT_ID=\"...\"" + reset)
		fmt.Println(yellow + "│" + reset + "      " + dim + "export GOOGLE_CLIENT_SECRET=\"...\"" + reset)

	case oauth.ProviderSlack:
		fmt.Println(yellow + "│" + reset + " " + bold + "Slack OAuth Setup" + reset)
		fmt.Println(yellow + "│" + reset)
		fmt.Println(yellow + "│" + reset + "   1. Go to: api.slack.com/apps")
		fmt.Println(yellow + "│" + reset + "   2. Create New App > From scratch")
		fmt.Println(yellow + "│" + reset + "   3. OAuth & Permissions > Add scopes:")
		fmt.Println(yellow + "│" + reset + "      channels:history, channels:read, users:read")
		fmt.Println(yellow + "│" + reset + "   4. Add redirect URL: http://localhost:8087/callback")
		fmt.Println(yellow + "│" + reset + "   5. Install to workspace")
		fmt.Println(yellow + "│" + reset + "   6. Set environment variables:")
		fmt.Println(yellow + "│" + reset + "      " + dim + "export SLACK_CLIENT_ID=\"...\"" + reset)
		fmt.Println(yellow + "│" + reset + "      " + dim + "export SLACK_CLIENT_SECRET=\"...\"" + reset)
	}

	fmt.Println(yellow + "│" + reset)
	fmt.Println(yellow + "╰" + strings.Repeat("─", 58) + "╯" + reset)
}

// showSetupHelp shows available setup commands.
func (t *TUI) showSetupHelp() {
	fmt.Println()
	fmt.Println(blue + "╭─" + reset + bold + " setup " + reset + blue + strings.Repeat("─", 50) + "╮" + reset)
	fmt.Println(blue + "│" + reset)
	fmt.Println(blue + "│" + reset + " " + bold + "Usage:" + reset + " /setup <provider>")
	fmt.Println(blue + "│" + reset)
	fmt.Println(blue + "│" + reset + " " + bold + "Providers:" + reset)
	fmt.Println(blue + "│" + reset + "   " + cyan + "google" + reset + "  - Setup Google OAuth (Gmail & Calendar)")
	fmt.Println(blue + "│" + reset + "   " + magenta + "slack" + reset + "   - Setup Slack OAuth")
	fmt.Println(blue + "│" + reset)
	fmt.Println(blue + "│" + reset + " " + dim + "This will guide you through creating OAuth credentials." + reset)
	fmt.Println(blue + "╰" + strings.Repeat("─", 58) + "╯" + reset)
}

// runSetup runs the setup wizard for a provider.
func (t *TUI) runSetup(ctx context.Context, provider string) {
	wizard := oauth.NewSetupWizard()

	provider = strings.ToLower(provider)
	switch provider {
	case "google", "gmail", "calendar":
		wizard.SetupGoogle(ctx)
	case "slack":
		wizard.SetupSlack(ctx)
	default:
		fmt.Println(red + "Unknown provider: " + provider + reset)
		fmt.Println(dim + "Available: google, slack" + reset)
	}
}

// backfillOCR runs OCR on all screenshots that don't have pre-computed text.
func (t *TUI) backfillOCR(ctx context.Context) {
	if t.apiKey == "" {
		fmt.Println(red + "Error: No API key configured for OCR" + reset)
		return
	}

	fmt.Println()
	fmt.Println(blue + "╭─" + reset + bold + " backfill OCR " + reset + blue + strings.Repeat("─", 43) + "╮" + reset)
	fmt.Println(blue + "│" + reset)

	// Count screenshots without OCR
	var total, needsOCR int
	t.db.QueryRow("SELECT COUNT(*) FROM captures WHERE source = 'screen'").Scan(&total)
	t.db.QueryRow("SELECT COUNT(*) FROM captures WHERE source = 'screen' AND (text_data IS NULL OR text_data = '')").Scan(&needsOCR)

	fmt.Printf(blue+"│"+reset+" Total screenshots: %d\n", total)
	fmt.Printf(blue+"│"+reset+" Need OCR: %s%d%s\n", yellow, needsOCR, reset)
	fmt.Println(blue + "│" + reset)

	if needsOCR == 0 {
		fmt.Println(blue + "│" + reset + " " + green + "All screenshots already have OCR!" + reset)
		fmt.Println(blue + "╰" + strings.Repeat("─", 58) + "╯" + reset)
		return
	}

	fmt.Println(blue + "│" + reset + " " + yellow + "This will make API calls and may take a while." + reset)
	fmt.Println(blue + "│" + reset + " " + dim + "Estimated time: ~" + fmt.Sprintf("%d", needsOCR*3) + " seconds" + reset)
	fmt.Println(blue + "╰" + strings.Repeat("─", 58) + "╯" + reset)
	fmt.Print("\n" + yellow + "Continue? [y/N]: " + reset)

	confirm, _ := t.reader.ReadString('\n')
	if strings.TrimSpace(strings.ToLower(confirm)) != "y" {
		fmt.Println(dim + "Cancelled" + reset)
		return
	}

	// Create OCR engine
	ocrEngine := ocr.NewVisionOCR(t.apiKey)
	if !ocrEngine.Available() {
		fmt.Println(red + "Error: OCR engine not available" + reset)
		return
	}

	// Get screenshots without OCR
	rows, err := t.db.QueryContext(ctx, `
		SELECT id, raw_data_path
		FROM captures
		WHERE source = 'screen'
		AND (text_data IS NULL OR text_data = '')
		AND raw_data_path IS NOT NULL
		ORDER BY timestamp DESC
	`)
	if err != nil {
		fmt.Printf(red+"Error: %v%s\n", err, reset)
		return
	}
	defer rows.Close()

	var toProcess []struct {
		id   int64
		path string
	}
	for rows.Next() {
		var id int64
		var path string
		rows.Scan(&id, &path)
		toProcess = append(toProcess, struct {
			id   int64
			path string
		}{id, path})
	}

	fmt.Println()
	processed := 0
	failed := 0

	for i, item := range toProcess {
		// Check if file exists
		if _, err := os.Stat(item.path); os.IsNotExist(err) {
			failed++
			continue
		}

		// Show progress
		fmt.Printf("\r%s[%d/%d]%s Processing... ", cyan, i+1, len(toProcess), reset)

		// Run OCR with timeout
		ocrCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		text, err := ocrEngine.ExtractTextFromFile(ocrCtx, item.path)
		cancel()

		if err != nil {
			failed++
			continue
		}

		// Update database
		_, err = t.db.ExecContext(ctx, "UPDATE captures SET text_data = ? WHERE id = ?", text, item.id)
		if err != nil {
			failed++
			continue
		}

		processed++
	}

	fmt.Println()
	fmt.Println()
	fmt.Printf(green+"✓ Processed: %d%s\n", processed, reset)
	if failed > 0 {
		fmt.Printf(yellow+"✗ Failed: %d%s\n", failed, reset)
	}
	fmt.Println(dim + "Future queries will be much faster!" + reset)
}

// triggerInsights manually runs LLM analysis on recent activity.
func (t *TUI) triggerInsights(ctx context.Context) {
	if t.apiKey == "" {
		fmt.Println(red + "Error: No API key configured" + reset)
		return
	}

	fmt.Println()
	fmt.Println(yellow + "╭─" + reset + bold + " generating insights " + reset + yellow + strings.Repeat("─", 36) + "╮" + reset)
	fmt.Println(yellow + "│" + reset)
	fmt.Println(yellow + "│" + reset + " " + dim + "Analyzing your recent activity..." + reset)
	fmt.Println(yellow + "│" + reset)

	spinner := NewSpinner()
	spinner.Start("Running LLM analysis")

	// Get recent captures (last hour)
	end := time.Now()
	start := end.Add(-1 * time.Hour)

	records, err := t.engine.GetByTimeRange(ctx, start, end)
	if err != nil {
		spinner.Stop()
		fmt.Printf(yellow+"│"+reset+" "+red+"Error: %v%s\n", err, reset)
		fmt.Println(yellow + "╰" + strings.Repeat("─", 58) + "╯" + reset)
		return
	}

	if len(records) == 0 {
		spinner.Stop()
		fmt.Println(yellow + "│" + reset + " " + dim + "No recent activity to analyze." + reset)
		fmt.Println(yellow + "╰" + strings.Repeat("─", 58) + "╯" + reset)
		return
	}

	// Build context for LLM
	contextStr := t.buildInsightContext(records)

	// Call LLM for insights
	insights, err := t.generateInsights(ctx, contextStr)
	spinner.Stop()

	if err != nil {
		fmt.Printf(yellow+"│"+reset+" "+red+"Error: %v%s\n", err, reset)
		fmt.Println(yellow + "╰" + strings.Repeat("─", 58) + "╯" + reset)
		return
	}

	if len(insights) == 0 {
		fmt.Println(yellow + "│" + reset + " " + dim + "No significant insights detected." + reset)
		fmt.Println(yellow + "╰" + strings.Repeat("─", 58) + "╯" + reset)
		return
	}

	// Display insights
	for _, insight := range insights {
		var icon string
		switch insight.Severity {
		case "warning":
			icon = "🟡"
		default:
			icon = "🔵"
		}
		fmt.Printf(yellow+"│"+reset+" %s %s%s%s\n", icon, bold, insight.Title, reset)
		fmt.Printf(yellow+"│"+reset+"   %s%s%s\n", dim, insight.Body, reset)
		fmt.Println(yellow + "│" + reset)

		// Save to database
		if t.store != nil {
			t.store.SaveInsight(&storage.InsightRecord{
				Type:          insight.Type,
				Severity:      insight.Severity,
				Title:         insight.Title,
				Body:          insight.Body,
				TriggerSource: "manual",
			})
		}
	}

	fmt.Println(yellow + "╰" + strings.Repeat("─", 58) + "╯" + reset)
}

type manualInsight struct {
	Type     string
	Title    string
	Body     string
	Severity string
}

func (t *TUI) buildInsightContext(records []query.CaptureRecord) string {
	var sb strings.Builder
	sb.WriteString("Activity from the last hour:\n\n")

	// Group by source
	bySource := make(map[string][]query.CaptureRecord)
	for _, r := range records {
		bySource[r.Source] = append(bySource[r.Source], r)
	}

	if windows := bySource["window"]; len(windows) > 0 {
		sb.WriteString("APPS:\n")
		seen := make(map[string]bool)
		for _, w := range windows {
			app := w.Metadata["app_class"]
			if app != "" && !seen[app] {
				seen[app] = true
				sb.WriteString(fmt.Sprintf("- %s\n", app))
			}
		}
		sb.WriteString("\n")
	}

	if bio := bySource["biometrics"]; len(bio) > 0 {
		sb.WriteString("STRESS:\n")
		for _, b := range bio {
			level := b.Metadata["stress_level"]
			score := b.Metadata["stress_score"]
			if level != "" {
				sb.WriteString(fmt.Sprintf("- %s: %s (score: %s)\n", b.Timestamp.Format("15:04"), level, score))
			}
		}
		sb.WriteString("\n")
	}

	if screens := bySource["screen"]; len(screens) > 0 {
		sb.WriteString("SCREEN:\n")
		count := 0
		for _, s := range screens {
			if s.TextData != "" && count < 3 {
				text := s.TextData
				if len(text) > 100 {
					text = text[:97] + "..."
				}
				sb.WriteString(fmt.Sprintf("- %s: %s\n", s.Timestamp.Format("15:04"), text))
				count++
			}
		}
	}

	return sb.String()
}

func (t *TUI) generateInsights(ctx context.Context, contextStr string) ([]manualInsight, error) {
	prompt := fmt.Sprintf(`Analyze this activity and generate 1-3 insights. Output JSON only.

%s

Output format:
[{"type": "pattern|summary", "title": "short title", "body": "insight (max 100 chars)", "severity": "info|warning"}]

Focus on: work patterns, stress correlations, productivity. Be specific. Output ONLY valid JSON.`, contextStr)

	// Use the LLM client directly
	if t.llmClient == nil {
		return nil, fmt.Errorf("no LLM client")
	}

	// Temporarily switch to cheap model for insights
	originalModel := t.llmClient.ChatModel
	t.llmClient.ChatModel = "deepseek/deepseek-chat"
	defer func() { t.llmClient.ChatModel = originalModel }()

	response, err := t.llmClient.Chat(ctx, []llm.Message{
		{Role: "user", Content: prompt},
	})
	if err != nil {
		return nil, err
	}

	// Parse JSON response
	content := strings.TrimSpace(response)
	startIdx := strings.Index(content, "[")
	endIdx := strings.LastIndex(content, "]")
	if startIdx >= 0 && endIdx > startIdx {
		content = content[startIdx : endIdx+1]
	}

	var insights []manualInsight
	if err := json.Unmarshal([]byte(content), &insights); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response")
	}

	return insights, nil
}

// showAlerts displays recent proactive insights from the daemon.
func (t *TUI) showAlerts(ctx context.Context) {
	if t.store == nil {
		fmt.Println(red + "Error: Storage not initialized" + reset)
		return
	}

	insights, err := t.store.GetRecentInsights(20)
	if err != nil {
		fmt.Printf(red+"Error: %v%s\n", err, reset)
		return
	}

	fmt.Println()
	fmt.Println(yellow + "╭─" + reset + bold + " proactive insights " + reset + yellow + strings.Repeat("─", 37) + "╮" + reset)

	if len(insights) == 0 {
		fmt.Println(yellow + "│" + reset)
		fmt.Println(yellow + "│" + reset + " " + dim + "No insights yet." + reset)
		fmt.Println(yellow + "│" + reset + " " + dim + "The daemon generates insights based on your activity." + reset)
		fmt.Println(yellow + "│" + reset)
		fmt.Println(yellow + "╰" + strings.Repeat("─", 58) + "╯" + reset)
		return
	}

	fmt.Println(yellow + "│" + reset)
	for _, i := range insights {
		// Severity icon
		var icon, severityColor string
		switch i.Severity {
		case "urgent":
			icon = "🔴"
			severityColor = red
		case "warning":
			icon = "🟡"
			severityColor = yellow
		default:
			icon = "🔵"
			severityColor = blue
		}

		// Format timestamp
		ts := i.CreatedAt.Format("15:04")
		if i.CreatedAt.Day() != time.Now().Day() {
			ts = i.CreatedAt.Format("Jan 02 15:04")
		}

		fmt.Printf(yellow+"│"+reset+" %s %s%s%s\n", icon, severityColor+bold, i.Title, reset)
		fmt.Printf(yellow+"│"+reset+"   %s%s%s\n", dim, i.Body, reset)
		fmt.Printf(yellow+"│"+reset+"   %s%s • %s%s\n", dim, ts, i.Type, reset)
		fmt.Println(yellow + "│" + reset)
	}

	fmt.Println(yellow + "│" + reset + " " + dim + "Insights are generated from stress patterns, context," + reset)
	fmt.Println(yellow + "│" + reset + " " + dim + "and periodic LLM analysis of your activity." + reset)
	fmt.Println(yellow + "╰" + strings.Repeat("─", 58) + "╯" + reset)
}

// startFocusMode starts an interactive conversation to create a focus mode.
func (t *TUI) startFocusMode(ctx context.Context) {
	if t.apiKey == "" {
		fmt.Println(red + "Error: No API key configured" + reset)
		return
	}

	if t.store == nil {
		fmt.Println(red + "Error: Storage not initialized" + reset)
		return
	}

	// Create a new builder
	t.focusBuilder = focus.NewBuilder(t.store, t.apiKey, "openai/gpt-4o-mini")
	t.inFocusChat = true

	fmt.Println()
	fmt.Println(green + "╭─" + reset + bold + " focus mode builder " + reset + green + strings.Repeat("─", 37) + "╮" + reset)
	fmt.Println(green + "│" + reset)
	fmt.Println(green + "│" + reset + " " + dim + "Let's create a focus mode for you." + reset)
	fmt.Println(green + "│" + reset + " " + dim + "Type 'cancel' to exit." + reset)
	fmt.Println(green + "│" + reset)

	// Get initial prompt from LLM
	initialResponse := t.focusBuilder.Start()
	fmt.Printf(green+"│"+reset+" %s%s%s\n", cyan, initialResponse, reset)
	fmt.Println(green + "│" + reset)

	// Interactive conversation loop
	for t.inFocusChat {
		fmt.Print(green + "│" + reset + " " + brightCyan + "You: " + reset)
		input, err := t.reader.ReadString('\n')
		if err != nil {
			break
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		if strings.ToLower(input) == "cancel" {
			fmt.Println(green + "│" + reset + " " + dim + "Cancelled." + reset)
			t.inFocusChat = false
			t.focusBuilder = nil
			break
		}

		// Send to builder
		response, mode, err := t.focusBuilder.Chat(input)
		if err != nil {
			fmt.Printf(green+"│"+reset+" "+red+"Error: %v%s\n", err, reset)
			continue
		}

		fmt.Println(green + "│" + reset)
		fmt.Printf(green+"│"+reset+" %s%s%s\n", cyan, response, reset)
		fmt.Println(green + "│" + reset)

		// Check if mode is complete
		if mode != nil {
			t.inFocusChat = false
			t.focusBuilder = nil

			fmt.Println(green + "│" + reset + " " + bold + green + "✓ Focus mode created!" + reset)
			fmt.Println(green + "│" + reset)
			fmt.Printf(green+"│"+reset+"   Name: %s%s%s\n", bold, mode.Name, reset)
			fmt.Printf(green+"│"+reset+"   Purpose: %s%s%s\n", dim, mode.Purpose, reset)
			fmt.Printf(green+"│"+reset+"   Allowed apps: %s%v%s\n", dim, mode.AllowedApps, reset)
			fmt.Printf(green+"│"+reset+"   Blocked apps: %s%v%s\n", dim, mode.BlockedApps, reset)
			fmt.Printf(green+"│"+reset+"   Blocked patterns: %s%v%s\n", dim, mode.BlockedPatterns, reset)
			if mode.DurationMinutes > 0 {
				fmt.Printf(green+"│"+reset+"   Duration: %s%d minutes%s\n", dim, mode.DurationMinutes, reset)
			}
			fmt.Println(green + "│" + reset)
			fmt.Printf(green+"│"+reset+" Use %s/start %s%s to begin.\n", cyan, mode.Name, reset)
		}
	}

	fmt.Println(green + "╰" + strings.Repeat("─", 58) + "╯" + reset)
}

// listFocusModes shows all saved focus modes.
func (t *TUI) listFocusModes(ctx context.Context) {
	if t.store == nil {
		fmt.Println(red + "Error: Storage not initialized" + reset)
		return
	}

	modes, err := t.store.ListFocusModes()
	if err != nil {
		fmt.Printf(red+"Error: %v%s\n", err, reset)
		return
	}

	fmt.Println()
	fmt.Println(green + "╭─" + reset + bold + " focus modes " + reset + green + strings.Repeat("─", 44) + "╮" + reset)

	if len(modes) == 0 {
		fmt.Println(green + "│" + reset)
		fmt.Println(green + "│" + reset + " " + dim + "No focus modes created yet." + reset)
		fmt.Println(green + "│" + reset + " " + dim + "Use /mode to create one." + reset)
		fmt.Println(green + "│" + reset)
		fmt.Println(green + "╰" + strings.Repeat("─", 58) + "╯" + reset)
		return
	}

	fmt.Println(green + "│" + reset)
	for _, m := range modes {
		// Parse apps
		allowedApps := focus.UnmarshalStringSlice(m.AllowedApps)
		blockedApps := focus.UnmarshalStringSlice(m.BlockedApps)

		fmt.Printf(green+"│"+reset+" %s%s%s %s(%s)%s\n", bold, m.Name, reset, dim, m.ID[:8], reset)
		fmt.Printf(green+"│"+reset+"   %s%s%s\n", dim, m.Purpose, reset)
		if len(allowedApps) > 0 {
			appsStr := strings.Join(allowedApps, ", ")
			if len(appsStr) > 40 {
				appsStr = appsStr[:37] + "..."
			}
			fmt.Printf(green+"│"+reset+"   Allowed: %s%s%s\n", cyan, appsStr, reset)
		}
		if len(blockedApps) > 0 {
			appsStr := strings.Join(blockedApps, ", ")
			if len(appsStr) > 40 {
				appsStr = appsStr[:37] + "..."
			}
			fmt.Printf(green+"│"+reset+"   Blocked: %s%s%s\n", red, appsStr, reset)
		}
		if m.DurationMinutes > 0 {
			fmt.Printf(green+"│"+reset+"   Duration: %s%d min%s\n", dim, m.DurationMinutes, reset)
		}

		// Get stats
		sessions, minutes, blocks, _ := t.store.GetFocusSessionStats(m.ID)
		if sessions > 0 {
			fmt.Printf(green+"│"+reset+"   Stats: %s%d sessions, %d min, %d blocks%s\n",
				dim, sessions, minutes, blocks, reset)
		}
		fmt.Println(green + "│" + reset)
	}

	fmt.Println(green + "│" + reset + " " + dim + "Use /start <name> to begin a session" + reset)
	fmt.Println(green + "╰" + strings.Repeat("─", 58) + "╯" + reset)
}

// startFocusSession activates a focus mode.
func (t *TUI) startFocusSession(ctx context.Context, nameOrID string) {
	if t.store == nil {
		fmt.Println(red + "Error: Storage not initialized" + reset)
		return
	}

	modes, err := t.store.ListFocusModes()
	if err != nil {
		fmt.Printf(red+"Error: %v%s\n", err, reset)
		return
	}

	// Find matching mode
	var match *storage.FocusModeRecord
	nameOrIDLower := strings.ToLower(nameOrID)
	for _, m := range modes {
		if strings.ToLower(m.Name) == nameOrIDLower ||
			strings.HasPrefix(m.ID, nameOrID) {
			match = &m
			break
		}
	}

	if match == nil {
		fmt.Printf(yellow+"No focus mode found matching '%s'%s\n", nameOrID, reset)
		fmt.Println(dim + "Use /modes to see available modes" + reset)
		return
	}

	// Convert to FocusMode
	mode := &focus.FocusMode{
		ID:              match.ID,
		Name:            match.Name,
		Purpose:         match.Purpose,
		AllowedApps:     focus.UnmarshalStringSlice(match.AllowedApps),
		BlockedApps:     focus.UnmarshalStringSlice(match.BlockedApps),
		BlockedPatterns: focus.UnmarshalStringSlice(match.BlockedPatterns),
		AllowedSites:    focus.UnmarshalStringSlice(match.AllowedSites),
		BrowserPolicy:   match.BrowserPolicy,
		DurationMinutes: match.DurationMinutes,
		CreatedAt:       match.CreatedAt,
	}

	// Start session
	sessionID, err := t.store.StartFocusSession(mode.ID, mode.DurationMinutes)
	if err != nil {
		fmt.Printf(red+"Error starting session: %v%s\n", err, reset)
		return
	}

	fmt.Println()
	fmt.Println(green + "╭─" + reset + bold + " focus mode active " + reset + green + strings.Repeat("─", 38) + "╮" + reset)
	fmt.Println(green + "│" + reset)
	fmt.Printf(green+"│"+reset+" %s%s%s is now active!\n", bold+green, mode.Name, reset)
	fmt.Println(green + "│" + reset)
	fmt.Printf(green+"│"+reset+" Session ID: %s%d%s\n", dim, sessionID, reset)
	fmt.Printf(green+"│"+reset+" Purpose: %s%s%s\n", dim, mode.Purpose, reset)
	fmt.Println(green + "│" + reset)
	fmt.Println(green + "│" + reset + " " + yellow + "The daemon will now enforce focus rules." + reset)
	fmt.Println(green + "│" + reset + " " + yellow + "Distracting windows will be closed after 5s warning." + reset)
	fmt.Println(green + "│" + reset)
	fmt.Println(green + "│" + reset + " " + dim + "Use /stop to end the session" + reset)
	fmt.Println(green + "╰" + strings.Repeat("─", 58) + "╯" + reset)

	// Start heartbeat goroutine to keep session alive
	t.startHeartbeat(sessionID)
}

// startHeartbeat starts a goroutine that periodically updates the session heartbeat.
func (t *TUI) startHeartbeat(sessionID int64) {
	// Stop any existing heartbeat
	t.stopHeartbeat()

	t.activeSessionID = sessionID
	t.heartbeatStop = make(chan struct{})
	t.heartbeatStopOnce = sync.Once{}

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-t.heartbeatStop:
				return
			case <-ticker.C:
				if t.store != nil && t.activeSessionID != 0 {
					t.store.UpdateFocusSessionHeartbeat(t.activeSessionID)
				}
			}
		}
	}()
}

// stopHeartbeat stops the heartbeat goroutine.
func (t *TUI) stopHeartbeat() {
	if t.heartbeatStop != nil {
		t.heartbeatStopOnce.Do(func() {
			close(t.heartbeatStop)
		})
	}
	t.activeSessionID = 0
}

// stopFocusSession ends the current focus session.
func (t *TUI) stopFocusSession(ctx context.Context) {
	// Stop heartbeat goroutine first
	t.stopHeartbeat()

	if t.store == nil {
		fmt.Println(red + "Error: Storage not initialized" + reset)
		return
	}

	// Get active session for stats display
	session, err := t.store.GetActiveFocusSession()
	if err != nil {
		fmt.Printf(red+"Error: %v%s\n", err, reset)
		return
	}

	if session == nil {
		fmt.Println(yellow + "No focus session currently active" + reset)
		return
	}

	// Get mode details to check if ended early
	mode, _ := t.store.GetFocusMode(session.ModeID)
	modeName := "Unknown"
	plannedDuration := 0
	if mode != nil {
		modeName = mode.Name
		plannedDuration = mode.DurationMinutes
	}

	// Calculate duration
	duration := time.Since(session.StartedAt)
	actualMinutes := int(duration.Minutes())

	quitReason := ""

	// Check if ended early (more than 5 minutes before planned duration)
	if plannedDuration > 0 && actualMinutes < (plannedDuration-5) {
		quitReason = t.askQuitReason(ctx, modeName, plannedDuration, actualMinutes)
	}

	// End the session with reason
	if err := t.store.EndFocusSessionWithReason(session.ID, quitReason, plannedDuration, actualMinutes); err != nil {
		fmt.Printf(red+"Error ending session: %v%s\n", err, reset)
		return
	}

	// Cleanup any other orphaned sessions
	ended, _ := t.store.EndAllActiveFocusSessions()
	if ended > 1 {
		fmt.Printf(dim+"(cleaned up %d orphaned sessions)%s\n", ended-1, reset)
	}

	fmt.Println()
	fmt.Println(blue + "╭─" + reset + bold + " session ended " + reset + blue + strings.Repeat("─", 42) + "╮" + reset)
	fmt.Println(blue + "│" + reset)
	fmt.Printf(blue+"│"+reset+" %s%s%s session complete.\n", bold, modeName, reset)
	fmt.Println(blue + "│" + reset)
	fmt.Printf(blue+"│"+reset+" Duration: %s%d / %d minutes%s\n", cyan, actualMinutes, plannedDuration, reset)
	fmt.Printf(blue+"│"+reset+" Distractions blocked: %s%d%s\n", cyan, session.BlocksCount, reset)
	if quitReason != "" {
		fmt.Printf(blue+"│"+reset+" Quit reason: %s%s%s\n", dim, quitReason, reset)
	}
	fmt.Println(blue + "│" + reset)
	fmt.Println(blue + "╰" + strings.Repeat("─", 58) + "╯" + reset)
}

// askQuitReason prompts the user why they're quitting early via LLM.
func (t *TUI) askQuitReason(ctx context.Context, modeName string, planned, actual int) string {
	if t.llmClient == nil || t.apiKey == "" {
		// Fallback to simple prompt if no LLM
		fmt.Println()
		fmt.Println(yellow + "╭─" + reset + bold + " early quit " + reset + yellow + strings.Repeat("─", 45) + "╮" + reset)
		fmt.Printf(yellow+"│"+reset+" You ended %s%s%s %d minutes early.\n", bold, modeName, reset, planned-actual)
		fmt.Println(yellow + "│" + reset)
		fmt.Print(yellow + "│" + reset + " Why are you stopping? (optional): ")
		reason, _ := t.reader.ReadString('\n')
		fmt.Println(yellow + "╰" + strings.Repeat("─", 58) + "╯" + reset)
		return strings.TrimSpace(reason)
	}

	// Use LLM to have a brief conversation about why they're quitting
	fmt.Println()
	fmt.Println(yellow + "╭─" + reset + bold + " early quit " + reset + yellow + strings.Repeat("─", 45) + "╮" + reset)
	fmt.Printf(yellow+"│"+reset+" You ended %s%s%s %d minutes early.\n", bold, modeName, reset, planned-actual)
	fmt.Println(yellow + "│" + reset)
	fmt.Println(yellow + "│" + reset + " " + dim + "The LLM will ask why - this helps track your focus patterns." + reset)
	fmt.Println(yellow + "│" + reset)

	prompt := fmt.Sprintf(`The user was in a focus mode called "%s" with a planned duration of %d minutes, but they're quitting after only %d minutes (%d minutes early).

Ask them in a friendly, conversational way why they're stopping early. Keep it very brief (1 sentence + question). Then summarize their response in 2-3 words.

Format your response like this:
[Your conversational question to the user]
---
SUMMARY: [2-3 word summary of their reason]

Example:
"No worries! What made you decide to stop now?"
---
SUMMARY: Task completed early`, modeName, planned, actual, planned-actual)

	response, err := t.llmClient.Chat(ctx, []llm.Message{
		{Role: "user", Content: prompt},
	})

	if err != nil {
		// Fallback
		fmt.Print(yellow + "│" + reset + " Why are you stopping? (optional): ")
		reason, _ := t.reader.ReadString('\n')
		fmt.Println(yellow + "╰" + strings.Repeat("─", 58) + "╯" + reset)
		return strings.TrimSpace(reason)
	}

	// Parse response to get question and extract summary later
	parts := strings.Split(response, "---")
	question := strings.TrimSpace(parts[0])

	fmt.Printf(yellow+"│"+reset+" %s\n", cyan+question+reset)
	fmt.Print(yellow + "│" + reset + " ")
	userResponse, _ := t.reader.ReadString('\n')
	userResponse = strings.TrimSpace(userResponse)

	// If user didn't answer, return empty
	if userResponse == "" {
		fmt.Println(yellow + "╰" + strings.Repeat("─", 58) + "╯" + reset)
		return ""
	}

	// Ask LLM to summarize the reason
	summaryPrompt := fmt.Sprintf(`User was asked why they stopped their focus session early. Their response: "%s"

Summarize this in 2-4 words max. Just output the summary, nothing else.
Examples: "Task completed", "Got distracted", "Emergency came up", "Feeling tired", "Meeting started"`, userResponse)

	summary, err := t.llmClient.Chat(ctx, []llm.Message{
		{Role: "user", Content: summaryPrompt},
	})

	fmt.Println(yellow + "╰" + strings.Repeat("─", 58) + "╯" + reset)

	if err != nil {
		// Return their raw response if summarization fails
		if len(userResponse) > 50 {
			return userResponse[:50]
		}
		return userResponse
	}

	return strings.TrimSpace(summary)
}

// showFocusStatus shows the current focus mode status.
func (t *TUI) showFocusStatus(ctx context.Context) {
	if t.store == nil {
		fmt.Println(red + "Error: Storage not initialized" + reset)
		return
	}

	session, err := t.store.GetActiveFocusSession()
	if err != nil {
		fmt.Printf(red+"Error: %v%s\n", err, reset)
		return
	}

	fmt.Println()
	fmt.Println(blue + "╭─" + reset + bold + " focus status " + reset + blue + strings.Repeat("─", 43) + "╮" + reset)
	fmt.Println(blue + "│" + reset)

	if session == nil {
		fmt.Println(blue + "│" + reset + " " + dim + "No focus session active" + reset)
		fmt.Println(blue + "│" + reset)
		fmt.Println(blue + "│" + reset + " " + dim + "Use /mode to create a focus mode" + reset)
		fmt.Println(blue + "│" + reset + " " + dim + "Use /start <name> to begin a session" + reset)
	} else {
		// Get mode details
		mode, _ := t.store.GetFocusMode(session.ModeID)
		modeName := "Unknown"
		if mode != nil {
			modeName = mode.Name
		}

		duration := time.Since(session.StartedAt)

		fmt.Printf(blue+"│"+reset+" %s●%s %s%s%s is active\n", green, reset, bold+green, modeName, reset)
		fmt.Println(blue + "│" + reset)
		fmt.Printf(blue+"│"+reset+" Started: %s%s%s\n", dim, session.StartedAt.Format("15:04"), reset)
		fmt.Printf(blue+"│"+reset+" Duration: %s%.0f minutes%s\n", cyan, duration.Minutes(), reset)
		fmt.Printf(blue+"│"+reset+" Blocks: %s%d%s\n", cyan, session.BlocksCount, reset)
		fmt.Println(blue + "│" + reset)
		fmt.Println(blue + "│" + reset + " " + dim + "Use /stop to end the session" + reset)
	}

	fmt.Println(blue + "│" + reset)
	fmt.Println(blue + "╰" + strings.Repeat("─", 58) + "╯" + reset)
}

// RunQuery runs the TUI in query mode (main entry point).
func RunQuery(apiKey string) error {
	// Open database
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	dbPath := filepath.Join(homeDir, ".local", "share", "mnemosyne", "mnemosyne.db")

	// Check if database exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return fmt.Errorf("database not found at %s - run the daemon first to capture some data", dbPath)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	tui := NewTUI(db, apiKey)

	// Set up signal handler to clean up focus sessions on exit
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		// Stop heartbeat and clean up any active focus sessions
		tui.stopHeartbeat()
		if tui.store != nil {
			if ended, err := tui.store.EndAllActiveFocusSessions(); err == nil && ended > 0 {
				fmt.Printf("\n%sCleaned up %d focus session(s)%s\n", dim, ended, reset)
			}
		}
		os.Exit(0)
	}()

	return tui.Run(context.Background())
}
