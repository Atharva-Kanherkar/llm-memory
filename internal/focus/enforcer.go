package focus

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Atharva-Kanherkar/mnemosyne/internal/notify"
	"github.com/Atharva-Kanherkar/mnemosyne/internal/storage"
)

// Enforcer monitors windows and enforces focus mode rules.
type Enforcer struct {
	store      *storage.Store
	controller *Controller
	notifier   *notify.DesktopNotifier
	apiKey     string
	llmModel   string

	mode      *FocusMode
	sessionID int64
	active    bool

	// Cache LLM decisions for browser tabs
	tabCache map[string]Decision
	cacheMu  sync.RWMutex

	// Track warned windows to avoid spam
	warnedWindows map[string]time.Time
	warnedMu      sync.Mutex

	httpClient *http.Client
	mu         sync.RWMutex
}

// NewEnforcer creates a new focus mode enforcer.
func NewEnforcer(store *storage.Store, apiKey, llmModel string) *Enforcer {
	return &Enforcer{
		store:         store,
		controller:    NewController(),
		notifier:      notify.NewDesktopNotifier(),
		apiKey:        apiKey,
		llmModel:      llmModel,
		tabCache:      make(map[string]Decision),
		warnedWindows: make(map[string]time.Time),
		httpClient:    &http.Client{Timeout: 30 * time.Second},
	}
}

// Start activates a focus mode.
func (e *Enforcer) Start(mode *FocusMode) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Start a session in the database
	sessionID, err := e.store.StartFocusSession(mode.ID)
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}

	e.mode = mode
	e.sessionID = sessionID
	e.active = true

	// Clear caches
	e.cacheMu.Lock()
	e.tabCache = make(map[string]Decision)
	e.cacheMu.Unlock()

	e.warnedMu.Lock()
	e.warnedWindows = make(map[string]time.Time)
	e.warnedMu.Unlock()

	log.Printf("[focus] Started mode: %s", mode.Name)
	e.notifier.Send(
		"Focus Mode Active",
		fmt.Sprintf("%s - Stay focused!", mode.Name),
		notify.UrgencyNormal,
	)

	return nil
}

// Stop deactivates the current focus mode.
func (e *Enforcer) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.active {
		return nil
	}

	// End the session
	if err := e.store.EndFocusSession(e.sessionID); err != nil {
		log.Printf("[focus] Failed to end session: %v", err)
	}

	modeName := ""
	if e.mode != nil {
		modeName = e.mode.Name
	}

	// Reset all window borders to normal
	e.resetAllBorders()

	e.mode = nil
	e.sessionID = 0
	e.active = false

	log.Printf("[focus] Stopped focus mode")
	e.notifier.Send(
		"Focus Mode Ended",
		fmt.Sprintf("%s session complete", modeName),
		notify.UrgencyNormal,
	)

	return nil
}

// resetAllBorders resets all window borders to default.
func (e *Enforcer) resetAllBorders() {
	windows, err := e.controller.ListWindows()
	if err != nil {
		return
	}
	for _, w := range windows {
		e.controller.ResetWindowBorder(w.Address)
	}
}

// IsActive returns whether focus mode is active.
func (e *Enforcer) IsActive() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.active
}

// GetCurrentMode returns the current active mode if any.
func (e *Enforcer) GetCurrentMode() *FocusMode {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.mode
}

// Run starts the enforcement loop.
func (e *Enforcer) Run(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.check()
		}
	}
}

func (e *Enforcer) check() {
	e.mu.RLock()
	if !e.active || e.mode == nil {
		e.mu.RUnlock()
		return
	}
	mode := e.mode
	sessionID := e.sessionID
	e.mu.RUnlock()

	// Get active window
	window, err := e.controller.GetActiveWindow()
	if err != nil {
		return
	}

	// Evaluate the window
	decision := e.evaluate(window, mode)

	if decision.Action == "warn" {
		// Set red border immediately
		e.controller.SetWindowWarned(window.Address)
		e.handleWarn(window, decision, sessionID)
	} else if decision.Allowed {
		// Set green border for allowed windows
		e.controller.SetWindowAllowed(window.Address)
	}
}

func (e *Enforcer) evaluate(window *Window, mode *FocusMode) Decision {
	appClass := strings.ToLower(window.Class)

	// 1. Check if explicitly allowed
	for _, allowed := range mode.AllowedApps {
		if strings.ToLower(allowed) == appClass {
			return Decision{Allowed: true, Action: "allow"}
		}
	}

	// 2. Check if explicitly blocked - but ask LLM if the CONTENT is relevant
	for _, blocked := range mode.BlockedApps {
		if strings.ToLower(blocked) == appClass {
			// Don't auto-block! Check if this specific window content is relevant
			return e.askLLMForApp(window, blocked, mode)
		}
	}

	// 3. For browsers, check the tab
	if IsBrowser(appClass) {
		return e.evaluateBrowserTab(window, mode)
	}

	// 4. Check title for blocked patterns - ask LLM if content is relevant
	titleLower := strings.ToLower(window.Title)
	for _, pattern := range mode.BlockedPatterns {
		if strings.Contains(titleLower, strings.ToLower(pattern)) {
			return e.askLLMWithContext(window.Title, pattern, mode)
		}
	}

	// 5. Default: allow unknown apps (don't be too restrictive)
	return Decision{Allowed: true, Action: "allow"}
}

func (e *Enforcer) askLLMForApp(window *Window, blockedApp string, mode *FocusMode) Decision {
	// Check cache first
	cacheKey := window.Class + "|" + window.Title
	e.cacheMu.RLock()
	if cached, ok := e.tabCache[cacheKey]; ok {
		e.cacheMu.RUnlock()
		return cached
	}
	e.cacheMu.RUnlock()

	prompt := fmt.Sprintf(`Focus mode purpose: %s

App: %s
Window title: %s

The app "%s" is in the blocked list, but I need you to check if this SPECIFIC window/content is actually relevant and useful for the focus purpose.

Examples:
- If purpose is "learning AI" and app is Discord with title "AI Engineers Server - #learning" → ALLOW (relevant community)
- If purpose is "learning AI" and app is Discord with title "Gaming Server - #memes" → BLOCK (not relevant)
- If purpose is "coding project" and app is Slack with title "Project Team - #dev" → ALLOW (work related)
- If purpose is "coding project" and app is Slack with title "Random - #offtopic" → BLOCK (distraction)

Is this SPECIFIC content aligned with "%s"?
Reply with ONLY one word: ALLOW or BLOCK`, mode.Purpose, window.Class, window.Title, blockedApp, mode.Purpose)

	return e.callLLMForDecision(prompt, cacheKey)
}

func (e *Enforcer) evaluateBrowserTab(window *Window, mode *FocusMode) Decision {
	title := window.Title
	titleLower := strings.ToLower(title)

	// Check cache first
	e.cacheMu.RLock()
	if cached, ok := e.tabCache[title]; ok {
		e.cacheMu.RUnlock()
		return cached
	}
	e.cacheMu.RUnlock()

	// Check allowed sites in title - these are always allowed
	for _, site := range mode.AllowedSites {
		if strings.Contains(titleLower, strings.ToLower(site)) {
			decision := Decision{Allowed: true, Action: "allow"}
			e.cacheDecision(title, decision)
			return decision
		}
	}

	// Check if title contains blocked patterns - but ASK LLM if it's actually relevant
	for _, pattern := range mode.BlockedPatterns {
		if strings.Contains(titleLower, strings.ToLower(pattern)) {
			// Don't auto-block! Ask LLM if this specific content is relevant to the purpose
			return e.askLLMWithContext(title, pattern, mode)
		}
	}

	// For other tabs, ask LLM if browser policy allows
	if mode.BrowserPolicy == BrowserPolicyAskLLM {
		return e.askLLM(title, mode)
	}

	// Default allow for allowlist mode if not matched
	return Decision{Allowed: true, Action: "allow"}
}

func (e *Enforcer) askLLMWithContext(title, matchedPattern string, mode *FocusMode) Decision {
	prompt := fmt.Sprintf(`Focus mode purpose: %s

Window/Tab title: %s

This matched the blocked pattern "%s", but I need you to check if this SPECIFIC content is actually relevant and useful for the focus purpose.

Examples:
- If purpose is "learning AI" and title is "YouTube - Machine Learning Tutorial" → ALLOW (educational, relevant)
- If purpose is "learning AI" and title is "YouTube - Funny Cat Videos" → BLOCK (entertainment, not relevant)
- If purpose is "studying math" and title is "Reddit - r/learnmath discussion" → ALLOW (educational, relevant)
- If purpose is "studying math" and title is "Reddit - r/memes" → BLOCK (entertainment, not relevant)

Is this SPECIFIC content aligned with "%s"?
Reply with ONLY one word: ALLOW or BLOCK`, mode.Purpose, title, matchedPattern, mode.Purpose)

	return e.callLLMForDecision(prompt, title)
}

func (e *Enforcer) askLLM(tabTitle string, mode *FocusMode) Decision {
	prompt := fmt.Sprintf(`Focus mode purpose: %s

Browser tab: %s

Is this browser tab aligned with the focus purpose? Consider:
- If it's a tool needed for the stated purpose, it's allowed
- If it's a distraction (social media, entertainment, news unrelated to the task), it's blocked
- Educational content related to the purpose should be ALLOWED
- When in doubt, allow

Reply with ONLY one word: ALLOW or BLOCK`, mode.Purpose, tabTitle)

	return e.callLLMForDecision(prompt, tabTitle)
}

func (e *Enforcer) callLLMForDecision(prompt, cacheKey string) Decision {
	reqBody := map[string]any{
		"model": e.llmModel,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens":  10,
		"temperature": 0,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		// On error, default to allow
		return Decision{Allowed: true, Action: "allow"}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST",
		"https://openrouter.ai/api/v1/chat/completions",
		bytes.NewReader(body))
	if err != nil {
		return Decision{Allowed: true, Action: "allow"}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)
	req.Header.Set("HTTP-Referer", "https://github.com/Atharva-Kanherkar/mnemosyne")
	req.Header.Set("X-Title", "Mnemosyne Focus Mode")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return Decision{Allowed: true, Action: "allow"}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return Decision{Allowed: true, Action: "allow"}
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return Decision{Allowed: true, Action: "allow"}
	}

	if len(result.Choices) == 0 {
		return Decision{Allowed: true, Action: "allow"}
	}

	response := strings.ToUpper(strings.TrimSpace(result.Choices[0].Message.Content))

	var decision Decision
	if strings.Contains(response, "BLOCK") {
		decision = Decision{
			Allowed: false,
			Action:  "warn",
			Reason:  "Content not aligned with focus purpose",
		}
	} else {
		decision = Decision{Allowed: true, Action: "allow"}
	}

	e.cacheDecision(cacheKey, decision)
	log.Printf("[focus] LLM decision for '%s': %s", truncateTitle(cacheKey, 50), decision.Action)
	return decision
}

func (e *Enforcer) cacheDecision(title string, decision Decision) {
	e.cacheMu.Lock()
	e.tabCache[title] = decision
	e.cacheMu.Unlock()
}

func (e *Enforcer) handleWarn(window *Window, decision Decision, sessionID int64) {
	// Check if we recently warned about this window
	e.warnedMu.Lock()
	if lastWarn, ok := e.warnedWindows[window.Address]; ok {
		if time.Since(lastWarn) < 30*time.Second {
			e.warnedMu.Unlock()
			// Reset border if we're not going to warn
			e.controller.ResetWindowBorder(window.Address)
			return // Don't spam warnings
		}
	}
	e.warnedWindows[window.Address] = time.Now()
	e.warnedMu.Unlock()

	// Flash the border red
	e.controller.FlashWindowRed(window.Address)

	// Send warning notification
	e.notifier.Send(
		"Distraction Detected",
		fmt.Sprintf("%s will close in 5 seconds\n%s", truncateTitle(window.Title, 40), decision.Reason),
		notify.UrgencyNormal,
	)

	// Wait 5 seconds with pulsing red border
	for i := 0; i < 5; i++ {
		time.Sleep(1 * time.Second)
		// Pulse effect - alternate between red and darker red
		if i%2 == 0 {
			e.controller.SetWindowBorder(window.Address, "rgba(ff0000ff)")
		} else {
			e.controller.SetWindowBorder(window.Address, "rgba(aa0000ff)")
		}
	}

	// Check if still on same window
	current, err := e.controller.GetActiveWindow()
	if err != nil {
		e.controller.ResetWindowBorder(window.Address)
		return
	}

	if current.Address != window.Address {
		log.Printf("[focus] User switched away from %s, not closing", window.Class)
		// Reset border on the window they switched away from
		e.controller.ResetWindowBorder(window.Address)
		return
	}

	// Close the window or tab
	log.Printf("[focus] Closing distraction: %s", window.Class)

	// For browsers, close just the tab (Ctrl+W) instead of the whole window
	if IsBrowser(strings.ToLower(window.Class)) {
		if err := e.controller.CloseBrowserTab(); err != nil {
			log.Printf("[focus] Failed to close browser tab: %v", err)
			// Fallback to closing window if tab close fails
			e.controller.CloseWindow(window.Address)
		}
	} else {
		// For non-browsers, close the window
		if err := e.controller.CloseWindow(window.Address); err != nil {
			log.Printf("[focus] Failed to close window: %v", err)
			e.controller.ResetWindowBorder(window.Address)
			return
		}
	}

	// Increment block count
	e.store.IncrementFocusSessionBlocks(sessionID)

	e.notifier.Send(
		"Distraction Blocked",
		fmt.Sprintf("Closed %s - Stay focused!", window.Class),
		notify.UrgencyLow,
	)
}

func truncateTitle(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
