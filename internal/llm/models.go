// Package llm - model catalog from OpenRouter API.
package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

// Model represents an OpenRouter model.
type Model struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Description   string  `json:"description"`
	ContextLength int     `json:"context_length"`
	Pricing       Pricing `json:"pricing"`
	Architecture  struct {
		Modality        string   `json:"modality"`
		InputModalities []string `json:"input_modalities"`
	} `json:"architecture"`
}

// Pricing holds model pricing info.
type Pricing struct {
	Prompt     string `json:"prompt"`
	Completion string `json:"completion"`
}

// ModelsResponse is the API response structure.
type ModelsResponse struct {
	Data []Model `json:"data"`
}

// FetchModels gets the current model list from OpenRouter.
func FetchModels(ctx context.Context) ([]Model, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://openrouter.ai/api/v1/models", nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var modelsResp ModelsResponse
	if err := json.Unmarshal(body, &modelsResp); err != nil {
		return nil, err
	}

	return modelsResp.Data, nil
}

// FilterModels filters models by criteria.
func FilterModels(models []Model, query string) []Model {
	query = strings.ToLower(query)
	var filtered []Model

	for _, m := range models {
		// Search in ID, name, and description
		if strings.Contains(strings.ToLower(m.ID), query) ||
			strings.Contains(strings.ToLower(m.Name), query) ||
			strings.Contains(strings.ToLower(m.Description), query) {
			filtered = append(filtered, m)
		}
	}

	return filtered
}

// GetFreeModels returns models with zero pricing.
func GetFreeModels(models []Model) []Model {
	var free []Model
	for _, m := range models {
		if m.Pricing.Prompt == "0" && m.Pricing.Completion == "0" {
			free = append(free, m)
		}
	}
	return free
}

// GetVisionModels returns models that support image input.
func GetVisionModels(models []Model) []Model {
	var vision []Model
	for _, m := range models {
		for _, mod := range m.Architecture.InputModalities {
			if mod == "image" {
				vision = append(vision, m)
				break
			}
		}
	}
	return vision
}

// GetByProvider returns models from a specific provider.
func GetByProvider(models []Model, provider string) []Model {
	provider = strings.ToLower(provider)
	var filtered []Model
	for _, m := range models {
		if strings.HasPrefix(strings.ToLower(m.ID), provider+"/") {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

// SortByContextLength sorts models by context length (descending).
func SortByContextLength(models []Model) {
	sort.Slice(models, func(i, j int) bool {
		return models[i].ContextLength > models[j].ContextLength
	})
}

// GetProvider extracts provider from model ID.
func GetProvider(modelID string) string {
	parts := strings.Split(modelID, "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// FormatModelInfo returns a formatted string for display.
func FormatModelInfo(m Model) string {
	provider := GetProvider(m.ID)
	ctx := formatContextLength(m.ContextLength)
	price := formatPrice(m.Pricing)

	return fmt.Sprintf("%s | %s | ctx: %s | %s", m.ID, provider, ctx, price)
}

func formatContextLength(ctx int) string {
	if ctx >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(ctx)/1000000)
	}
	if ctx >= 1000 {
		return fmt.Sprintf("%dk", ctx/1000)
	}
	return fmt.Sprintf("%d", ctx)
}

func formatPrice(p Pricing) string {
	if p.Prompt == "0" && p.Completion == "0" {
		return "FREE"
	}
	// Price is per token, convert to per 1M
	return fmt.Sprintf("$%s/$%s per 1M", p.Prompt, p.Completion)
}

// RecommendedModels returns a curated list of good models for Mnemosyne.
var RecommendedModels = []string{
	// Fast & cheap
	"openai/gpt-4o-mini",
	"anthropic/claude-3.5-haiku",
	"google/gemini-2.0-flash-001",
	"deepseek/deepseek-chat",
	"qwen/qwen-2.5-72b-instruct",

	// Best quality
	"anthropic/claude-3.5-sonnet",
	"openai/gpt-4o",
	"google/gemini-2.0-flash-thinking-exp",
	"deepseek/deepseek-r1",

	// Long context
	"google/gemini-pro-1.5",
	"anthropic/claude-3-opus",

	// Free
	"openrouter/free",
	"qwen/qwen3-coder-next",
	"stepfun/step-3.5-flash:free",

	// Chinese models
	"qwen/qwen-2.5-72b-instruct",
	"deepseek/deepseek-chat",
	"z-ai/glm-4.7-flash",
	"moonshotai/kimi-k2.5",
	"minimax/minimax-m2-her",
}
