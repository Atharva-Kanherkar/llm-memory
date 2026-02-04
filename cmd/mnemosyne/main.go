// Package main is the entry point for the Mnemosyne daemon.
//
// Go convention: the main package is always called "main" and must have
// a main() function. The binary name comes from the directory name (mnemosyne).
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Atharva-Kanherkar/mnemosyne/internal/capture/window"
	"github.com/Atharva-Kanherkar/mnemosyne/internal/config"
	"github.com/Atharva-Kanherkar/mnemosyne/internal/platform"
)

func main() {
	// Set up logging with timestamps
	// log.SetFlags controls what prefix each log line gets
	// Ldate = date, Ltime = time, Lshortfile = filename:line
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	log.Println("Mnemosyne starting...")

	// Load configuration
	// We'll implement this next - for now it returns defaults
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Detect platform (Hyprland, X11, macOS, etc.)
	// This tells us which capture methods to use
	plat, err := platform.Detect()
	if err != nil {
		log.Fatalf("Failed to detect platform: %v", err)
	}
	log.Printf("Platform detected: %s", plat)

	// Create a context that we'll cancel on shutdown
	// Context is Go's way of handling cancellation and timeouts
	// When we cancel this context, all goroutines using it will know to stop
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure cancel is called when main exits

	// Set up signal handling for graceful shutdown
	// This is how daemons handle Ctrl+C (SIGINT) and kill signals (SIGTERM)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the capture loop in a goroutine
	// A goroutine is a lightweight thread managed by Go runtime
	// The 'go' keyword spawns it to run concurrently
	go captureLoop(ctx, cfg, plat)

	// Wait for shutdown signal
	// This blocks until we receive SIGINT or SIGTERM
	sig := <-sigChan
	log.Printf("Received signal %v, shutting down...", sig)

	// Cancel the context - this signals all goroutines to stop
	cancel()

	// Give goroutines time to clean up
	time.Sleep(500 * time.Millisecond)

	log.Println("Mnemosyne stopped.")
}

// captureLoop runs the main capture cycle.
// It takes a context (for cancellation), config, and platform info.
func captureLoop(ctx context.Context, cfg *config.Config, plat *platform.Platform) {
	// A Ticker sends a value on its channel at regular intervals
	// This is how we implement "capture every N seconds"
	ticker := time.NewTicker(time.Duration(cfg.CaptureIntervalSeconds) * time.Second)
	defer ticker.Stop() // Clean up the ticker when we exit

	// Create the window capturer
	windowCapturer := window.New(plat)
	if !windowCapturer.Available() {
		log.Println("WARNING: Window capture not available on this system")
	}

	log.Printf("Capture loop started (interval: %ds)", cfg.CaptureIntervalSeconds)

	// Track the last window to avoid logging duplicates
	var lastWindow string

	for {
		// select is Go's way of waiting on multiple channels
		// It blocks until one of the cases is ready
		select {
		case <-ctx.Done():
			// Context was cancelled - time to shut down
			log.Println("Capture loop stopping...")
			return

		case <-ticker.C:
			// Ticker fired - time to capture
			if windowCapturer.Available() {
				result, err := windowCapturer.Capture(ctx)
				if err != nil {
					log.Printf("Window capture error: %v", err)
					continue
				}

				// Build a summary of the current window
				appClass := result.Metadata["app_class"]
				title := result.Metadata["window_title"]
				workspace := result.Metadata["workspace_name"]

				// Truncate long titles for cleaner logs
				if len(title) > 60 {
					title = title[:57] + "..."
				}

				// Only log if window changed
				currentWindow := appClass + "|" + title
				if currentWindow != lastWindow {
					log.Printf("[%s] %s - %s", workspace, strings.ToUpper(appClass), title)
					lastWindow = currentWindow
				}
			}
		}
	}
}
