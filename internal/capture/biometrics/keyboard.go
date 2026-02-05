// Package biometrics - keyboard dynamics tracking for stress detection.
//
// This file implements keystroke timing analysis using evdev.
// Requires the user to be in the 'input' group:
//
//	sudo usermod -aG input $USER
//
// We track:
// - Key hold time (press to release)
// - Flight time (release to next press)
// - Error keys (backspace, delete)
// - Typing pauses
//
// Privacy note: We do NOT log which keys are pressed, only timing data.
package biometrics

import (
	"bufio"
	"context"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Atharva-Kanherkar/mnemosyne/internal/platform"
)

// Linux input event structure (from linux/input.h)
// struct input_event {
//     struct timeval time;
//     __u16 type;
//     __u16 code;
//     __s32 value;
// }

const (
	evKey      = 0x01 // EV_KEY
	keyPress   = 1    // Key pressed
	keyRelease = 0    // Key released

	// Error keys we track
	keyBackspace = 14
	keyDelete    = 111
)

// KeyboardTracker tracks keystroke dynamics.
type KeyboardTracker struct {
	platform   *platform.Platform
	analyzer   *Analyzer
	running    bool
	stopCh     chan struct{}
	devicePath string

	// State tracking
	pressedKeys map[uint16]time.Time // Key code -> press time
	lastRelease time.Time
}

// NewKeyboardTracker creates a new keyboard tracker.
func NewKeyboardTracker(plat *platform.Platform, analyzer *Analyzer) *KeyboardTracker {
	return &KeyboardTracker{
		platform:    plat,
		analyzer:    analyzer,
		stopCh:      make(chan struct{}),
		pressedKeys: make(map[uint16]time.Time),
	}
}

// Available checks if keyboard tracking is available.
func (k *KeyboardTracker) Available() bool {
	// Find keyboard device
	device := k.findKeyboardDevice()
	if device == "" {
		return false
	}

	// Check if we can read it
	f, err := os.Open(device)
	if err != nil {
		return false
	}
	f.Close()

	k.devicePath = device
	return true
}

// findKeyboardDevice finds the primary keyboard input device.
func (k *KeyboardTracker) findKeyboardDevice() string {
	// Look in /dev/input/by-id for keyboard
	matches, _ := filepath.Glob("/dev/input/by-id/*-kbd")
	if len(matches) > 0 {
		return matches[0]
	}

	// Fallback: scan /dev/input/event* for keyboards
	// Check /proc/bus/input/devices
	f, err := os.Open("/proc/bus/input/devices")
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var currentHandler string
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "H: Handlers=") {
			// Extract event handler
			parts := strings.Fields(line)
			for _, p := range parts {
				if strings.HasPrefix(p, "event") {
					currentHandler = "/dev/input/" + p
				}
			}
		}

		if strings.Contains(line, "EV=120013") || // Typical keyboard
			strings.Contains(strings.ToLower(line), "keyboard") {
			if currentHandler != "" {
				return currentHandler
			}
		}

		if line == "" {
			currentHandler = ""
		}
	}

	return ""
}

// Start begins keyboard tracking.
func (k *KeyboardTracker) Start(ctx context.Context) {
	if k.running || k.devicePath == "" {
		return
	}
	k.running = true

	go k.trackLoop(ctx)
}

// Stop stops keyboard tracking.
func (k *KeyboardTracker) Stop() {
	if !k.running {
		return
	}
	close(k.stopCh)
	k.running = false
}

// trackLoop reads keyboard events.
func (k *KeyboardTracker) trackLoop(ctx context.Context) {
	f, err := os.Open(k.devicePath)
	if err != nil {
		return
	}
	defer f.Close()

	// input_event is 24 bytes on 64-bit Linux
	// timeval (16 bytes) + type (2) + code (2) + value (4)
	eventSize := 24
	buf := make([]byte, eventSize)

	for {
		select {
		case <-ctx.Done():
			return
		case <-k.stopCh:
			return
		default:
			// Read with timeout would be better, but for simplicity
			// we just read (will block)
			n, err := f.Read(buf)
			if err != nil || n != eventSize {
				continue
			}

			k.processEvent(buf)
		}
	}
}

// processEvent processes a single input event.
func (k *KeyboardTracker) processEvent(buf []byte) {
	// Parse the event
	// Skip timeval (16 bytes), read type, code, value
	eventType := binary.LittleEndian.Uint16(buf[16:18])
	code := binary.LittleEndian.Uint16(buf[18:20])
	value := int32(binary.LittleEndian.Uint32(buf[20:24]))

	// We only care about key events
	if eventType != evKey {
		return
	}

	now := time.Now()

	switch value {
	case keyPress:
		k.pressedKeys[code] = now

	case keyRelease:
		// Calculate hold time
		if pressTime, ok := k.pressedKeys[code]; ok {
			holdTime := now.Sub(pressTime)
			delete(k.pressedKeys, code)

			// Is this an error key?
			isError := code == keyBackspace || code == keyDelete

			// Record the keystroke (timing only, not which key)
			k.analyzer.RecordKeystroke(holdTime, isError)
		}

		k.lastRelease = now
	}
}

// GetStatus returns tracking status info.
func (k *KeyboardTracker) GetStatus() map[string]string {
	status := map[string]string{
		"available": "false",
		"device":    "",
		"running":   "false",
	}

	if k.Available() {
		status["available"] = "true"
		status["device"] = k.devicePath
	}

	if k.running {
		status["running"] = "true"
	}

	return status
}
