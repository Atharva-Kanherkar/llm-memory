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
	httpClient *http.Client
}

// NewVisionOCR creates a new vision-based OCR engine.
func NewVisionOCR(apiKey string) *VisionOCR {
	return &VisionOCR{
		apiKey:     apiKey,
		baseURL:    "https://openrouter.ai/api/v1",
		model:      "openai/gpt-4o-mini", // Vision-capable and cheap
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

// Available returns true if we have an API key.
func (v *VisionOCR) Available() bool {
	return v.apiKey != ""
}

// ExtractText extracts text from image bytes using vision model.
func (v *VisionOCR) ExtractText(ctx context.Context, imageData []byte) (string, error) {
	if !v.Available() {
		return "", fmt.Errorf("no API key configured")
	}

	// Encode image as base64
	b64Image := base64.StdEncoding.EncodeToString(imageData)

	// Build the request with image
	req := map[string]interface{}{
		"model": v.model,
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": `Extract all visible text from this screenshot. Focus on:
1. Window titles and application names
2. Code or terminal content
3. Any readable text in the UI
4. Browser tabs or URLs if visible

Format the output clearly. If it's code, preserve the structure.
If you can identify what the user is working on, mention it briefly.
Keep response concise but complete.`,
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
		"max_tokens": 1000,
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
