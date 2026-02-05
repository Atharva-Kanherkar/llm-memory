// Package activity captures user activity signals.
//
// On Wayland, we can't capture keystrokes directly (security feature).
// Instead, we capture:
// - Cursor position (to detect movement)
// - Active window changes (to detect activity)
// - Idle time (via system signals)
//
// This helps us understand:
// - Is the user actively working or idle?
// - How long have they been focused on current task?
// - When did they last interact?
package activity

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/Atharva-Kanherkar/mnemosyne/internal/capture"
	"github.com/Atharva-Kanherkar/mnemosyne/internal/platform"
)

// Capturer captures user activity state.
type Capturer struct {
	platform *platform.Platform

	// Track previous state to detect changes
	lastCursorX, lastCursorY int
	lastWindowAddr           string
	lastActivityTime         time.Time
	sessionStartTime         time.Time
}

// New creates a new activity Capturer.
func New(plat *platform.Platform) *Capturer {
	now := time.Now()
	return &Capturer{
		platform:         plat,
		lastActivityTime: now,
		sessionStartTime: now,
	}
}

// Name returns the capturer identifier.
func (c *Capturer) Name() string {
	return "activity"
}

// Available checks if activity capture is possible.
func (c *Capturer) Available() bool {
	// Available on Hyprland via hyprctl
	return c.platform.DisplayServer == platform.DisplayServerHyprland
}

// HyprlandCursorPos represents cursor position from hyprctl.
type HyprlandCursorPos struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// Capture gets the current activity state.
func (c *Capturer) Capture(ctx context.Context) (*capture.Result, error) {
	if c.platform.DisplayServer != platform.DisplayServerHyprland {
		return nil, fmt.Errorf("activity capture only supported on Hyprland")
	}

	result := capture.NewResult("activity")

	// Get cursor position
	cursorCmd := exec.CommandContext(ctx, "hyprctl", "cursorpos", "-j")
	cursorOutput, err := cursorCmd.Output()
	if err == nil {
		var pos HyprlandCursorPos
		if err := json.Unmarshal(cursorOutput, &pos); err == nil {
			result.SetMetadata("cursor_x", fmt.Sprintf("%d", pos.X))
			result.SetMetadata("cursor_y", fmt.Sprintf("%d", pos.Y))

			// Check if cursor moved
			cursorMoved := pos.X != c.lastCursorX || pos.Y != c.lastCursorY
			result.SetMetadata("cursor_moved", fmt.Sprintf("%t", cursorMoved))

			if cursorMoved {
				c.lastCursorX = pos.X
				c.lastCursorY = pos.Y
				c.lastActivityTime = time.Now()
			}
		}
	}

	// Get active window to detect window switches
	windowCmd := exec.CommandContext(ctx, "hyprctl", "activewindow", "-j")
	windowOutput, err := windowCmd.Output()
	if err == nil {
		var window struct {
			Address string `json:"address"`
			Class   string `json:"class"`
		}
		if err := json.Unmarshal(windowOutput, &window); err == nil {
			windowChanged := window.Address != c.lastWindowAddr
			result.SetMetadata("window_changed", fmt.Sprintf("%t", windowChanged))

			if windowChanged {
				c.lastWindowAddr = window.Address
				c.lastActivityTime = time.Now()
			}
		}
	}

	// Calculate idle time and session duration
	now := time.Now()
	idleSeconds := now.Sub(c.lastActivityTime).Seconds()
	sessionSeconds := now.Sub(c.sessionStartTime).Seconds()

	result.SetMetadata("idle_seconds", fmt.Sprintf("%.0f", idleSeconds))
	result.SetMetadata("session_seconds", fmt.Sprintf("%.0f", sessionSeconds))

	// Determine activity state
	var activityState string
	switch {
	case idleSeconds < 30:
		activityState = "active"
	case idleSeconds < 300: // 5 minutes
		activityState = "idle"
	case idleSeconds < 1800: // 30 minutes
		activityState = "away"
	default:
		activityState = "inactive"
	}
	result.SetMetadata("state", activityState)

	// Check if any activity detected this capture
	isActive := idleSeconds < float64(10) // Active if activity within 10 seconds
	result.SetMetadata("is_active", fmt.Sprintf("%t", isActive))

	return result, nil
}

// ResetSession resets the session timer (e.g., after a long break).
func (c *Capturer) ResetSession() {
	c.sessionStartTime = time.Now()
	c.lastActivityTime = time.Now()
}

// GetIdleTime returns how long since last detected activity.
func (c *Capturer) GetIdleTime() time.Duration {
	return time.Since(c.lastActivityTime)
}

// GetSessionDuration returns how long the current session has been active.
func (c *Capturer) GetSessionDuration() time.Duration {
	return time.Since(c.sessionStartTime)
}
