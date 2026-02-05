// Package clipboard provides clipboard content capture.
//
// On Wayland, we use wl-paste to get clipboard contents.
// On X11, we'd use xclip.
//
// We only capture text content for now (not images).
// Clipboard is useful for tracking what you're copying/working with.
package clipboard

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os/exec"
	"strings"

	"github.com/Atharva-Kanherkar/mnemosyne/internal/capture"
	"github.com/Atharva-Kanherkar/mnemosyne/internal/platform"
)

// Capturer captures clipboard contents.
type Capturer struct {
	platform *platform.Platform

	// lastHash stores hash of last captured content to avoid duplicates
	lastHash string

	// MaxLength is the maximum text length to capture (to avoid huge pastes)
	MaxLength int
}

// New creates a new clipboard Capturer.
func New(plat *platform.Platform) *Capturer {
	return &Capturer{
		platform:  plat,
		MaxLength: 10000, // 10KB max
	}
}

// Name returns the capturer identifier.
func (c *Capturer) Name() string {
	return "clipboard"
}

// Available checks if clipboard capture is possible.
func (c *Capturer) Available() bool {
	if c.platform.IsWayland() {
		return c.platform.HasWlPaste
	}
	return c.platform.HasXclip
}

// Capture gets the current clipboard contents.
func (c *Capturer) Capture(ctx context.Context) (*capture.Result, error) {
	if c.platform.IsWayland() {
		return c.captureWayland(ctx)
	}
	return c.captureX11(ctx)
}

// captureWayland uses wl-paste to get clipboard on Wayland.
func (c *Capturer) captureWayland(ctx context.Context) (*capture.Result, error) {
	// First check what types are available
	typesCmd := exec.CommandContext(ctx, "wl-paste", "--list-types")
	typesOutput, _ := typesCmd.Output()
	types := strings.TrimSpace(string(typesOutput))

	// Only capture if text is available
	if !strings.Contains(types, "text/plain") && !strings.Contains(types, "UTF8_STRING") {
		// Clipboard doesn't contain text
		result := capture.NewResult("clipboard")
		result.SetMetadata("has_text", "false")
		result.SetMetadata("available_types", types)
		return result, nil
	}

	// Get text content
	cmd := exec.CommandContext(ctx, "wl-paste", "-n") // -n = no newline at end
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Empty clipboard or error
		result := capture.NewResult("clipboard")
		result.SetMetadata("has_text", "false")
		result.SetMetadata("error", stderr.String())
		return result, nil
	}

	text := stdout.String()

	// Check if content changed
	hash := hashContent(text)
	if hash == c.lastHash {
		// Content unchanged - return result with flag
		result := capture.NewResult("clipboard")
		result.SetMetadata("changed", "false")
		return result, nil
	}
	c.lastHash = hash

	// Truncate if too long
	if len(text) > c.MaxLength {
		text = text[:c.MaxLength] + "... [truncated]"
	}

	result := capture.NewResult("clipboard")
	result.TextData = text
	result.SetMetadata("has_text", "true")
	result.SetMetadata("changed", "true")
	result.SetMetadata("length", fmt.Sprintf("%d", len(text)))
	result.SetMetadata("hash", hash[:16]) // Short hash for dedup

	// Try to detect content type
	contentType := detectContentType(text)
	result.SetMetadata("content_type", contentType)

	return result, nil
}

// captureX11 uses xclip to get clipboard on X11.
func (c *Capturer) captureX11(ctx context.Context) (*capture.Result, error) {
	cmd := exec.CommandContext(ctx, "xclip", "-selection", "clipboard", "-o")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		result := capture.NewResult("clipboard")
		result.SetMetadata("has_text", "false")
		return result, nil
	}

	text := stdout.String()
	hash := hashContent(text)

	if hash == c.lastHash {
		result := capture.NewResult("clipboard")
		result.SetMetadata("changed", "false")
		return result, nil
	}
	c.lastHash = hash

	if len(text) > c.MaxLength {
		text = text[:c.MaxLength] + "... [truncated]"
	}

	result := capture.NewResult("clipboard")
	result.TextData = text
	result.SetMetadata("has_text", "true")
	result.SetMetadata("changed", "true")
	result.SetMetadata("length", fmt.Sprintf("%d", len(text)))

	return result, nil
}

// hashContent creates a hash of the content for deduplication.
func hashContent(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:])
}

// detectContentType tries to guess what kind of content is in the clipboard.
func detectContentType(text string) string {
	trimmed := strings.TrimSpace(text)

	// Check for URLs
	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		return "url"
	}

	// Check for file paths
	if strings.HasPrefix(trimmed, "/") || strings.HasPrefix(trimmed, "~/") {
		return "path"
	}

	// Check for code patterns
	if strings.Contains(trimmed, "func ") || strings.Contains(trimmed, "function ") {
		return "code"
	}
	if strings.Contains(trimmed, "import ") || strings.Contains(trimmed, "package ") {
		return "code"
	}
	if strings.Contains(trimmed, "class ") || strings.Contains(trimmed, "def ") {
		return "code"
	}

	// Check for JSON
	if (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) ||
		(strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")) {
		return "json"
	}

	// Check for command
	if strings.HasPrefix(trimmed, "git ") || strings.HasPrefix(trimmed, "sudo ") ||
		strings.HasPrefix(trimmed, "npm ") || strings.HasPrefix(trimmed, "go ") {
		return "command"
	}

	// Default to text
	if len(trimmed) > 100 {
		return "text-long"
	}
	return "text"
}
