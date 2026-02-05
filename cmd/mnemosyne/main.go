// Package main is the entry point for the Mnemosyne daemon and query interface.
//
// Usage:
//
//	mnemosyne          - Start the capture daemon
//	mnemosyne daemon   - Start the capture daemon
//	mnemosyne query    - Start the interactive query interface
//	mnemosyne ask "question" - Ask a single question
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/Atharva-Kanherkar/mnemosyne/internal/config"
	"github.com/Atharva-Kanherkar/mnemosyne/internal/daemon"
	"github.com/Atharva-Kanherkar/mnemosyne/internal/platform"
	"github.com/Atharva-Kanherkar/mnemosyne/internal/storage"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// openDatabase opens the SQLite database.
func openDatabase(path string) (*sql.DB, error) {
	return sql.Open("sqlite3", path)
}

func main() {
	// Parse command
	cmd := "daemon"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	switch cmd {
	case "daemon", "d", "capture":
		runDaemon()
	case "query", "q", "tui":
		runQuery()
	case "ask", "a":
		if len(os.Args) < 3 {
			fmt.Println("Usage: mnemosyne ask \"your question\"")
			os.Exit(1)
		}
		question := strings.Join(os.Args[2:], " ")
		runAsk(question)
	case "stats", "s":
		runStats()
	case "widget", "w":
		// Parse widget subcommand
		if len(os.Args) > 2 {
			switch os.Args[2] {
			case "json":
				RunWidgetJSON()
			case "line", "oneline":
				RunWidgetOneLine()
			default:
				RunWidget()
			}
		} else {
			RunWidget()
		}
	case "help", "-h", "--help":
		printHelp()
	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`Mnemosyne - Your Personal Memory Assistant

Usage:
  mnemosyne [command]

Commands:
  daemon, d    Start the capture daemon (default)
  query, q     Start interactive query interface
  ask "..."    Ask a single question
  stats, s     Show capture statistics
  widget, w    Show focus mode widget (floating timer)
  widget json  Output widget state as JSON (for eww)
  widget line  Output one-line status (for waybar/polybar)
  help         Show this help

Environment:
  OPENROUTER_API_KEY   API key for OpenRouter (required for queries)

Examples:
  mnemosyne                           # Start capturing
  mnemosyne query                     # Interactive mode
  mnemosyne ask "what was I doing?"   # Quick question
  mnemosyne widget                    # Focus timer widget`)
}

func runDaemon() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Println("Mnemosyne starting...")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	plat, err := platform.Detect()
	if err != nil {
		log.Fatalf("Failed to detect platform: %v", err)
	}
	log.Printf("Platform detected: %s", plat)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get home directory: %v", err)
	}
	dataDir := filepath.Join(homeDir, ".local", "share", "mnemosyne")

	store, err := storage.New(dataDir)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()

	log.Printf("Storage initialized at: %s", dataDir)

	// Get API key for OCR pre-computation
	apiKey := cfg.LLM.OpenRouterKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENROUTER_API_KEY")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	manager := daemon.NewManager(cfg, plat, store, apiKey)
	manager.Start(ctx)

	log.Println("Mnemosyne running. Press Ctrl+C to stop.")

	sig := <-sigChan
	log.Printf("Received signal %v, shutting down...", sig)

	cancel()
	manager.Stop()

	stats, err := store.Stats()
	if err == nil {
		log.Printf("Session stats: %d captures stored", stats.TotalCaptures)
	}

	log.Println("Mnemosyne stopped.")
}

func runQuery() {
	cfg, _ := config.Load()
	apiKey := cfg.LLM.OpenRouterKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENROUTER_API_KEY")
	}

	if apiKey == "" {
		fmt.Println("Warning: No OpenRouter API key configured.")
		fmt.Println("Set OPENROUTER_API_KEY environment variable for LLM queries.")
		fmt.Println("You can still use /stats, /recent, /search commands.\n")
	}

	if err := RunQuery(apiKey); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runAsk(question string) {
	cfg, _ := config.Load()
	apiKey := cfg.LLM.OpenRouterKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENROUTER_API_KEY")
	}

	if apiKey == "" {
		fmt.Println("Error: No OpenRouter API key configured.")
		fmt.Println("Set OPENROUTER_API_KEY environment variable.")
		os.Exit(1)
	}

	// Open database
	homeDir, _ := os.UserHomeDir()
	dbPath := filepath.Join(homeDir, ".local", "share", "mnemosyne", "mnemosyne.db")

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		fmt.Println("Error: No captures found. Run the daemon first.")
		os.Exit(1)
	}

	db, err := openDatabase(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	tui := NewTUI(db, apiKey)

	fmt.Println("Thinking...")
	answer, err := tui.engine.Ask(context.Background(), question)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(answer)
}

func runStats() {
	homeDir, _ := os.UserHomeDir()
	dbPath := filepath.Join(homeDir, ".local", "share", "mnemosyne", "mnemosyne.db")

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		fmt.Println("No captures found. Run the daemon first.")
		os.Exit(1)
	}

	db, err := openDatabase(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	tui := NewTUI(db, "")
	tui.showStats(context.Background())
}
