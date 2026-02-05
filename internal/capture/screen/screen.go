// Package screen provides screen capture functionality.
//
// On Hyprland/Wayland, we use grim to capture screenshots.
// grim can capture:
// - All outputs combined (default)
// - A specific output (monitor) with -o flag
// - A region with -g flag
//
// Screenshots are captured as PNG and can be OCR'd for text extraction.
package screen

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image/png"
	"os/exec"

	"github.com/Atharva-Kanherkar/mnemosyne/internal/capture"
	"github.com/Atharva-Kanherkar/mnemosyne/internal/platform"
)

// Capturer captures screenshots.
type Capturer struct {
	platform *platform.Platform

	// CaptureAllMonitors determines whether to capture all monitors or just focused
	CaptureAllMonitors bool

	// Quality is the PNG compression level (not used by grim, but for future)
	Quality int
}

// New creates a new screen Capturer.
func New(plat *platform.Platform) *Capturer {
	return &Capturer{
		platform:           plat,
		CaptureAllMonitors: false, // Default to focused monitor only (saves space)
		Quality:            90,
	}
}

// Name returns the capturer identifier.
func (c *Capturer) Name() string {
	return "screen"
}

// Available checks if screen capture is possible.
func (c *Capturer) Available() bool {
	return c.platform.CanCaptureScreen()
}

// Capture takes a screenshot.
func (c *Capturer) Capture(ctx context.Context) (*capture.Result, error) {
	switch c.platform.DisplayServer {
	case platform.DisplayServerHyprland, platform.DisplayServerSway, platform.DisplayServerWayland:
		return c.captureGrim(ctx)
	case platform.DisplayServerX11:
		return c.captureScrot(ctx)
	case platform.DisplayServerMacOS:
		return c.captureMacOS(ctx)
	default:
		return nil, fmt.Errorf("unsupported display server: %s", c.platform.DisplayServer)
	}
}

// Monitor represents a display output.
type Monitor struct {
	ID      int     `json:"id"`
	Name    string  `json:"name"`
	Width   int     `json:"width"`
	Height  int     `json:"height"`
	X       int     `json:"x"`
	Y       int     `json:"y"`
	Scale   float64 `json:"scale"`
	Focused bool    `json:"focused"`
}

// GetMonitors returns list of available monitors on Hyprland.
func (c *Capturer) GetMonitors(ctx context.Context) ([]Monitor, error) {
	if c.platform.DisplayServer != platform.DisplayServerHyprland {
		return nil, fmt.Errorf("GetMonitors only supported on Hyprland")
	}

	cmd := exec.CommandContext(ctx, "hyprctl", "monitors", "-j")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("hyprctl monitors failed: %w", err)
	}

	var monitors []Monitor
	if err := json.Unmarshal(output, &monitors); err != nil {
		return nil, fmt.Errorf("failed to parse monitors: %w", err)
	}

	return monitors, nil
}

// captureGrim captures screenshot using grim (Wayland).
func (c *Capturer) captureGrim(ctx context.Context) (*capture.Result, error) {
	var cmd *exec.Cmd

	if c.CaptureAllMonitors {
		// Capture all monitors combined
		// grim - outputs PNG to stdout
		cmd = exec.CommandContext(ctx, "grim", "-")
	} else {
		// Capture only the focused monitor
		// First, find which monitor is focused
		monitors, err := c.GetMonitors(ctx)
		if err != nil {
			// Fall back to capturing all if we can't get monitors
			cmd = exec.CommandContext(ctx, "grim", "-")
		} else {
			focusedMonitor := ""
			for _, m := range monitors {
				if m.Focused {
					focusedMonitor = m.Name
					break
				}
			}
			if focusedMonitor != "" {
				cmd = exec.CommandContext(ctx, "grim", "-o", focusedMonitor, "-")
			} else {
				cmd = exec.CommandContext(ctx, "grim", "-")
			}
		}
	}

	// Capture stdout (the PNG data)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("grim failed: %w (stderr: %s)", err, stderr.String())
	}

	pngData := stdout.Bytes()

	// Get image dimensions
	width, height := 0, 0
	if img, err := png.Decode(bytes.NewReader(pngData)); err == nil {
		bounds := img.Bounds()
		width = bounds.Dx()
		height = bounds.Dy()
	}

	result := capture.NewResult("screen")
	result.RawData = pngData
	result.SetMetadata("format", "png")
	result.SetMetadata("width", fmt.Sprintf("%d", width))
	result.SetMetadata("height", fmt.Sprintf("%d", height))
	result.SetMetadata("size_bytes", fmt.Sprintf("%d", len(pngData)))

	return result, nil
}

// captureScrot captures screenshot using scrot (X11).
func (c *Capturer) captureScrot(ctx context.Context) (*capture.Result, error) {
	// scrot doesn't support stdout output, so we need a temp file
	// For now, return not implemented
	return nil, fmt.Errorf("X11 screen capture not implemented yet")
}

// captureMacOS captures screenshot on macOS.
func (c *Capturer) captureMacOS(ctx context.Context) (*capture.Result, error) {
	return nil, fmt.Errorf("macOS screen capture not implemented yet")
}

// CaptureRegion captures a specific region of the screen.
// Uses slurp for region selection on Wayland.
func (c *Capturer) CaptureRegion(ctx context.Context, x, y, width, height int) (*capture.Result, error) {
	if !c.platform.HasSlurp {
		return nil, fmt.Errorf("slurp not available for region capture")
	}

	geometry := fmt.Sprintf("%d,%d %dx%d", x, y, width, height)
	cmd := exec.CommandContext(ctx, "grim", "-g", geometry, "-")

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("grim region capture failed: %w", err)
	}

	result := capture.NewResult("screen")
	result.RawData = stdout.Bytes()
	result.SetMetadata("format", "png")
	result.SetMetadata("region", geometry)

	return result, nil
}

// Helper to decode PNG dimensions without fully decoding the image
func decodePNGDimensions(data []byte) (width, height int, err error) {
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return 0, 0, err
	}
	bounds := img.Bounds()
	return bounds.Dx(), bounds.Dy(), nil
}
