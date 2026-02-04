// Package capture defines the core capture interfaces and types.
//
// The design principle: every capture source (screen, window, git, clipboard)
// implements the same Capturer interface. This lets us treat them uniformly
// in the capture manager.
package capture

import (
	"context"
	"time"
)

// Capturer is the interface that all capture sources must implement.
// In Go, interfaces are defined by the methods they require, not by
// explicit "implements" declarations. If a type has these methods,
// it automatically implements Capturer.
type Capturer interface {
	// Name returns the capturer identifier (e.g., "screen", "window", "git")
	Name() string

	// Available checks if this capturer can run on the current system.
	// For example, window capture needs hyprctl on Hyprland.
	Available() bool

	// Capture takes a snapshot and returns the capture result.
	// The context allows for cancellation (if the daemon is shutting down).
	Capture(ctx context.Context) (*Result, error)
}

// Result holds the output of a capture operation.
// Different capturers will populate different fields.
type Result struct {
	// Source identifies which capturer produced this (e.g., "window", "screen")
	Source string

	// Timestamp when the capture was taken
	Timestamp time.Time

	// RawData holds binary data (screenshots, audio, etc.)
	// This will be nil for text-only captures like window info.
	RawData []byte

	// TextData holds text content (window title, clipboard text, OCR result)
	TextData string

	// Metadata holds source-specific key-value data
	// For window capture: app class, window title, PID, etc.
	// For git capture: repo, branch, commit hash, etc.
	Metadata map[string]string
}

// NewResult creates a Result with the timestamp set to now.
// This is a common Go pattern - a constructor function.
func NewResult(source string) *Result {
	return &Result{
		Source:    source,
		Timestamp: time.Now(),
		Metadata:  make(map[string]string),
	}
}

// SetMetadata is a helper to set a metadata key-value pair.
// It returns the Result so you can chain calls: result.SetMetadata("a", "1").SetMetadata("b", "2")
func (r *Result) SetMetadata(key, value string) *Result {
	if r.Metadata == nil {
		r.Metadata = make(map[string]string)
	}
	r.Metadata[key] = value
	return r
}
