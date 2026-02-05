// Package llm provides LLM integration via OpenRouter.
//
// OpenRouter is an API proxy that gives access to multiple LLM providers
// (OpenAI, Anthropic, etc.) through a single API.
//
// We use it for:
// - Chat completions (answering queries about past activity)
// - Embeddings (semantic search over captures)
package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// Client is an OpenRouter API client.
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client

	// Models
	ChatModel      string
	EmbeddingModel string

	// Debug mode
	Debug bool
}

// NewClient creates a new OpenRouter client.
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:         apiKey,
		baseURL:        "https://openrouter.ai/api/v1",
		httpClient:     &http.Client{Timeout: 180 * time.Second}, // 3 min for vision OCR
		ChatModel:      "openai/gpt-4o-mini",
		EmbeddingModel: "openai/text-embedding-3-small",
		Debug:          false,
	}
}

// SetDebug enables or disables debug logging.
func (c *Client) SetDebug(enabled bool) {
	c.Debug = enabled
}

// debugLog logs a message if debug mode is enabled.
func (c *Client) debugLog(format string, args ...interface{}) {
	if c.Debug {
		log.Printf("[llm] "+format, args...)
	}
}

// Message represents a chat message.
type Message struct {
	Role    string `json:"role"` // "system", "user", or "assistant"
	Content string `json:"content"`
}

// ChatRequest is the request body for chat completions.
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
}

// StreamChunk represents a single SSE chunk from streaming response.
type StreamChunk struct {
	ID      string `json:"id"`
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

// ChatResponse is the response from chat completions.
type ChatResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

// EmbeddingRequest is the request body for embeddings.
type EmbeddingRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

// EmbeddingResponse is the response from embeddings.
type EmbeddingResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

// Chat sends a chat completion request.
func (c *Client) Chat(ctx context.Context, messages []Message) (string, error) {
	req := ChatRequest{
		Model:       c.ChatModel,
		Messages:    messages,
		MaxTokens:   4096, // Large for comprehensive memory recall
		Temperature: 0.7,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("HTTP-Referer", "https://github.com/Atharva-Kanherkar/mnemosyne")
	httpReq.Header.Set("X-Title", "Mnemosyne")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w (body: %s)", err, string(respBody))
	}

	if chatResp.Error != nil {
		return "", fmt.Errorf("API error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	content := chatResp.Choices[0].Message.Content
	finishReason := chatResp.Choices[0].FinishReason

	// Warn if response was truncated
	if finishReason == "length" {
		content += "\n\n[Response truncated due to token limit]"
	}

	return content, nil
}

// ChatWithSystem sends a chat with a system prompt.
func (c *Client) ChatWithSystem(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userMessage},
	}
	return c.Chat(ctx, messages)
}

// ChatStream sends a streaming chat request, calling onChunk for each token.
// Returns the complete response text.
func (c *Client) ChatStream(ctx context.Context, messages []Message, onChunk func(string)) (string, error) {
	c.debugLog("ChatStream starting with model: %s", c.ChatModel)

	req := ChatRequest{
		Model:       c.ChatModel,
		Messages:    messages,
		MaxTokens:   4096,
		Temperature: 0.7,
		Stream:      true,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	c.debugLog("Request body size: %d bytes", len(body))

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("HTTP-Referer", "https://github.com/Atharva-Kanherkar/mnemosyne")
	httpReq.Header.Set("X-Title", "Mnemosyne")
	httpReq.Header.Set("Accept", "text/event-stream")

	c.debugLog("Sending request to OpenRouter...")
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		c.debugLog("Request failed: %v", err)
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	c.debugLog("Response status: %d", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.debugLog("API error body: %s", string(body))
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Read SSE stream
	c.debugLog("Starting to read SSE stream...")
	var fullResponse strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	lineCount := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineCount++

		if lineCount <= 3 {
			c.debugLog("SSE line %d: %s", lineCount, line)
		}

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		// Parse SSE data line
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")

			// Check for stream end
			if data == "[DONE]" {
				c.debugLog("Received [DONE] signal")
				break
			}

			// Parse chunk
			var chunk StreamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				c.debugLog("Failed to parse chunk: %v", err)
				continue // Skip malformed chunks
			}

			// Extract content delta
			if len(chunk.Choices) > 0 {
				content := chunk.Choices[0].Delta.Content
				if content != "" {
					fullResponse.WriteString(content)
					if onChunk != nil {
						onChunk(content)
					}
				}

				// Check for finish reason
				if chunk.Choices[0].FinishReason == "length" {
					c.debugLog("Response truncated due to length")
					if onChunk != nil {
						onChunk("\n\n[Response truncated]")
					}
					fullResponse.WriteString("\n\n[Response truncated]")
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		c.debugLog("Scanner error: %v", err)
		return fullResponse.String(), fmt.Errorf("stream error: %w", err)
	}

	c.debugLog("Stream complete. Lines read: %d, Response length: %d", lineCount, fullResponse.Len())
	return fullResponse.String(), nil
}

// ChatStreamWithSystem sends a streaming chat with a system prompt.
func (c *Client) ChatStreamWithSystem(ctx context.Context, systemPrompt, userMessage string, onChunk func(string)) (string, error) {
	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userMessage},
	}
	return c.ChatStream(ctx, messages, onChunk)
}

// Embed generates an embedding vector for the given text.
func (c *Client) Embed(ctx context.Context, text string) ([]float64, error) {
	req := EmbeddingRequest{
		Model: c.EmbeddingModel,
		Input: text,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		c.baseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("HTTP-Referer", "https://github.com/Atharva-Kanherkar/mnemosyne")
	httpReq.Header.Set("X-Title", "Mnemosyne")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var embResp EmbeddingResponse
	if err := json.Unmarshal(respBody, &embResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if embResp.Error != nil {
		return nil, fmt.Errorf("API error: %s", embResp.Error.Message)
	}

	if len(embResp.Data) == 0 {
		return nil, fmt.Errorf("no embedding data in response")
	}

	return embResp.Data[0].Embedding, nil
}

// SummarizeSystemPrompt is the system prompt for summaries.
const SummarizeSystemPrompt = `You are Mnemosyne, summarizing a user's computer activity.
Given multi-modal data (windows, screen OCR, clipboard, git, stress biometrics), provide:

1. WHAT they were doing (apps, tasks, content)
2. HOW they were doing (stress level, focus, fragmentation)
3. KEY MOMENTS (important things they copied, code changes, stress spikes)

Be specific: mention actual window titles, code snippets, stress indicators.
Keep it to 3-5 sentences but make them rich with detail.
This is their memory - help them remember everything important.

FORMATTING: Output plain text only. NO markdown (no #, **, -, *). Use natural paragraphs.`

// Summarize generates a summary of the given text.
func (c *Client) Summarize(ctx context.Context, text string) (string, error) {
	return c.ChatWithSystem(ctx, SummarizeSystemPrompt, text)
}

// SummarizeStream generates a summary with streaming output.
func (c *Client) SummarizeStream(ctx context.Context, text string, onChunk func(string)) (string, error) {
	return c.ChatStreamWithSystem(ctx, SummarizeSystemPrompt, text, onChunk)
}

// MnemosyneSystemPrompt is the system prompt for memory queries.
const MnemosyneSystemPrompt = `You are Mnemosyne, a comprehensive personal memory and cognitive assistant.
You have access to RICH, MULTI-MODAL context about the user's computer activity:

DATA SOURCES YOU HAVE:
1. WINDOW CONTEXT: What apps/websites were open, window titles, workspace names
2. SCREEN CONTENT: OCR/vision-extracted text from screenshots - actual code, documents, web pages they were viewing
3. CLIPBOARD: Everything they copied - code snippets, URLs, text
4. GIT ACTIVITY: Repos, branches, commits, uncommitted changes, what code they modified
5. ACTIVITY PATTERNS: Idle time, active/inactive states
6. BIOMETRICS/STRESS DATA: Real-time stress analysis based on:
   - Mouse jitter (erratic movement indicates anxiety)
   - Typing pauses (hesitation indicates cognitive load)
   - Typing error rate (backspace frequency)
   - Window switching frequency (fragmented attention)
   - Rapid context switches (anxiety indicator)

YOUR RESPONSE STYLE:
- Give COMPREHENSIVE answers that weave together ALL relevant context
- If they ask "what was I doing", tell them the apps, the content on screen, what they copied, their stress level
- If they ask about stress/anxiety, explain what triggered it based on the context
- Reference specific timestamps, window titles, code snippets
- Connect the dots: "You were stressed while working on X, copying error messages, rapidly switching windows"
- Be conversational but thorough - this is their external memory
- If you see patterns (e.g., stress correlating with certain tasks), point them out

FORMATTING RULES (VERY IMPORTANT):
- Output is displayed in a terminal, so DO NOT use markdown formatting
- NO headers with # or ###
- NO bold with ** or __
- NO bullet points with - or *
- Use plain text only, with natural paragraph breaks
- Use line breaks to separate sections
- Write in a flowing, conversational style

The user has memory difficulties and relies on you to reconstruct their experience completely.`

// AnswerQuery answers a question about past activity given context.
func (c *Client) AnswerQuery(ctx context.Context, query string, context string) (string, error) {
	userMessage := fmt.Sprintf("Complete context from all sensors:\n%s\n\nUser's question: %s", context, query)
	return c.ChatWithSystem(ctx, MnemosyneSystemPrompt, userMessage)
}

// AnswerQueryStream answers a question with streaming output.
func (c *Client) AnswerQueryStream(ctx context.Context, query string, context string, onChunk func(string)) (string, error) {
	userMessage := fmt.Sprintf("Complete context from all sensors:\n%s\n\nUser's question: %s", context, query)
	return c.ChatStreamWithSystem(ctx, MnemosyneSystemPrompt, userMessage, onChunk)
}

// SetModels allows changing the models used.
func (c *Client) SetModels(chatModel, embeddingModel string) {
	if chatModel != "" {
		c.ChatModel = chatModel
	}
	if embeddingModel != "" {
		c.EmbeddingModel = embeddingModel
	}
}
