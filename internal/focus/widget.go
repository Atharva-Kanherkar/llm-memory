package focus

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// WidgetState holds the current state for the widget display.
type WidgetState struct {
	Active       bool      `json:"active"`
	ModeName     string    `json:"mode_name"`
	Purpose      string    `json:"purpose"`
	StartedAt    time.Time `json:"started_at"`
	ElapsedSecs  int       `json:"elapsed_secs"`
	BlocksCount  int       `json:"blocks_count"`
	LastDecision string    `json:"last_decision"` // "allowed: VS Code" or "blocked: YouTube"
	LastApp      string    `json:"last_app"`
	LastAction   string    `json:"last_action"` // "allow" or "block"
	UpdatedAt    time.Time `json:"updated_at"`
}

// WidgetBroadcaster broadcasts focus state for widgets to read.
type WidgetBroadcaster struct {
	statePath string
	state     WidgetState
	mu        sync.RWMutex
}

// NewWidgetBroadcaster creates a new widget broadcaster.
func NewWidgetBroadcaster(dataDir string) *WidgetBroadcaster {
	return &WidgetBroadcaster{
		statePath: filepath.Join(dataDir, "focus_widget.json"),
	}
}

// UpdateState updates and broadcasts the widget state.
func (w *WidgetBroadcaster) UpdateState(state WidgetState) {
	w.mu.Lock()
	defer w.mu.Unlock()

	state.UpdatedAt = time.Now()
	if state.Active && !state.StartedAt.IsZero() {
		state.ElapsedSecs = int(time.Since(state.StartedAt).Seconds())
	}
	w.state = state

	// Write to file for external widgets to read
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(w.statePath, data, 0644)
}

// RecordDecision records a decision for display.
func (w *WidgetBroadcaster) RecordDecision(app, title, action string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.state.LastApp = app
	w.state.LastAction = action
	if action == "allow" {
		w.state.LastDecision = fmt.Sprintf("✓ %s", truncateStr(title, 30))
	} else {
		w.state.LastDecision = fmt.Sprintf("✗ %s", truncateStr(title, 30))
	}
	w.state.UpdatedAt = time.Now()

	// Write updated state
	data, _ := json.MarshalIndent(w.state, "", "  ")
	os.WriteFile(w.statePath, data, 0644)
}

// IncrementBlocks increments the block counter.
func (w *WidgetBroadcaster) IncrementBlocks() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.state.BlocksCount++
	w.state.UpdatedAt = time.Now()

	data, _ := json.MarshalIndent(w.state, "", "  ")
	os.WriteFile(w.statePath, data, 0644)
}

// Clear clears the widget state (when focus mode ends).
func (w *WidgetBroadcaster) Clear() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.state = WidgetState{
		Active:    false,
		UpdatedAt: time.Now(),
	}

	data, _ := json.MarshalIndent(w.state, "", "  ")
	os.WriteFile(w.statePath, data, 0644)
}

// GetState returns the current state.
func (w *WidgetBroadcaster) GetState() WidgetState {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.state
}

// ReadWidgetState reads the widget state from the file (for external readers).
func ReadWidgetState(dataDir string) (*WidgetState, error) {
	path := filepath.Join(dataDir, "focus_widget.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var state WidgetState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	// Recalculate elapsed time
	if state.Active && !state.StartedAt.IsZero() {
		state.ElapsedSecs = int(time.Since(state.StartedAt).Seconds())
	}

	return &state, nil
}

func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
