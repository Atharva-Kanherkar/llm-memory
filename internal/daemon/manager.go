// Package daemon provides the Manager that orchestrates all capture sources.
//
// The Manager runs multiple capturers at different intervals:
// - Window: every 5 seconds (lightweight, important for context)
// - Screen: every 60 seconds (heavy, but visual context is valuable)
// - Git: every 30 seconds (lightweight, important for coding context)
// - Clipboard: every 5 seconds (only captures on change)
// - Activity: every 5 seconds (idle detection)
// - Audio: every 5 minutes (opt-in only, for ambient audio anchors)
package daemon

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/Atharva-Kanherkar/mnemosyne/internal/capture/activity"
	"github.com/Atharva-Kanherkar/mnemosyne/internal/capture/audio"
	"github.com/Atharva-Kanherkar/mnemosyne/internal/capture/biometrics"
	"github.com/Atharva-Kanherkar/mnemosyne/internal/capture/clipboard"
	"github.com/Atharva-Kanherkar/mnemosyne/internal/capture/git"
	"github.com/Atharva-Kanherkar/mnemosyne/internal/capture/screen"
	"github.com/Atharva-Kanherkar/mnemosyne/internal/capture/window"
	"github.com/Atharva-Kanherkar/mnemosyne/internal/config"
	"github.com/Atharva-Kanherkar/mnemosyne/internal/integrations"
	"github.com/Atharva-Kanherkar/mnemosyne/internal/ocr"
	"github.com/Atharva-Kanherkar/mnemosyne/internal/platform"
	"github.com/Atharva-Kanherkar/mnemosyne/internal/storage"
)

// Manager orchestrates all capture sources.
type Manager struct {
	cfg      *config.Config
	platform *platform.Platform
	store    *storage.Store

	// Capturers
	windowCapturer     *window.Capturer
	screenCapturer     *screen.Capturer
	gitCapturer        *git.Capturer
	clipboardCapturer  *clipboard.Capturer
	activityCapturer   *activity.Capturer
	audioCapturer      *audio.Capturer
	biometricsCapturer *biometrics.Capturer

	// OCR for pre-computing screen text
	ocrEngine *ocr.VisionOCR

	// External integrations (Gmail, Slack, Calendar)
	integrations *integrations.Manager

	// Biometrics trackers (high-frequency)
	mouseTracker    *biometrics.MouseTracker
	keyboardTracker *biometrics.KeyboardTracker

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// State tracking
	lastWindow string // To detect window changes
}

// CaptureIntervals defines how often each source is captured.
type CaptureIntervals struct {
	Window       time.Duration
	Screen       time.Duration
	Git          time.Duration
	Clipboard    time.Duration
	Activity     time.Duration
	Audio        time.Duration
	Biometrics   time.Duration
	Integrations time.Duration
}

// DefaultIntervals returns sensible default intervals.
func DefaultIntervals() CaptureIntervals {
	return CaptureIntervals{
		Window:       5 * time.Second,
		Screen:       60 * time.Second,
		Git:          30 * time.Second,
		Clipboard:    5 * time.Second,
		Activity:     5 * time.Second,
		Audio:        5 * time.Minute,  // Every 5 minutes, if enabled
		Biometrics:   30 * time.Second, // Stress snapshot every 30 seconds
		Integrations: 5 * time.Minute,  // Gmail, Slack, Calendar every 5 minutes
	}
}

// NewManager creates a new capture Manager.
func NewManager(cfg *config.Config, plat *platform.Platform, store *storage.Store, apiKey string) *Manager {
	// Create biometrics capturer and get the analyzer for trackers
	bioCapturer := biometrics.NewCapturer(plat)
	analyzer := bioCapturer.GetAnalyzer()

	// Create OCR engine for pre-computing screen text
	var ocrEngine *ocr.VisionOCR
	if apiKey != "" {
		ocrEngine = ocr.NewVisionOCR(apiKey)
		log.Println("[ocr] Vision OCR enabled for pre-computed screen text")
	} else {
		log.Println("[ocr] No API key - OCR disabled (queries will be slower)")
	}

	// Create integrations manager
	var intMgr *integrations.Manager
	intMgr, err := integrations.NewManager(cfg.StoragePath)
	if err != nil {
		log.Printf("[integrations] Failed to initialize: %v", err)
	} else {
		status := intMgr.GetProviderStatus()
		for provider, s := range status {
			if s["authenticated"] {
				log.Printf("[integrations] %s: connected", provider)
			}
		}
	}

	return &Manager{
		cfg:                cfg,
		platform:           plat,
		store:              store,
		windowCapturer:     window.New(plat),
		screenCapturer:     screen.New(plat),
		gitCapturer:        git.New(),
		clipboardCapturer:  clipboard.New(plat),
		activityCapturer:   activity.New(plat),
		audioCapturer:      audio.New(plat),
		biometricsCapturer: bioCapturer,
		ocrEngine:          ocrEngine,
		integrations:       intMgr,
		mouseTracker:       biometrics.NewMouseTracker(plat, analyzer),
		keyboardTracker:    biometrics.NewKeyboardTracker(plat, analyzer),
	}
}

// Start begins all capture loops.
func (m *Manager) Start(ctx context.Context) {
	m.ctx, m.cancel = context.WithCancel(ctx)
	intervals := DefaultIntervals()

	// Log what we're starting
	log.Println("Starting capture manager...")
	m.logAvailableCapturers()

	// Start each capturer in its own goroutine
	if m.cfg.WindowCaptureEnabled && m.windowCapturer.Available() {
		m.wg.Add(1)
		go m.runCaptureLoop("window", intervals.Window, m.captureWindow)
	}

	if m.cfg.ScreenCaptureEnabled && m.screenCapturer.Available() {
		m.wg.Add(1)
		go m.runCaptureLoop("screen", intervals.Screen, m.captureScreen)
	}

	if m.cfg.GitCaptureEnabled && m.gitCapturer.Available() {
		m.wg.Add(1)
		go m.runCaptureLoop("git", intervals.Git, m.captureGit)
	}

	if m.cfg.ClipboardCaptureEnabled && m.clipboardCapturer.Available() {
		m.wg.Add(1)
		go m.runCaptureLoop("clipboard", intervals.Clipboard, m.captureClipboard)
	}

	if m.activityCapturer.Available() {
		m.wg.Add(1)
		go m.runCaptureLoop("activity", intervals.Activity, m.captureActivity)
	}

	// Audio is opt-in
	if m.audioCapturer.Available() {
		m.wg.Add(1)
		go m.runCaptureLoop("audio", intervals.Audio, m.captureAudio)
	}

	// Biometrics (stress tracking)
	if m.biometricsCapturer.Available() {
		// Start high-frequency trackers
		m.mouseTracker.Start(m.ctx)
		if m.keyboardTracker.Available() {
			m.keyboardTracker.Start(m.ctx)
			log.Println("[biometrics] Keyboard tracking enabled")
		} else {
			log.Println("[biometrics] Keyboard tracking unavailable (add user to 'input' group)")
		}

		// Start periodic stress snapshots
		m.wg.Add(1)
		go m.runCaptureLoop("biometrics", intervals.Biometrics, m.captureBiometrics)
	}

	// External integrations (Gmail, Slack, Calendar)
	if m.integrations != nil {
		m.wg.Add(1)
		go m.runCaptureLoop("integrations", intervals.Integrations, m.captureIntegrations)
	}
}

// Stop gracefully stops all capture loops.
func (m *Manager) Stop() {
	log.Println("Stopping capture manager...")

	// Stop biometrics trackers first
	if m.mouseTracker != nil {
		m.mouseTracker.Stop()
	}
	if m.keyboardTracker != nil {
		m.keyboardTracker.Stop()
	}

	// Close integrations manager
	if m.integrations != nil {
		m.integrations.Close()
	}

	m.cancel()
	m.wg.Wait()
	log.Println("Capture manager stopped")
}

// runCaptureLoop runs a capture function at regular intervals.
func (m *Manager) runCaptureLoop(name string, interval time.Duration, captureFn func() error) {
	defer m.wg.Done()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("[%s] Starting capture loop (interval: %s)", name, interval)

	// Run once immediately
	if err := captureFn(); err != nil {
		log.Printf("[%s] Capture error: %v", name, err)
	}

	for {
		select {
		case <-m.ctx.Done():
			log.Printf("[%s] Stopping capture loop", name)
			return
		case <-ticker.C:
			if err := captureFn(); err != nil {
				log.Printf("[%s] Capture error: %v", name, err)
			}
		}
	}
}

// captureWindow captures window information.
func (m *Manager) captureWindow() error {
	result, err := m.windowCapturer.Capture(m.ctx)
	if err != nil {
		return err
	}

	// Check if window changed
	currentWindow := result.Metadata["app_class"] + "|" + result.Metadata["window_title"]
	if currentWindow == m.lastWindow {
		return nil // No change, don't save
	}
	m.lastWindow = currentWindow

	// Record window switch for biometrics/stress analysis
	if m.biometricsCapturer != nil {
		m.biometricsCapturer.GetAnalyzer().RecordWindowSwitch()
	}

	// Log the change
	title := result.Metadata["window_title"]
	if len(title) > 50 {
		title = title[:47] + "..."
	}
	log.Printf("[window] %s: %s", result.Metadata["app_class"], title)

	// Save to storage
	if m.store != nil {
		_, err = m.store.Save(result)
	}
	return err
}

// captureScreen captures a screenshot and pre-computes OCR text.
func (m *Manager) captureScreen() error {
	result, err := m.screenCapturer.Capture(m.ctx)
	if err != nil {
		return err
	}

	log.Printf("[screen] Captured %s bytes", result.Metadata["size_bytes"])

	// Pre-compute OCR if available
	if m.ocrEngine != nil && m.ocrEngine.Available() && len(result.RawData) > 0 {
		// Use a separate context with timeout for OCR
		ocrCtx, cancel := context.WithTimeout(m.ctx, 30*time.Second)
		defer cancel()

		ocrText, ocrErr := m.ocrEngine.ExtractText(ocrCtx, result.RawData)
		if ocrErr != nil {
			log.Printf("[ocr] Error extracting text: %v", ocrErr)
		} else if ocrText != "" {
			result.TextData = ocrText
			result.SetMetadata("ocr_precomputed", "true")
			// Log a brief preview
			preview := ocrText
			if len(preview) > 100 {
				preview = preview[:97] + "..."
			}
			log.Printf("[ocr] Extracted: %s", preview)
		}
	}

	if m.store != nil {
		_, err = m.store.Save(result)
	}
	return err
}

// captureGit captures git state.
func (m *Manager) captureGit() error {
	result, err := m.gitCapturer.Capture(m.ctx)
	if err != nil {
		return err
	}

	// Only log if in a repo
	if result.Metadata["in_repo"] == "true" {
		log.Printf("[git] %s @ %s (%s)",
			result.Metadata["repo_name"],
			result.Metadata["branch"],
			result.Metadata["commit"])

		if m.store != nil {
			_, err = m.store.Save(result)
		}
	}
	return err
}

// captureClipboard captures clipboard contents.
func (m *Manager) captureClipboard() error {
	result, err := m.clipboardCapturer.Capture(m.ctx)
	if err != nil {
		return err
	}

	// Only save if content changed
	if result.Metadata["changed"] != "true" {
		return nil
	}

	contentType := result.Metadata["content_type"]
	length := result.Metadata["length"]
	log.Printf("[clipboard] %s (%s chars)", contentType, length)

	if m.store != nil {
		_, err = m.store.Save(result)
	}
	return err
}

// captureActivity captures user activity state.
func (m *Manager) captureActivity() error {
	result, err := m.activityCapturer.Capture(m.ctx)
	if err != nil {
		return err
	}

	// Only log state changes or periodically
	// For now, just track without logging every time
	if m.store != nil {
		_, err = m.store.Save(result)
	}
	return err
}

// captureAudio captures an audio snippet.
func (m *Manager) captureAudio() error {
	result, err := m.audioCapturer.Capture(m.ctx)
	if err != nil {
		return err
	}

	log.Printf("[audio] Captured %s bytes (%sms)",
		result.Metadata["size_bytes"],
		result.Metadata["duration_ms"])

	if m.store != nil {
		_, err = m.store.Save(result)
	}
	return err
}

// captureBiometrics captures stress/anxiety metrics.
func (m *Manager) captureBiometrics() error {
	result, err := m.biometricsCapturer.Capture(m.ctx)
	if err != nil {
		return err
	}

	// Log stress level changes or significant stress
	level := result.Metadata["stress_level"]
	score := result.Metadata["stress_score"]

	if level == "elevated" || level == "high" || level == "anxious" {
		log.Printf("[biometrics] Stress: %s (score: %s)", level, score)
		if result.TextData != "" {
			log.Printf("[biometrics] %s", result.TextData)
		}
	}

	if m.store != nil {
		_, err = m.store.Save(result)
	}
	return err
}

// captureIntegrations captures data from external services (Gmail, Slack, Calendar).
func (m *Manager) captureIntegrations() error {
	if m.integrations == nil {
		return nil
	}

	// Capture Gmail if authenticated
	if gmailResult, err := m.integrations.CaptureGmail(m.ctx); err == nil && gmailResult != nil {
		if m.store != nil {
			m.store.Save(gmailResult)
		}
		unread := gmailResult.Metadata["unread_count"]
		log.Printf("[gmail] Captured emails (unread: %s)", unread)
	}

	// Capture Slack if authenticated
	if slackResult, err := m.integrations.CaptureSlack(m.ctx); err == nil && slackResult != nil {
		if m.store != nil {
			m.store.Save(slackResult)
		}
		count := slackResult.Metadata["message_count"]
		log.Printf("[slack] Captured messages: %s", count)
	}

	// Capture Calendar if authenticated
	if calResult, err := m.integrations.CaptureCalendar(m.ctx); err == nil && calResult != nil {
		if m.store != nil {
			m.store.Save(calResult)
		}
		count := calResult.Metadata["event_count"]
		next := calResult.Metadata["next_event"]
		if next != "" {
			log.Printf("[calendar] Captured %s events, next: %s", count, next)
		} else {
			log.Printf("[calendar] Captured %s events", count)
		}
	}

	return nil
}

// logAvailableCapturers logs which capturers are available.
func (m *Manager) logAvailableCapturers() {
	available := []string{}
	unavailable := []string{}

	check := func(name string, enabled, avail bool) {
		if !enabled {
			unavailable = append(unavailable, name+" (disabled)")
		} else if avail {
			available = append(available, name)
		} else {
			unavailable = append(unavailable, name+" (unavailable)")
		}
	}

	check("window", m.cfg.WindowCaptureEnabled, m.windowCapturer.Available())
	check("screen", m.cfg.ScreenCaptureEnabled, m.screenCapturer.Available())
	check("git", m.cfg.GitCaptureEnabled, m.gitCapturer.Available())
	check("clipboard", m.cfg.ClipboardCaptureEnabled, m.clipboardCapturer.Available())
	check("activity", true, m.activityCapturer.Available())
	check("audio", m.audioCapturer.Enabled, m.audioCapturer.Available())
	check("biometrics", true, m.biometricsCapturer.Available())

	log.Printf("Available capturers: %v", available)
	if len(unavailable) > 0 {
		log.Printf("Unavailable capturers: %v", unavailable)
	}
}

// EnableAudio enables audio capture (opt-in).
func (m *Manager) EnableAudio() {
	m.audioCapturer.Enable()
}

// DisableAudio disables audio capture.
func (m *Manager) DisableAudio() {
	m.audioCapturer.Disable()
}
