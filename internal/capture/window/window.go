// Package window provides window information capture.
//
// It detects the active window and extracts:
// - Application class (e.g., "firefox", "code", "kitty")
// - Window title (e.g., "GitHub - Mozilla Firefox")
// - Process ID
// - Workspace information
//
// This is crucial for context - knowing WHAT you were looking at.
package window

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/Atharva-Kanherkar/mnemosyne/internal/capture"
	"github.com/Atharva-Kanherkar/mnemosyne/internal/platform"
)

// Capturer captures active window information.
// It implements the capture.Capturer interface.
type Capturer struct {
	platform *platform.Platform
}

// New creates a new window Capturer.
// In Go, "New" functions are the conventional way to create instances.
func New(plat *platform.Platform) *Capturer {
	return &Capturer{
		platform: plat,
	}
}

// Name returns the capturer identifier.
func (c *Capturer) Name() string {
	return "window"
}

// Available checks if window capture is possible on this system.
func (c *Capturer) Available() bool {
	return c.platform.CanCaptureWindow()
}

// Capture gets the current active window information.
func (c *Capturer) Capture(ctx context.Context) (*capture.Result, error) {
	// Dispatch to the appropriate implementation based on display server
	switch c.platform.DisplayServer {
	case platform.DisplayServerHyprland:
		return c.captureHyprland(ctx)
	case platform.DisplayServerX11:
		return c.captureX11(ctx)
	case platform.DisplayServerMacOS:
		return c.captureMacOS(ctx)
	default:
		return nil, fmt.Errorf("unsupported display server: %s", c.platform.DisplayServer)
	}
}

// HyprlandWindow represents the JSON response from hyprctl activewindow -j
// The struct tags map JSON keys to Go struct fields.
// Hyprland returns a lot of info - we capture what's useful.
type HyprlandWindow struct {
	Address      string `json:"address"`
	At           [2]int `json:"at"`           // [x, y] position
	Size         [2]int `json:"size"`         // [width, height]
	Workspace    HyprlandWorkspace `json:"workspace"`
	Floating     bool   `json:"floating"`
	Class        string `json:"class"`        // Application class (e.g., "firefox")
	Title        string `json:"title"`        // Window title
	PID          int    `json:"pid"`
	InitialClass string `json:"initialClass"`
	InitialTitle string `json:"initialTitle"`
	Mapped       bool   `json:"mapped"`
	Fullscreen   int    `json:"fullscreen"`   // 0, 1, or 2 for different fullscreen modes
	FocusHistoryID int  `json:"focusHistoryID"`
}

type HyprlandWorkspace struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// captureHyprland uses hyprctl to get window info on Hyprland.
func (c *Capturer) captureHyprland(ctx context.Context) (*capture.Result, error) {
	// exec.CommandContext creates a command that will be killed if ctx is cancelled
	// This is important for graceful shutdown - if the daemon stops, we don't
	// want hyprctl hanging around.
	cmd := exec.CommandContext(ctx, "hyprctl", "activewindow", "-j")

	// Output() runs the command and captures stdout
	output, err := cmd.Output()
	if err != nil {
		// Wrap the error with context - this is a Go best practice
		// fmt.Errorf with %w creates an error chain that can be unwrapped
		return nil, fmt.Errorf("hyprctl failed: %w", err)
	}

	// Parse the JSON response
	var window HyprlandWindow
	if err := json.Unmarshal(output, &window); err != nil {
		return nil, fmt.Errorf("failed to parse hyprctl output: %w", err)
	}

	// Build the capture result
	result := capture.NewResult("window")
	result.TextData = window.Title

	// Store detailed info in metadata
	result.SetMetadata("app_class", window.Class)
	result.SetMetadata("window_title", window.Title)
	result.SetMetadata("pid", fmt.Sprintf("%d", window.PID))
	result.SetMetadata("workspace_id", fmt.Sprintf("%d", window.Workspace.ID))
	result.SetMetadata("workspace_name", window.Workspace.Name)
	result.SetMetadata("floating", fmt.Sprintf("%t", window.Floating))
	result.SetMetadata("fullscreen", fmt.Sprintf("%d", window.Fullscreen))
	result.SetMetadata("position", fmt.Sprintf("%d,%d", window.At[0], window.At[1]))
	result.SetMetadata("size", fmt.Sprintf("%dx%d", window.Size[0], window.Size[1]))

	// Try to extract URL if it's a browser
	// Browsers often put the URL or page title in the window title
	if isBrowser(window.Class) {
		// The title format is usually "Page Title - Browser Name"
		// We store the full title; URL extraction can be improved later
		result.SetMetadata("is_browser", "true")
	}

	return result, nil
}

// captureX11 captures window info on X11 (placeholder for now)
func (c *Capturer) captureX11(ctx context.Context) (*capture.Result, error) {
	// TODO: Implement using xdotool + xprop
	return nil, fmt.Errorf("X11 window capture not implemented yet")
}

// captureMacOS captures window info on macOS (placeholder for now)
func (c *Capturer) captureMacOS(ctx context.Context) (*capture.Result, error) {
	// TODO: Implement using Accessibility APIs or AppleScript
	return nil, fmt.Errorf("macOS window capture not implemented yet")
}

// isBrowser checks if an app class is a known browser.
func isBrowser(class string) bool {
	browsers := []string{
		"firefox",
		"chromium",
		"chrome",
		"google-chrome",
		"brave",
		"brave-browser",
		"microsoft-edge",
		"safari",
		"opera",
		"vivaldi",
		"librewolf",
		"zen",
		"zen-browser",
	}

	// Convert to lowercase for comparison
	classLower := class
	for _, b := range browsers {
		if classLower == b {
			return true
		}
	}
	return false
}
