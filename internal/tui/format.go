// Package tui provides terminal UI formatting utilities.
package tui

import (
	"regexp"
	"strings"
)

// ANSI color codes
const (
	Reset     = "\033[0m"
	Bold      = "\033[1m"
	Dim       = "\033[2m"
	Italic    = "\033[3m"
	Underline = "\033[4m"

	// Colors
	Black   = "\033[30m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"

	// Bright colors
	BrightBlack   = "\033[90m"
	BrightRed     = "\033[91m"
	BrightGreen   = "\033[92m"
	BrightYellow  = "\033[93m"
	BrightBlue    = "\033[94m"
	BrightMagenta = "\033[95m"
	BrightCyan    = "\033[96m"
	BrightWhite   = "\033[97m"

	// Background colors
	BgBlue   = "\033[44m"
	BgGreen  = "\033[42m"
	BgYellow = "\033[43m"
	BgRed    = "\033[41m"
)

// FormatResponse converts markdown-style LLM output to colored terminal output.
func FormatResponse(text string) string {
	lines := strings.Split(text, "\n")
	var result []string

	for _, line := range lines {
		formatted := formatLine(line)
		result = append(result, formatted)
	}

	return strings.Join(result, "\n")
}

// formatLine formats a single line.
func formatLine(line string) string {
	trimmed := strings.TrimSpace(line)

	// Headers: ### Header -> colored bold
	if strings.HasPrefix(trimmed, "### ") {
		content := strings.TrimPrefix(trimmed, "### ")
		return Cyan + Bold + "━━━ " + content + " ━━━" + Reset
	}
	if strings.HasPrefix(trimmed, "## ") {
		content := strings.TrimPrefix(trimmed, "## ")
		return BrightCyan + Bold + "▶ " + content + Reset
	}
	if strings.HasPrefix(trimmed, "# ") {
		content := strings.TrimPrefix(trimmed, "# ")
		return BrightWhite + Bold + "█ " + content + Reset
	}

	// Bullet points
	if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
		content := trimmed[2:]
		content = formatInline(content)
		return "  " + Cyan + "•" + Reset + " " + content
	}

	// Numbered lists
	if match, _ := regexp.MatchString(`^\d+\.\s`, trimmed); match {
		parts := strings.SplitN(trimmed, ". ", 2)
		if len(parts) == 2 {
			num := parts[0]
			content := formatInline(parts[1])
			return "  " + Yellow + num + "." + Reset + " " + content
		}
	}

	// Code blocks
	if strings.HasPrefix(trimmed, "```") {
		if trimmed == "```" {
			return Dim + "  ─────────────────────" + Reset
		}
		lang := strings.TrimPrefix(trimmed, "```")
		return Dim + "  ───── " + lang + " ─────" + Reset
	}

	// Horizontal rules
	if trimmed == "---" || trimmed == "***" || trimmed == "___" {
		return Dim + "────────────────────────────────────────" + Reset
	}

	// Regular line - apply inline formatting
	return formatInline(line)
}

// formatInline handles inline formatting like **bold** and *italic*.
func formatInline(text string) string {
	// Bold: **text** or __text__
	boldRegex := regexp.MustCompile(`\*\*(.+?)\*\*|__(.+?)__`)
	text = boldRegex.ReplaceAllStringFunc(text, func(match string) string {
		// Extract content between markers
		content := strings.Trim(match, "*_")
		return Bold + BrightWhite + content + Reset
	})

	// Italic: *text* or _text_ (but not inside words)
	italicRegex := regexp.MustCompile(`(?:^|[^*_])\*([^*]+?)\*(?:[^*_]|$)|(?:^|[^*_])_([^_]+?)_(?:[^*_]|$)`)
	text = italicRegex.ReplaceAllStringFunc(text, func(match string) string {
		content := strings.Trim(match, "*_ ")
		return Italic + content + Reset
	})

	// Inline code: `code`
	codeRegex := regexp.MustCompile("`([^`]+)`")
	text = codeRegex.ReplaceAllStringFunc(text, func(match string) string {
		content := strings.Trim(match, "`")
		return BgBlue + White + " " + content + " " + Reset
	})

	// Timestamps: [HH:MM:SS] or [YYYY-MM-DD HH:MM:SS]
	timeRegex := regexp.MustCompile(`\[(\d{2}:\d{2}:\d{2}|\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2})\]`)
	text = timeRegex.ReplaceAllStringFunc(text, func(match string) string {
		return Dim + match + Reset
	})

	// Stress levels
	text = strings.ReplaceAll(text, "calm", Green+"calm"+Reset)
	text = strings.ReplaceAll(text, "normal", Blue+"normal"+Reset)
	text = strings.ReplaceAll(text, "elevated", Yellow+"elevated"+Reset)
	text = strings.ReplaceAll(text, "high stress", BrightRed+"high stress"+Reset)
	text = strings.ReplaceAll(text, "anxious", Red+Bold+"anxious"+Reset)

	return text
}

// Box draws a box around text.
func Box(title, content string) string {
	lines := strings.Split(content, "\n")
	maxLen := len(title)
	for _, line := range lines {
		if len(stripAnsi(line)) > maxLen {
			maxLen = len(stripAnsi(line))
		}
	}

	width := maxLen + 4
	top := Cyan + "╭" + strings.Repeat("─", width) + "╮" + Reset
	titleLine := Cyan + "│" + Reset + Bold + " " + title + strings.Repeat(" ", width-len(title)-1) + Cyan + "│" + Reset
	separator := Cyan + "├" + strings.Repeat("─", width) + "┤" + Reset
	bottom := Cyan + "╰" + strings.Repeat("─", width) + "╯" + Reset

	var result []string
	result = append(result, top, titleLine, separator)
	for _, line := range lines {
		padding := width - len(stripAnsi(line)) - 1
		if padding < 0 {
			padding = 0
		}
		result = append(result, Cyan+"│"+Reset+" "+line+strings.Repeat(" ", padding)+Cyan+"│"+Reset)
	}
	result = append(result, bottom)

	return strings.Join(result, "\n")
}

// stripAnsi removes ANSI escape codes for length calculation.
func stripAnsi(text string) string {
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return ansiRegex.ReplaceAllString(text, "")
}

// ProgressBar creates a simple progress bar.
func ProgressBar(label string, value, max float64, width int) string {
	if max == 0 {
		max = 100
	}
	percentage := value / max
	filled := int(percentage * float64(width))
	if filled > width {
		filled = width
	}

	var color string
	switch {
	case percentage < 0.3:
		color = Green
	case percentage < 0.6:
		color = Yellow
	case percentage < 0.8:
		color = BrightYellow
	default:
		color = Red
	}

	bar := color + strings.Repeat("█", filled) + Dim + strings.Repeat("░", width-filled) + Reset
	return label + " [" + bar + "] " + Dim + string(rune(int(percentage*100))) + "%" + Reset
}

// StressIndicator returns a colored stress indicator.
func StressIndicator(level string) string {
	switch level {
	case "calm":
		return Green + "●" + Reset + " calm"
	case "normal":
		return Blue + "●" + Reset + " normal"
	case "elevated":
		return Yellow + "●" + Reset + " elevated"
	case "high":
		return BrightRed + "●" + Reset + " high"
	case "anxious":
		return Red + Bold + "●" + Reset + " " + Red + "anxious" + Reset
	default:
		return Dim + "○" + Reset + " " + level
	}
}

// Spinner characters for loading animation
var SpinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
