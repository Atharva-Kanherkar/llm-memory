package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/Atharva-Kanherkar/mnemosyne/internal/focus"
)

// ANSI escape codes for colors and styling
const (
	// Colors
	cReset   = "\033[0m"
	cBold    = "\033[1m"
	cDim     = "\033[2m"
	cBlink   = "\033[5m"
	cReverse = "\033[7m"

	// Foreground colors
	cBlack   = "\033[30m"
	cRed     = "\033[31m"
	cGreen   = "\033[32m"
	cYellow  = "\033[33m"
	cBlue    = "\033[34m"
	cMagenta = "\033[35m"
	cCyan    = "\033[36m"
	cWhite   = "\033[37m"

	// Bright colors
	cBrightGreen  = "\033[92m"
	cBrightYellow = "\033[93m"
	cBrightCyan   = "\033[96m"
	cBrightWhite  = "\033[97m"

	// Background colors
	cBgBlack = "\033[40m"
	cBgRed   = "\033[41m"
	cBgGreen = "\033[42m"

	// Clear screen and cursor control
	clearScreen = "\033[2J\033[H"
	hideCursor  = "\033[?25l"
	showCursor  = "\033[?25h"
)

// RunWidget runs the aesthetic focus mode widget.
func RunWidget() error {
	homeDir, _ := os.UserHomeDir()
	dataDir := filepath.Join(homeDir, ".local", "share", "mnemosyne")

	// Hide cursor for cleaner display
	fmt.Print(hideCursor)
	defer fmt.Print(showCursor)

	// Handle Ctrl+C gracefully
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c
		fmt.Print(showCursor)
		fmt.Print(clearScreen)
		os.Exit(0)
	}()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var lastDecision string
	decisionTime := time.Time{}

	for {
		state, err := focus.ReadWidgetState(dataDir)
		if err != nil {
			state = &focus.WidgetState{Active: false}
		}

		// Track decision display time
		if state.LastDecision != lastDecision {
			lastDecision = state.LastDecision
			decisionTime = time.Now()
		}

		// Clear and draw
		fmt.Print(clearScreen)
		drawWidget(state, decisionTime)

		<-ticker.C
	}
}

func drawWidget(state *focus.WidgetState, decisionTime time.Time) {
	width := 42

	if !state.Active {
		drawInactiveWidget(width)
		return
	}

	drawActiveWidget(state, width, decisionTime)
}

func drawInactiveWidget(width int) {
	// Minimalist inactive state
	fmt.Println()
	fmt.Printf("  %s%s┌%s┐%s\n", cDim, cCyan, strings.Repeat("─", width-2), cReset)
	fmt.Printf("  %s%s│%s%s  ○  FOCUS MODE INACTIVE  %s%s│%s\n",
		cDim, cCyan, cReset, cDim, strings.Repeat(" ", width-28), cCyan, cReset)
	fmt.Printf("  %s%s└%s┘%s\n", cDim, cCyan, strings.Repeat("─", width-2), cReset)
	fmt.Println()
	fmt.Printf("  %sRun /mode in mnemosyne to start%s\n", cDim, cReset)
}

func drawActiveWidget(state *focus.WidgetState, width int, decisionTime time.Time) {
	// Calculate time
	hours := state.ElapsedSecs / 3600
	minutes := (state.ElapsedSecs % 3600) / 60
	seconds := state.ElapsedSecs % 60

	// Determine pulse state for the dot (blinks every second)
	pulse := state.ElapsedSecs%2 == 0
	dotColor := cBrightGreen
	dot := "●"
	if pulse {
		dotColor = cGreen
	}

	// Top border with glow effect
	fmt.Println()
	fmt.Printf("  %s%s╭%s╮%s\n", cGreen, cBold, strings.Repeat("─", width-2), cReset)

	// Status line with pulsing dot
	statusLine := fmt.Sprintf("%s%s %s%s%s FOCUSING %s",
		dotColor, dot, cReset, cBold, cBrightWhite, cReset)
	fmt.Printf("  %s│%s %s %s%s│%s\n",
		cGreen, cReset, statusLine, strings.Repeat(" ", width-22), cGreen, cReset)

	// Separator
	fmt.Printf("  %s├%s┤%s\n", cGreen, strings.Repeat("─", width-2), cReset)

	// Timer display - big and centered
	timeStr := fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
	timerLine := fmt.Sprintf("%s%s%s", cBold+cBrightCyan, timeStr, cReset)
	padding := (width - 12) / 2
	fmt.Printf("  %s│%s%s%s%s%s│%s\n",
		cGreen, cReset, strings.Repeat(" ", padding), timerLine, strings.Repeat(" ", padding-2), cGreen, cReset)

	// Mode name
	modeName := truncateForWidget(state.ModeName, width-6)
	modePadding := (width - len(modeName) - 4) / 2
	fmt.Printf("  %s│%s%s%s%s%s%s%s│%s\n",
		cGreen, cReset, strings.Repeat(" ", modePadding), cDim, modeName, cReset,
		strings.Repeat(" ", width-len(modeName)-modePadding-4), cGreen, cReset)

	// Separator
	fmt.Printf("  %s├%s┤%s\n", cGreen, strings.Repeat("─", width-2), cReset)

	// Stats line
	blocksStr := fmt.Sprintf("Blocked: %s%d%s", cYellow, state.BlocksCount, cReset)
	fmt.Printf("  %s│%s  %s%s%s│%s\n",
		cGreen, cReset, blocksStr, strings.Repeat(" ", width-16-len(fmt.Sprintf("%d", state.BlocksCount))), cGreen, cReset)

	// Last decision (fades after 5 seconds)
	if state.LastDecision != "" && time.Since(decisionTime) < 5*time.Second {
		decisionColor := cBrightGreen
		if state.LastAction == "block" || state.LastAction == "warn" {
			decisionColor = cRed
		}
		decision := truncateForWidget(state.LastDecision, width-6)
		fmt.Printf("  %s│%s  %s%s%s%s%s│%s\n",
			cGreen, cReset, decisionColor, decision, cReset,
			strings.Repeat(" ", width-len(decision)-6), cGreen, cReset)
	} else {
		// Empty line when no recent decision
		fmt.Printf("  %s│%s%s%s│%s\n",
			cGreen, cReset, strings.Repeat(" ", width-4), cGreen, cReset)
	}

	// Bottom border
	fmt.Printf("  %s╰%s╯%s\n", cGreen, strings.Repeat("─", width-2), cReset)

	// Hint
	fmt.Println()
	fmt.Printf("  %sCtrl+C to close • /stop to end session%s\n", cDim, cReset)
}

func truncateForWidget(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

// RunWidgetOneLine outputs a single line for status bars (waybar, polybar, etc.)
func RunWidgetOneLine() {
	homeDir, _ := os.UserHomeDir()
	dataDir := filepath.Join(homeDir, ".local", "share", "mnemosyne")

	state, err := focus.ReadWidgetState(dataDir)
	if err != nil || !state.Active {
		fmt.Println("○ Focus Off")
		return
	}

	hours := state.ElapsedSecs / 3600
	minutes := (state.ElapsedSecs % 3600) / 60
	seconds := state.ElapsedSecs % 60

	fmt.Printf("● %s %02d:%02d:%02d [%d blocked]\n",
		state.ModeName, hours, minutes, seconds, state.BlocksCount)
}

// RunWidgetJSON outputs JSON for eww/custom widgets.
func RunWidgetJSON() {
	homeDir, _ := os.UserHomeDir()
	dataDir := filepath.Join(homeDir, ".local", "share", "mnemosyne")

	// Read the JSON file directly
	data, err := os.ReadFile(filepath.Join(dataDir, "focus_widget.json"))
	if err != nil {
		fmt.Println("{\"active\": false}")
		return
	}
	fmt.Println(string(data))
}
