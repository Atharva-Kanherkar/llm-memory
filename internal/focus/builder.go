package focus

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Atharva-Kanherkar/mnemosyne/internal/storage"
)

// Builder creates focus modes through LLM conversation.
type Builder struct {
	store      *storage.Store
	apiKey     string
	llmModel   string
	httpClient *http.Client

	// Conversation state
	messages []Message
	phase    int // 0=initial, 1=gathering, 2=complete
	result   *FocusMode
}

// Message represents a chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// NewBuilder creates a new focus mode builder.
func NewBuilder(store *storage.Store, apiKey, llmModel string) *Builder {
	return &Builder{
		store:      store,
		apiKey:     apiKey,
		llmModel:   llmModel,
		httpClient: &http.Client{Timeout: 60 * time.Second},
		messages:   make([]Message, 0),
		phase:      0,
	}
}

const systemPrompt = `You are helping the user create a focus mode for their computer. Your job is to have a brief conversation to understand:

1. What they will be doing (the purpose/task)
2. Which apps they need to use
3. What distractions should be blocked
4. How long the session should be (optional)

Keep your responses conversational and SHORT (1-2 sentences + a question). Don't be overly formal.

When you have gathered enough information, output a JSON block with the focus mode configuration. The JSON should be wrapped in triple backticks with json marker like this:

` + "```json" + `
{
  "name": "Study Mode",
  "purpose": "Studying algorithms using textbooks and Claude",
  "allowed_apps": ["code", "firefox", "zathura", "kitty"],
  "blocked_apps": ["discord", "slack"],
  "blocked_patterns": ["youtube", "reddit", "twitter", "netflix", "twitch"],
  "allowed_sites": ["claude.ai", "github.com", "stackoverflow.com"],
  "duration_minutes": 120
}
` + "```" + `

Guidelines for the JSON:
- name: Short descriptive name (e.g., "Deep Work", "Study Mode", "Writing Session")
- purpose: Brief description of what they're doing
- allowed_apps: Apps they explicitly said they need (use lowercase, no spaces)
- blocked_apps: Apps that are obvious distractions for their task
- blocked_patterns: Keywords/sites that would be distracting (will match browser titles)
- allowed_sites: Websites they need for their work
- duration_minutes: How long in minutes (0 = no limit)

Common app names: code, firefox, chromium, kitty, alacritty, zathura, obsidian, notion, slack, discord, telegram, signal

Start by asking what they want to focus on.`

// Start begins a new conversation to build a focus mode.
func (b *Builder) Start() string {
	b.messages = []Message{
		{Role: "system", Content: systemPrompt},
	}
	b.phase = 1

	// Get initial response from LLM
	response, _ := b.chat("")
	return response
}

// Chat sends a user message and gets a response.
func (b *Builder) Chat(userInput string) (string, *FocusMode, error) {
	response, err := b.chat(userInput)
	if err != nil {
		return "", nil, err
	}

	// Check if response contains JSON (mode is complete)
	if mode := b.extractMode(response); mode != nil {
		b.result = mode
		b.phase = 2

		// Save to database
		record := &storage.FocusModeRecord{
			ID:              mode.ID,
			Name:            mode.Name,
			Purpose:         mode.Purpose,
			AllowedApps:     mode.MarshalAllowedApps(),
			BlockedApps:     mode.MarshalBlockedApps(),
			BlockedPatterns: mode.MarshalBlockedPatterns(),
			AllowedSites:    mode.MarshalAllowedSites(),
			BrowserPolicy:   mode.BrowserPolicy,
			DurationMinutes: mode.DurationMinutes,
			CreatedAt:       mode.CreatedAt,
		}

		if err := b.store.SaveFocusMode(record); err != nil {
			return "", nil, fmt.Errorf("failed to save mode: %w", err)
		}

		// Clean response - remove JSON block for display
		cleanResponse := b.cleanResponse(response)
		return cleanResponse, mode, nil
	}

	return response, nil, nil
}

func (b *Builder) chat(userInput string) (string, error) {
	if userInput != "" {
		b.messages = append(b.messages, Message{Role: "user", Content: userInput})
	}

	reqBody := map[string]any{
		"model":       b.llmModel,
		"messages":    b.messages,
		"max_tokens":  500,
		"temperature": 0.7,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST",
		"https://openrouter.ai/api/v1/chat/completions",
		bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+b.apiKey)
	req.Header.Set("HTTP-Referer", "https://github.com/Atharva-Kanherkar/mnemosyne")
	req.Header.Set("X-Title", "Mnemosyne Focus Builder")

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", err
	}

	if result.Error != nil {
		return "", fmt.Errorf("API error: %s", result.Error.Message)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no response from LLM")
	}

	response := result.Choices[0].Message.Content
	b.messages = append(b.messages, Message{Role: "assistant", Content: response})

	return response, nil
}

func (b *Builder) extractMode(response string) *FocusMode {
	// Look for JSON block
	startMarker := "```json"
	endMarker := "```"

	startIdx := strings.Index(response, startMarker)
	if startIdx == -1 {
		return nil
	}

	jsonStart := startIdx + len(startMarker)
	remaining := response[jsonStart:]
	endIdx := strings.Index(remaining, endMarker)
	if endIdx == -1 {
		return nil
	}

	jsonStr := strings.TrimSpace(remaining[:endIdx])

	var raw struct {
		Name            string   `json:"name"`
		Purpose         string   `json:"purpose"`
		AllowedApps     []string `json:"allowed_apps"`
		BlockedApps     []string `json:"blocked_apps"`
		BlockedPatterns []string `json:"blocked_patterns"`
		AllowedSites    []string `json:"allowed_sites"`
		DurationMinutes int      `json:"duration_minutes"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
		return nil
	}

	// Generate ID
	idBytes := make([]byte, 8)
	rand.Read(idBytes)
	id := hex.EncodeToString(idBytes)

	return &FocusMode{
		ID:              id,
		Name:            raw.Name,
		Purpose:         raw.Purpose,
		AllowedApps:     raw.AllowedApps,
		BlockedApps:     raw.BlockedApps,
		BlockedPatterns: raw.BlockedPatterns,
		BrowserPolicy:   BrowserPolicyAskLLM,
		AllowedSites:    raw.AllowedSites,
		DurationMinutes: raw.DurationMinutes,
		CreatedAt:       time.Now(),
	}
}

func (b *Builder) cleanResponse(response string) string {
	// Remove JSON block from response for cleaner display
	startMarker := "```json"
	startIdx := strings.Index(response, startMarker)
	if startIdx == -1 {
		return response
	}

	// Get text before JSON
	before := strings.TrimSpace(response[:startIdx])

	// Get text after JSON
	endMarker := "```"
	jsonStart := startIdx + len(startMarker)
	remaining := response[jsonStart:]
	endIdx := strings.Index(remaining, endMarker)
	if endIdx == -1 {
		return before
	}

	after := strings.TrimSpace(remaining[endIdx+len(endMarker):])

	if after != "" {
		return before + "\n\n" + after
	}
	return before
}

// IsComplete returns whether the conversation is complete.
func (b *Builder) IsComplete() bool {
	return b.phase == 2
}

// GetResult returns the built focus mode if complete.
func (b *Builder) GetResult() *FocusMode {
	return b.result
}

// Reset clears the conversation state.
func (b *Builder) Reset() {
	b.messages = nil
	b.phase = 0
	b.result = nil
}
