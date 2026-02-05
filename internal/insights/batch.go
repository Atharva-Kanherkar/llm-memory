package insights

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Atharva-Kanherkar/mnemosyne/internal/storage"
)

// BatchAnalyzer performs periodic LLM analysis of captured data.
type BatchAnalyzer struct {
	store      *storage.Store
	apiKey     string
	model      string
	interval   time.Duration
	lastRun    time.Time
	httpClient *http.Client
}

// NewBatchAnalyzer creates a new batch analyzer.
func NewBatchAnalyzer(store *storage.Store, apiKey, model string, interval time.Duration) *BatchAnalyzer {
	return &BatchAnalyzer{
		store:      store,
		apiKey:     apiKey,
		model:      model,
		interval:   interval,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

// Run starts the batch analysis loop.
func (b *BatchAnalyzer) Run(ctx context.Context, onInsight func(*Insight)) {
	// Initial delay to let some data accumulate
	select {
	case <-ctx.Done():
		return
	case <-time.After(5 * time.Minute):
	}

	ticker := time.NewTicker(b.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			insights, err := b.analyze(ctx)
			if err != nil {
				log.Printf("[batch] Analysis error: %v", err)
				continue
			}
			for _, insight := range insights {
				onInsight(&insight)
			}
		}
	}
}

func (b *BatchAnalyzer) analyze(ctx context.Context) ([]Insight, error) {
	end := time.Now()
	start := b.lastRun
	if start.IsZero() {
		start = end.Add(-b.interval)
	}

	// Get captures from the period
	captures, err := b.store.GetByTimeRange(start, end)
	if err != nil {
		return nil, err
	}

	if len(captures) == 0 {
		b.lastRun = end
		return nil, nil
	}

	// Build context for LLM
	contextStr := b.buildContext(captures)

	// Call LLM
	insights, err := b.callLLM(ctx, contextStr, start, end)
	if err != nil {
		return nil, err
	}

	b.lastRun = end
	return insights, nil
}

func (b *BatchAnalyzer) buildContext(captures []storage.CaptureRecord) string {
	var sb strings.Builder
	sb.WriteString("Activity data from the last period:\n\n")

	// Group by source for cleaner presentation
	bySource := make(map[string][]storage.CaptureRecord)
	for _, c := range captures {
		bySource[c.Source] = append(bySource[c.Source], c)
	}

	// Windows - show app switches
	if windows := bySource["window"]; len(windows) > 0 {
		sb.WriteString("APPS USED:\n")
		seen := make(map[string]bool)
		for _, w := range windows {
			app := w.Metadata["app_class"]
			if app != "" && !seen[app] {
				seen[app] = true
				sb.WriteString(fmt.Sprintf("- %s\n", app))
			}
		}
		sb.WriteString("\n")
	}

	// Biometrics - summarize stress
	if biometrics := bySource["biometrics"]; len(biometrics) > 0 {
		sb.WriteString("STRESS LEVELS:\n")
		for _, b := range biometrics {
			level := b.Metadata["stress_level"]
			score := b.Metadata["stress_score"]
			if level != "" {
				sb.WriteString(fmt.Sprintf("- %s: %s (score: %s)\n", b.Timestamp.Format("15:04"), level, score))
			}
		}
		sb.WriteString("\n")
	}

	// Screen OCR - key activities
	if screens := bySource["screen"]; len(screens) > 0 {
		sb.WriteString("SCREEN ACTIVITY:\n")
		count := 0
		for _, s := range screens {
			if s.TextData != "" && count < 5 {
				sb.WriteString(fmt.Sprintf("- %s: %s\n", s.Timestamp.Format("15:04"), truncate(s.TextData, 100)))
				count++
			}
		}
		sb.WriteString("\n")
	}

	// Clipboard - what was copied
	if clips := bySource["clipboard"]; len(clips) > 0 {
		sb.WriteString("CLIPBOARD:\n")
		count := 0
		for _, c := range clips {
			if c.TextData != "" && count < 3 {
				sb.WriteString(fmt.Sprintf("- %s\n", truncate(c.TextData, 50)))
				count++
			}
		}
		sb.WriteString("\n")
	}

	// Git - coding activity
	if git := bySource["git"]; len(git) > 0 {
		sb.WriteString("GIT:\n")
		for _, g := range git[:min(3, len(git))] {
			if g.TextData != "" {
				sb.WriteString(fmt.Sprintf("- %s\n", truncate(g.TextData, 80)))
			}
		}
	}

	return sb.String()
}

const batchPrompt = `Analyze this activity snapshot and generate insights. Output JSON only.

%s

Generate 1-3 insights in this JSON format:
[
  {
    "type": "pattern|summary|anomaly",
    "title": "short title (max 50 chars)",
    "body": "actionable insight (max 150 chars)",
    "severity": "info|warning"
  }
]

Focus on:
1. Work patterns (deep focus, fragmentation, task switches)
2. Stress correlations (what activities associate with stress)
3. Productivity observations
4. Actionable suggestions

Be specific - mention actual apps, times, content. Skip obvious observations.
Output ONLY valid JSON array, no other text.`

func (b *BatchAnalyzer) callLLM(ctx context.Context, contextStr string, start, end time.Time) ([]Insight, error) {
	prompt := fmt.Sprintf(batchPrompt, contextStr)

	reqBody := map[string]any{
		"model": b.model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens":  500,
		"temperature": 0.3,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		"https://openrouter.ai/api/v1/chat/completions",
		bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+b.apiKey)
	req.Header.Set("HTTP-Referer", "https://github.com/Atharva-Kanherkar/mnemosyne")
	req.Header.Set("X-Title", "Mnemosyne Batch Analysis")

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	if result.Error != nil {
		return nil, fmt.Errorf("API error: %s", result.Error.Message)
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no response from LLM")
	}

	// Parse LLM response as JSON array
	content := strings.TrimSpace(result.Choices[0].Message.Content)

	// Try to extract JSON array
	startIdx := strings.Index(content, "[")
	endIdx := strings.LastIndex(content, "]")
	if startIdx >= 0 && endIdx > startIdx {
		content = content[startIdx : endIdx+1]
	}

	var rawInsights []struct {
		Type     string `json:"type"`
		Title    string `json:"title"`
		Body     string `json:"body"`
		Severity string `json:"severity"`
	}

	if err := json.Unmarshal([]byte(content), &rawInsights); err != nil {
		log.Printf("[batch] Failed to parse LLM response: %v\nContent: %s", err, content)
		return nil, nil // Don't fail, just skip this batch
	}

	// Convert to Insight structs
	insights := make([]Insight, 0, len(rawInsights))
	for _, r := range rawInsights {
		insightType := InsightTypeLLMSummary
		if r.Type == "pattern" {
			insightType = InsightTypePattern
		}

		severity := SeverityInfo
		if r.Severity == "warning" {
			severity = SeverityWarning
		}

		insights = append(insights, Insight{
			Type:           insightType,
			Severity:       severity,
			Title:          r.Title,
			Body:           r.Body,
			TriggerSource:  TriggerLLMBatch,
			CreatedAt:      time.Now(),
			TimeRangeStart: &start,
			TimeRangeEnd:   &end,
		})
	}

	log.Printf("[batch] Generated %d insights from LLM", len(insights))
	return insights, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
