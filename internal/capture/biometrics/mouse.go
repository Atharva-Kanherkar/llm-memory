// Package biometrics - mouse tracking for stress detection.
//
// This file implements high-frequency mouse position tracking.
// On Hyprland, we use `hyprctl cursorpos` to get cursor position.
// We sample at ~20Hz to capture jitter and movement patterns.
package biometrics

import (
	"bytes"
	"context"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/Atharva-Kanherkar/mnemosyne/internal/platform"
)

// MouseTracker tracks cursor position at high frequency.
type MouseTracker struct {
	platform *platform.Platform
	analyzer *Analyzer
	running  bool
	stopCh   chan struct{}
}

// NewMouseTracker creates a new mouse tracker.
func NewMouseTracker(plat *platform.Platform, analyzer *Analyzer) *MouseTracker {
	return &MouseTracker{
		platform: plat,
		analyzer: analyzer,
		stopCh:   make(chan struct{}),
	}
}

// Start begins mouse tracking.
func (m *MouseTracker) Start(ctx context.Context) {
	if m.running {
		return
	}
	m.running = true

	go m.trackLoop(ctx)
}

// Stop stops mouse tracking.
func (m *MouseTracker) Stop() {
	if !m.running {
		return
	}
	close(m.stopCh)
	m.running = false
}

// trackLoop polls cursor position at high frequency.
func (m *MouseTracker) trackLoop(ctx context.Context) {
	ticker := time.NewTicker(50 * time.Millisecond) // 20Hz
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case <-ticker.C:
			x, y, err := m.getCursorPosition(ctx)
			if err == nil {
				m.analyzer.RecordMousePosition(x, y)
			}
		}
	}
}

// getCursorPosition gets current cursor position from Hyprland.
func (m *MouseTracker) getCursorPosition(ctx context.Context) (int, int, error) {
	// hyprctl cursorpos returns "X, Y"
	cmd := exec.CommandContext(ctx, "hyprctl", "cursorpos")
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return 0, 0, err
	}

	// Parse "1234, 567"
	parts := strings.Split(strings.TrimSpace(out.String()), ",")
	if len(parts) != 2 {
		return 0, 0, nil
	}

	x, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, err
	}

	y, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, err
	}

	return x, y, nil
}
