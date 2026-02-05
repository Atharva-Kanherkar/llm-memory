// Package ocr provides text extraction from images.
//
// This file implements LLM-based OCR using vision models via OpenRouter.
// This is often better than traditional OCR because:
// 1. It understands context (code vs prose vs UI)
// 2. It can describe what's on screen, not just extract text
// 3. No local dependencies needed
package ocr

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// VisionOCR uses a vision-capable LLM for text extraction.
type VisionOCR struct {
	apiKey     string
	baseURL    string
	model      string
	compressor string // Cheap model for text compression
	httpClient *http.Client
}

// NewVisionOCR creates a new vision-based OCR engine.
func NewVisionOCR(apiKey string) *VisionOCR {
	return &VisionOCR{
		apiKey:     apiKey,
		baseURL:    "https://openrouter.ai/api/v1",
		model:      "openai/gpt-4o-mini",      // Vision-capable for OCR
		compressor: "deepseek/deepseek-chat",  // Very cheap for compression ($0.07/M tokens)
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

// Available returns true if we have an API key.
func (v *VisionOCR) Available() bool {
	return v.apiKey != ""
}

// ExtractText extracts text from image and compresses it using a two-stage pipeline:
// Stage 1: Vision model extracts raw text/description from screenshot
// Stage 2: Cheap text model compresses it to minimal tokens while preserving meaning
func (v *VisionOCR) ExtractText(ctx context.Context, imageData []byte) (string, error) {
	if !v.Available() {
		return "", fmt.Errorf("no API key configured")
	}

	// Stage 1: Extract raw text/description from image
	rawText, err := v.extractRaw(ctx, imageData)
	if err != nil {
		return "", err
	}

	// Stage 2: Compress using cheap model
	compressed, err := v.compressText(ctx, rawText)
	if err != nil {
		// If compression fails, return truncated raw text
		if len(rawText) > 200 {
			return rawText[:200] + "...", nil
		}
		return rawText, nil
	}

	return compressed, nil
}

// extractRaw extracts raw text/description from screenshot using vision model.
func (v *VisionOCR) extractRaw(ctx context.Context, imageData []byte) (string, error) {
	b64Image := base64.StdEncoding.EncodeToString(imageData)

	req := map[string]interface{}{
		"model": v.model,
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": `Extract key information from this screenshot:
1. Application name and window title
2. What the user is doing (coding, browsing, chatting, etc.)
3. Important visible text (file names, URLs, code snippets, messages)
4. Any errors or notifications visible

Be thorough but focus on what matters for remembering this moment later.`,
					},
					{
						"type": "image_url",
						"image_url": map[string]string{
							"url": "data:image/png;base64," + b64Image,
						},
					},
				},
			},
		},
		"max_tokens": 500,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		v.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+v.apiKey)
	httpReq.Header.Set("HTTP-Referer", "https://github.com/Atharva-Kanherkar/mnemosyne")
	httpReq.Header.Set("X-Title", "Mnemosyne OCR")

	resp, err := v.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
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
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Error != nil {
		return "", fmt.Errorf("API error: %s", result.Error.Message)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no response from model")
	}

	return result.Choices[0].Message.Content, nil
}

// compressText uses a cheap model to compress raw OCR text into minimal tokens.
// This is Stage 2 of the pipeline - takes ~500 tokens input → ~50 tokens output.
func (v *VisionOCR) compressText(ctx context.Context, rawText string) (string, error) {
	if rawText == "" {
		return "", nil
	}

	req := map[string]interface{}{
		"model": v.compressor,
		"messages": []map[string]interface{}{
			{
				"role": "system",
				"content": `You are a compression engine. Compress the input to 1-2 sentences MAX.
Keep ONLY: app name, user action, key file/URL names, errors.
Drop: UI descriptions, formatting, redundant info.
Output raw text, no quotes or prefixes.`,
			},
			{
				"role": "user",
				"content": rawText,
			},
		},
		"max_tokens": 100, // Force very short output
		"temperature": 0,  // Deterministic
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		v.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+v.apiKey)
	httpReq.Header.Set("HTTP-Referer", "https://github.com/Atharva-Kanherkar/mnemosyne")
	httpReq.Header.Set("X-Title", "Mnemosyne Compress")

	resp, err := v.httpClient.Do(httpReq)
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
		return "", fmt.Errorf("compression error: %s", result.Error.Message)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no compression output")
	}

	return result.Choices[0].Message.Content, nil
}

// ExtractTextFromFile extracts text from an image file.
func (v *VisionOCR) ExtractTextFromFile(ctx context.Context, imagePath string) (string, error) {
	data, err := os.ReadFile(imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to read image: %w", err)
	}
	return v.ExtractText(ctx, data)
}

// DescribeScreen provides a higher-level description of what's on screen.
func (v *VisionOCR) DescribeScreen(ctx context.Context, imageData []byte) (string, error) {
	if !v.Available() {
		return "", fmt.Errorf("no API key configured")
	}

	b64Image := base64.StdEncoding.EncodeToString(imageData)

	req := map[string]interface{}{
		"model": v.model,
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": `Briefly describe what the user is doing in this screenshot.
What application are they using? What task are they working on?
Keep it to 1-2 sentences, like a memory log entry.`,
					},
					{
						"type": "image_url",
						"image_url": map[string]string{
							"url": "data:image/png;base64," + b64Image,
						},
					},
				},
			},
		},
		"max_tokens": 150,
	}

	body, _ := json.Marshal(req)

	httpReq, _ := http.NewRequestWithContext(ctx, "POST",
		v.baseURL+"/chat/completions", bytes.NewReader(body))

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+v.apiKey)
	httpReq.Header.Set("HTTP-Referer", "https://github.com/Atharva-Kanherkar/mnemosyne")

	resp, err := v.httpClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	json.Unmarshal(respBody, &result)

	if len(result.Choices) > 0 {
		return result.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("no response")
}
