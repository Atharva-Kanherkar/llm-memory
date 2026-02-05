package main

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Atharva-Kanherkar/mnemosyne/internal/config"
	"github.com/Atharva-Kanherkar/mnemosyne/internal/llm"
	"github.com/Atharva-Kanherkar/mnemosyne/internal/query"
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
	engine    *query.Engine
	reader    *bufio.Reader
	llmClient *llm.Client
	apiKey    string
	db        *sql.DB
	cfg       *config.Config
	debug     bool
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

	return &TUI{
		engine:    query.NewWithOCR(db, llmClient, apiKey),
		reader:    bufio.NewReader(os.Stdin),
		llmClient: llmClient,
		apiKey:    apiKey,
		db:        db,
		cfg:       cfg,
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
	fmt.Println(blue + "    /model" + reset + dim + "     list or change AI model" + reset)
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
	return tui.Run(context.Background())
}
