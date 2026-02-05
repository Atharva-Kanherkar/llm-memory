// Package biometrics provides stress and anxiety detection through behavioral analysis.
//
// Based on research from:
// - CMU: Keystroke dynamics and stress detection
// - Behavior Research Methods: Mouse tracking for stress measurement
// - IEEE: Emotion recognition from keyboard/mouse dynamics
//
// Key findings from research:
// - Typing pauses (count, duration, variance) correlate strongly with stress
// - 70% of users show decreased typing speed when anxious
// - Mouse movement jitter increases under stress
// - Window switching frequency indicates task fragmentation/anxiety
package biometrics

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/Atharva-Kanherkar/mnemosyne/internal/capture"
	"github.com/Atharva-Kanherkar/mnemosyne/internal/platform"
)

// StressLevel represents detected stress/anxiety level.
type StressLevel int

const (
	StressLevelCalm StressLevel = iota
	StressLevelNormal
	StressLevelElevated
	StressLevelHigh
	StressLevelAnxious
)

func (s StressLevel) String() string {
	switch s {
	case StressLevelCalm:
		return "calm"
	case StressLevelNormal:
		return "normal"
	case StressLevelElevated:
		return "elevated"
	case StressLevelHigh:
		return "high"
	case StressLevelAnxious:
		return "anxious"
	default:
		return "unknown"
	}
}

// MouseMetrics tracks mouse movement patterns.
type MouseMetrics struct {
	// Movement analysis
	Positions       []Position // Recent positions for trajectory analysis
	AvgSpeed        float64    // Pixels per second
	SpeedVariance   float64    // Variance in movement speed
	Jitter          float64    // High-frequency movement noise (0-1)
	DirectnessRatio float64    // Actual distance / straight line distance
	Hesitations     int        // Number of pauses mid-movement

	// Click patterns
	ClicksPerMinute float64
	DoubleClickRate float64

	// Scroll behavior
	ScrollSpeed    float64
	ScrollVariance float64
}

// Position represents a cursor position with timestamp.
type Position struct {
	X, Y      int
	Timestamp time.Time
}

// KeystrokeMetrics tracks typing patterns.
type KeystrokeMetrics struct {
	// Timing metrics (in milliseconds)
	AvgHoldTime      float64 // How long keys are pressed
	HoldTimeVariance float64
	AvgFlightTime    float64 // Time between key release and next press
	FlightVariance   float64

	// Pause analysis (research shows this is most predictive)
	PauseCount       int     // Number of typing pauses > 500ms
	AvgPauseDuration float64 // Mean pause duration
	PauseDurationSD  float64 // Standard deviation of pauses

	// Speed and errors
	KeysPerMinute float64
	ErrorRate     float64 // Backspace/delete frequency
	TypingBursts  int     // Number of fast typing bursts
}

// ContextMetrics tracks application context patterns.
type ContextMetrics struct {
	// Window switching (indicator of task fragmentation)
	SwitchesPerMinute  float64
	AvgTimePerWindow   float64 // Seconds
	UniqueWindowsCount int
	RapidSwitches      int // Switches < 5 seconds apart

	// Idle patterns
	IdlePeriodsCount  int
	AvgIdleDuration   float64
	LongestIdlePeriod float64
}

// StressSnapshot captures a moment's stress indicators.
type StressSnapshot struct {
	Timestamp     time.Time
	Mouse         MouseMetrics
	Keystrokes    KeystrokeMetrics
	Context       ContextMetrics
	OverallScore  float64 // 0-100
	Level         StressLevel
	TopIndicators []string // What's contributing most to stress
}

// Analyzer tracks behavioral patterns and detects stress.
type Analyzer struct {
	platform *platform.Platform
	mu       sync.RWMutex

	// Rolling windows for analysis
	mousePositions  []Position
	windowSwitches  []time.Time
	keystrokeEvents []keystrokeEvent
	idlePeriods     []idlePeriod

	// Configuration
	windowSize     time.Duration // How far back to analyze
	sampleInterval time.Duration // How often to sample mouse position

	// Baseline (personalized calibration)
	baselineSpeed float64
	baselineKPM   float64
	baselineSet   bool
}

type keystrokeEvent struct {
	Timestamp time.Time
	HoldTime  time.Duration
	IsError   bool // backspace, delete
}

type idlePeriod struct {
	Start    time.Time
	Duration time.Duration
}

// NewAnalyzer creates a new stress analyzer.
func NewAnalyzer(plat *platform.Platform) *Analyzer {
	return &Analyzer{
		platform:        plat,
		mousePositions:  make([]Position, 0, 1000),
		windowSwitches:  make([]time.Time, 0, 100),
		keystrokeEvents: make([]keystrokeEvent, 0, 1000),
		idlePeriods:     make([]idlePeriod, 0, 100),
		windowSize:      5 * time.Minute,
		sampleInterval:  50 * time.Millisecond, // 20Hz mouse sampling
	}
}

// RecordMousePosition records a cursor position.
func (a *Analyzer) RecordMousePosition(x, y int) {
	a.mu.Lock()
	defer a.mu.Unlock()

	pos := Position{X: x, Y: y, Timestamp: time.Now()}
	a.mousePositions = append(a.mousePositions, pos)

	// Keep only recent positions
	a.pruneOldData()
}

// RecordWindowSwitch records a window switch event.
func (a *Analyzer) RecordWindowSwitch() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.windowSwitches = append(a.windowSwitches, time.Now())
}

// RecordKeystroke records a keystroke event.
func (a *Analyzer) RecordKeystroke(holdTime time.Duration, isError bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.keystrokeEvents = append(a.keystrokeEvents, keystrokeEvent{
		Timestamp: time.Now(),
		HoldTime:  holdTime,
		IsError:   isError,
	})
}

// RecordIdlePeriod records an idle period.
func (a *Analyzer) RecordIdlePeriod(duration time.Duration) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.idlePeriods = append(a.idlePeriods, idlePeriod{
		Start:    time.Now().Add(-duration),
		Duration: duration,
	})
}

// pruneOldData removes data outside the analysis window.
func (a *Analyzer) pruneOldData() {
	cutoff := time.Now().Add(-a.windowSize)

	// Prune mouse positions
	newPositions := make([]Position, 0, len(a.mousePositions))
	for _, p := range a.mousePositions {
		if p.Timestamp.After(cutoff) {
			newPositions = append(newPositions, p)
		}
	}
	a.mousePositions = newPositions

	// Prune window switches
	newSwitches := make([]time.Time, 0, len(a.windowSwitches))
	for _, t := range a.windowSwitches {
		if t.After(cutoff) {
			newSwitches = append(newSwitches, t)
		}
	}
	a.windowSwitches = newSwitches

	// Prune keystroke events
	newKeystrokes := make([]keystrokeEvent, 0, len(a.keystrokeEvents))
	for _, k := range a.keystrokeEvents {
		if k.Timestamp.After(cutoff) {
			newKeystrokes = append(newKeystrokes, k)
		}
	}
	a.keystrokeEvents = newKeystrokes
}

// Analyze computes current stress metrics.
func (a *Analyzer) Analyze() *StressSnapshot {
	a.mu.RLock()
	defer a.mu.RUnlock()

	snapshot := &StressSnapshot{
		Timestamp: time.Now(),
	}

	// Calculate mouse metrics
	snapshot.Mouse = a.calculateMouseMetrics()

	// Calculate keystroke metrics
	snapshot.Keystrokes = a.calculateKeystrokeMetrics()

	// Calculate context metrics
	snapshot.Context = a.calculateContextMetrics()

	// Calculate overall stress score
	snapshot.OverallScore, snapshot.TopIndicators = a.calculateStressScore(snapshot)
	snapshot.Level = scoreToLevel(snapshot.OverallScore)

	return snapshot
}

// calculateMouseMetrics analyzes mouse movement patterns.
func (a *Analyzer) calculateMouseMetrics() MouseMetrics {
	m := MouseMetrics{}

	if len(a.mousePositions) < 2 {
		return m
	}

	// Calculate speeds between consecutive points
	var speeds []float64
	var totalDistance float64
	var directDistance float64

	first := a.mousePositions[0]
	last := a.mousePositions[len(a.mousePositions)-1]

	for i := 1; i < len(a.mousePositions); i++ {
		prev := a.mousePositions[i-1]
		curr := a.mousePositions[i]

		dx := float64(curr.X - prev.X)
		dy := float64(curr.Y - prev.Y)
		dist := math.Sqrt(dx*dx + dy*dy)
		totalDistance += dist

		dt := curr.Timestamp.Sub(prev.Timestamp).Seconds()
		if dt > 0 {
			speed := dist / dt
			speeds = append(speeds, speed)
		}

		// Check for hesitations (very slow movement after faster movement)
		if len(speeds) > 1 && speeds[len(speeds)-1] < 10 && speeds[len(speeds)-2] > 100 {
			m.Hesitations++
		}
	}

	// Direct distance (straight line from first to last)
	dx := float64(last.X - first.X)
	dy := float64(last.Y - first.Y)
	directDistance = math.Sqrt(dx*dx + dy*dy)

	// Average speed
	if len(speeds) > 0 {
		var sum float64
		for _, s := range speeds {
			sum += s
		}
		m.AvgSpeed = sum / float64(len(speeds))

		// Speed variance
		var variance float64
		for _, s := range speeds {
			diff := s - m.AvgSpeed
			variance += diff * diff
		}
		m.SpeedVariance = variance / float64(len(speeds))
	}

	// Directness ratio (1.0 = perfectly direct, higher = more wandering)
	if directDistance > 0 {
		m.DirectnessRatio = totalDistance / directDistance
	} else {
		m.DirectnessRatio = 1.0
	}

	// Jitter: high-frequency direction changes
	m.Jitter = a.calculateJitter()

	return m
}

// calculateJitter measures high-frequency movement noise.
func (a *Analyzer) calculateJitter() float64 {
	if len(a.mousePositions) < 3 {
		return 0
	}

	var directionChanges int
	var totalMovements int

	for i := 2; i < len(a.mousePositions); i++ {
		// Calculate direction vectors
		dx1 := a.mousePositions[i-1].X - a.mousePositions[i-2].X
		dy1 := a.mousePositions[i-1].Y - a.mousePositions[i-2].Y
		dx2 := a.mousePositions[i].X - a.mousePositions[i-1].X
		dy2 := a.mousePositions[i].Y - a.mousePositions[i-1].Y

		// Skip if not moving
		if (dx1 == 0 && dy1 == 0) || (dx2 == 0 && dy2 == 0) {
			continue
		}

		totalMovements++

		// Check for direction change using dot product
		dot := float64(dx1*dx2 + dy1*dy2)
		mag1 := math.Sqrt(float64(dx1*dx1 + dy1*dy1))
		mag2 := math.Sqrt(float64(dx2*dx2 + dy2*dy2))

		if mag1 > 0 && mag2 > 0 {
			cosAngle := dot / (mag1 * mag2)
			// Direction change if angle > 90 degrees
			if cosAngle < 0 {
				directionChanges++
			}
		}
	}

	if totalMovements == 0 {
		return 0
	}

	// Normalize to 0-1
	return float64(directionChanges) / float64(totalMovements)
}

// calculateKeystrokeMetrics analyzes typing patterns.
func (a *Analyzer) calculateKeystrokeMetrics() KeystrokeMetrics {
	k := KeystrokeMetrics{}

	if len(a.keystrokeEvents) < 2 {
		return k
	}

	// Hold times
	var holdTimes []float64
	var errors int
	for _, e := range a.keystrokeEvents {
		holdTimes = append(holdTimes, float64(e.HoldTime.Milliseconds()))
		if e.IsError {
			errors++
		}
	}

	// Average hold time
	var sumHold float64
	for _, h := range holdTimes {
		sumHold += h
	}
	k.AvgHoldTime = sumHold / float64(len(holdTimes))

	// Hold time variance
	var varianceHold float64
	for _, h := range holdTimes {
		diff := h - k.AvgHoldTime
		varianceHold += diff * diff
	}
	k.HoldTimeVariance = varianceHold / float64(len(holdTimes))

	// Flight times (time between keystrokes)
	var flightTimes []float64
	for i := 1; i < len(a.keystrokeEvents); i++ {
		flight := a.keystrokeEvents[i].Timestamp.Sub(a.keystrokeEvents[i-1].Timestamp)
		flightTimes = append(flightTimes, float64(flight.Milliseconds()))
	}

	if len(flightTimes) > 0 {
		var sumFlight float64
		for _, f := range flightTimes {
			sumFlight += f
		}
		k.AvgFlightTime = sumFlight / float64(len(flightTimes))

		// Flight variance
		var varianceFlight float64
		for _, f := range flightTimes {
			diff := f - k.AvgFlightTime
			varianceFlight += diff * diff
		}
		k.FlightVariance = varianceFlight / float64(len(flightTimes))

		// Count pauses (> 500ms between keystrokes)
		var pauseDurations []float64
		for _, f := range flightTimes {
			if f > 500 {
				k.PauseCount++
				pauseDurations = append(pauseDurations, f)
			}
		}

		if len(pauseDurations) > 0 {
			var sumPause float64
			for _, p := range pauseDurations {
				sumPause += p
			}
			k.AvgPauseDuration = sumPause / float64(len(pauseDurations))

			// Pause SD
			var variancePause float64
			for _, p := range pauseDurations {
				diff := p - k.AvgPauseDuration
				variancePause += diff * diff
			}
			k.PauseDurationSD = math.Sqrt(variancePause / float64(len(pauseDurations)))
		}
	}

	// Keys per minute
	if len(a.keystrokeEvents) > 0 {
		first := a.keystrokeEvents[0].Timestamp
		last := a.keystrokeEvents[len(a.keystrokeEvents)-1].Timestamp
		duration := last.Sub(first).Minutes()
		if duration > 0 {
			k.KeysPerMinute = float64(len(a.keystrokeEvents)) / duration
		}
	}

	// Error rate
	k.ErrorRate = float64(errors) / float64(len(a.keystrokeEvents))

	return k
}

// calculateContextMetrics analyzes window/task patterns.
func (a *Analyzer) calculateContextMetrics() ContextMetrics {
	c := ContextMetrics{}

	if len(a.windowSwitches) == 0 {
		return c
	}

	// Switches per minute
	if len(a.windowSwitches) > 0 {
		first := a.windowSwitches[0]
		last := a.windowSwitches[len(a.windowSwitches)-1]
		duration := last.Sub(first).Minutes()
		if duration > 0 {
			c.SwitchesPerMinute = float64(len(a.windowSwitches)) / duration
		}
	}

	// Rapid switches (< 5 seconds apart)
	for i := 1; i < len(a.windowSwitches); i++ {
		if a.windowSwitches[i].Sub(a.windowSwitches[i-1]) < 5*time.Second {
			c.RapidSwitches++
		}
	}

	// Average time per window
	if len(a.windowSwitches) > 1 {
		var totalTime time.Duration
		for i := 1; i < len(a.windowSwitches); i++ {
			totalTime += a.windowSwitches[i].Sub(a.windowSwitches[i-1])
		}
		c.AvgTimePerWindow = totalTime.Seconds() / float64(len(a.windowSwitches)-1)
	}

	// Idle periods
	c.IdlePeriodsCount = len(a.idlePeriods)
	if len(a.idlePeriods) > 0 {
		var totalIdle time.Duration
		for _, p := range a.idlePeriods {
			totalIdle += p.Duration
			if p.Duration.Seconds() > c.LongestIdlePeriod {
				c.LongestIdlePeriod = p.Duration.Seconds()
			}
		}
		c.AvgIdleDuration = totalIdle.Seconds() / float64(len(a.idlePeriods))
	}

	return c
}

// calculateStressScore computes an overall stress score (0-100).
func (a *Analyzer) calculateStressScore(s *StressSnapshot) (float64, []string) {
	var score float64
	var indicators []string

	// Mouse metrics contribution (0-30 points)
	mouseScore := 0.0

	// High jitter indicates stress
	if s.Mouse.Jitter > 0.3 {
		mouseScore += 15
		indicators = append(indicators, "high mouse jitter")
	} else if s.Mouse.Jitter > 0.15 {
		mouseScore += 8
	}

	// High speed variance indicates erratic movement
	if s.Mouse.SpeedVariance > 50000 {
		mouseScore += 10
		indicators = append(indicators, "erratic mouse speed")
	} else if s.Mouse.SpeedVariance > 20000 {
		mouseScore += 5
	}

	// Low directness (wandering cursor)
	if s.Mouse.DirectnessRatio > 3.0 {
		mouseScore += 5
		indicators = append(indicators, "indirect mouse paths")
	}

	score += mouseScore

	// Keystroke metrics contribution (0-40 points)
	// Research shows this is most predictive
	keystrokeScore := 0.0

	// Many pauses is the strongest indicator
	if s.Keystrokes.PauseCount > 10 {
		keystrokeScore += 15
		indicators = append(indicators, "many typing pauses")
	} else if s.Keystrokes.PauseCount > 5 {
		keystrokeScore += 8
	}

	// High pause duration SD indicates inconsistent thinking
	if s.Keystrokes.PauseDurationSD > 1000 {
		keystrokeScore += 10
		indicators = append(indicators, "inconsistent pause duration")
	}

	// High error rate
	if s.Keystrokes.ErrorRate > 0.15 {
		keystrokeScore += 10
		indicators = append(indicators, "high typing error rate")
	} else if s.Keystrokes.ErrorRate > 0.08 {
		keystrokeScore += 5
	}

	// Low typing speed (compared to baseline or absolute)
	if s.Keystrokes.KeysPerMinute > 0 && s.Keystrokes.KeysPerMinute < 30 {
		keystrokeScore += 5
		indicators = append(indicators, "slow typing speed")
	}

	score += keystrokeScore

	// Context metrics contribution (0-30 points)
	contextScore := 0.0

	// Rapid window switching indicates fragmented attention
	if s.Context.RapidSwitches > 10 {
		contextScore += 15
		indicators = append(indicators, "rapid window switching")
	} else if s.Context.RapidSwitches > 5 {
		contextScore += 8
	}

	// High switch rate
	if s.Context.SwitchesPerMinute > 3 {
		contextScore += 10
		indicators = append(indicators, "high context switching")
	} else if s.Context.SwitchesPerMinute > 1.5 {
		contextScore += 5
	}

	// Very short average time per window
	if s.Context.AvgTimePerWindow > 0 && s.Context.AvgTimePerWindow < 10 {
		contextScore += 5
		indicators = append(indicators, "brief focus periods")
	}

	score += contextScore

	// Cap at 100
	if score > 100 {
		score = 100
	}

	return score, indicators
}

// scoreToLevel converts a numeric score to a stress level.
func scoreToLevel(score float64) StressLevel {
	switch {
	case score < 15:
		return StressLevelCalm
	case score < 35:
		return StressLevelNormal
	case score < 55:
		return StressLevelElevated
	case score < 75:
		return StressLevelHigh
	default:
		return StressLevelAnxious
	}
}

// Capturer wraps Analyzer to implement the capture.Capturer interface.
type Capturer struct {
	analyzer                 *Analyzer
	platform                 *platform.Platform
	lastCursorX, lastCursorY int
}

// NewCapturer creates a new biometrics capturer.
func NewCapturer(plat *platform.Platform) *Capturer {
	return &Capturer{
		analyzer: NewAnalyzer(plat),
		platform: plat,
	}
}

// Name returns the capturer name.
func (c *Capturer) Name() string {
	return "biometrics"
}

// Available checks if biometrics capture is available.
func (c *Capturer) Available() bool {
	// Available on Hyprland (we can get cursor position)
	return c.platform.DisplayServer == platform.DisplayServerHyprland
}

// Capture captures current biometric data.
func (c *Capturer) Capture(ctx context.Context) (*capture.Result, error) {
	// Get current stress analysis
	snapshot := c.analyzer.Analyze()

	result := &capture.Result{
		Source:    "biometrics",
		Timestamp: time.Now(),
		Metadata: map[string]string{
			"stress_level":         snapshot.Level.String(),
			"stress_score":         fmt.Sprintf("%.1f", snapshot.OverallScore),
			"mouse_jitter":         fmt.Sprintf("%.3f", snapshot.Mouse.Jitter),
			"mouse_speed_variance": fmt.Sprintf("%.1f", snapshot.Mouse.SpeedVariance),
			"typing_pauses":        fmt.Sprintf("%d", snapshot.Keystrokes.PauseCount),
			"typing_error_rate":    fmt.Sprintf("%.3f", snapshot.Keystrokes.ErrorRate),
			"window_switches_pm":   fmt.Sprintf("%.2f", snapshot.Context.SwitchesPerMinute),
			"rapid_switches":       fmt.Sprintf("%d", snapshot.Context.RapidSwitches),
		},
	}

	if len(snapshot.TopIndicators) > 0 {
		result.TextData = "Indicators: " + join(snapshot.TopIndicators, ", ")
	}

	return result, nil
}

// GetAnalyzer returns the underlying analyzer for direct recording.
func (c *Capturer) GetAnalyzer() *Analyzer {
	return c.analyzer
}

func join(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
