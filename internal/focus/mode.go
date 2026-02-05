// Package focus provides AI-powered focus mode with distraction blocking.
package focus

import (
	"encoding/json"
	"time"
)

// FocusMode represents a saved focus mode with rules.
type FocusMode struct {
	ID      string `json:"id"`
	Name    string `json:"name"`    // "Study Mode"
	Purpose string `json:"purpose"` // "Studying algorithms using books, ChatGPT, Claude"

	// Rules
	AllowedApps     []string `json:"allowed_apps"`     // ["code", "firefox", "kitty", "zathura"]
	BlockedApps     []string `json:"blocked_apps"`     // ["slack", "discord"]
	BlockedPatterns []string `json:"blocked_patterns"` // ["youtube", "reddit", "twitter"]

	// Browser policy
	BrowserPolicy string   `json:"browser_policy"` // "ask_llm" | "allowlist" | "block_all"
	AllowedSites  []string `json:"allowed_sites"`  // ["github.com", "stackoverflow.com", "claude.ai"]

	// Timing
	DurationMinutes int       `json:"duration_minutes"` // 0 = until manually stopped
	CreatedAt       time.Time `json:"created_at"`
}

// FocusSession represents an active or completed focus session.
type FocusSession struct {
	ID          int64      `json:"id"`
	ModeID      string     `json:"mode_id"`
	StartedAt   time.Time  `json:"started_at"`
	EndedAt     *time.Time `json:"ended_at,omitempty"`
	BlocksCount int        `json:"blocks_count"` // How many distractions blocked
}

// Decision represents an enforcement decision for a window.
type Decision struct {
	Allowed bool
	Reason  string
	Action  string // "allow" | "warn" | "close"
}

// Window represents info about a window from Hyprland.
type Window struct {
	Address string `json:"address"`
	Class   string `json:"class"`
	Title   string `json:"title"`
	PID     int    `json:"pid"`
}

// MarshalAllowedApps converts allowed apps to JSON string.
func (m *FocusMode) MarshalAllowedApps() string {
	data, _ := json.Marshal(m.AllowedApps)
	return string(data)
}

// MarshalBlockedApps converts blocked apps to JSON string.
func (m *FocusMode) MarshalBlockedApps() string {
	data, _ := json.Marshal(m.BlockedApps)
	return string(data)
}

// MarshalBlockedPatterns converts blocked patterns to JSON string.
func (m *FocusMode) MarshalBlockedPatterns() string {
	data, _ := json.Marshal(m.BlockedPatterns)
	return string(data)
}

// MarshalAllowedSites converts allowed sites to JSON string.
func (m *FocusMode) MarshalAllowedSites() string {
	data, _ := json.Marshal(m.AllowedSites)
	return string(data)
}

// UnmarshalStringSlice parses a JSON string into a string slice.
func UnmarshalStringSlice(data string) []string {
	if data == "" {
		return nil
	}
	var result []string
	json.Unmarshal([]byte(data), &result)
	return result
}

// BrowserPolicies
const (
	BrowserPolicyAskLLM    = "ask_llm"
	BrowserPolicyAllowlist = "allowlist"
	BrowserPolicyBlockAll  = "block_all"
)

// Common browser app classes
var BrowserClasses = []string{
	"firefox",
	"chromium",
	"google-chrome",
	"brave-browser",
	"microsoft-edge",
	"vivaldi",
	"opera",
	"librewolf",
	"waterfox",
	"zen",         // Zen browser
	"zen-browser",
}

// IsBrowser checks if an app class is a browser.
func IsBrowser(appClass string) bool {
	for _, browser := range BrowserClasses {
		if appClass == browser {
			return true
		}
	}
	return false
}
