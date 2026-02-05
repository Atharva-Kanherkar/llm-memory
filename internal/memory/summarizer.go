// Package memory provides persistent memory through periodic summarization.
//
// Architecture:
// - Raw captures: High detail, used for recent queries (last 30 min)
// - Hourly summaries: Compressed, ~50 tokens each, used for "today" queries
// - Daily summaries: Highly compressed, ~100 tokens, used for "this week" queries
//
// This mimics how human memory works - recent events are detailed,
// older events are compressed into gist.
package memory

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

// SummaryType represents the granularity of a summary.
type SummaryType string

const (
	SummaryHourly SummaryType = "hourly"
	SummaryDaily  SummaryType = "daily"
)

// Summary represents a compressed memory summary.
type Summary struct {
	ID        int64       `json:"id"`
	Type      SummaryType `json:"type"`
	StartTime time.Time   `json:"start_time"`
	EndTime   time.Time   `json:"end_time"`
	Content   string      `json:"content"` // Compressed summary text
	Apps      string      `json:"apps"`    // Comma-separated app list
	Tokens    int         `json:"tokens"`  // Approximate token count
	CreatedAt time.Time   `json:"created_at"`
}

// Summarizer creates periodic summaries of activity.
type Summarizer struct {
	store      *storage.Store
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewSummarizer creates a new memory summarizer.
func NewSummarizer(store *storage.Store, apiKey string) *Summarizer {
	return &Summarizer{
		store:      store,
		apiKey:     apiKey,
		model:      "deepseek/deepseek-chat", // Very cheap for compression
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

// Run starts the periodic summarization loop.
func (s *Summarizer) Run(ctx context.Context) {
	// Run hourly summarization
	hourlyTicker := time.NewTicker(30 * time.Minute) // Check every 30 min, summarize completed hours
	defer hourlyTicker.Stop()

	// Run daily summarization
	dailyTicker := time.NewTicker(1 * time.Hour) // Check every hour for completed days
	defer dailyTicker.Stop()

	// Initial run after startup delay
	time.Sleep(2 * time.Minute)
	s.summarizeRecentHours(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-hourlyTicker.C:
			s.summarizeRecentHours(ctx)
		case <-dailyTicker.C:
			s.summarizeYesterday(ctx)
		}
	}
}

// summarizeRecentHours summarizes any completed hours that don't have summaries yet.
func (s *Summarizer) summarizeRecentHours(ctx context.Context) {
	now := time.Now()

	// Check last 12 hours for missing summaries
	for i := 1; i <= 12; i++ {
		hourStart := time.Date(now.Year(), now.Month(), now.Day(), now.Hour()-i, 0, 0, 0, now.Location())
		hourEnd := hourStart.Add(1 * time.Hour)

		// Skip if this hour isn't complete yet
		if hourEnd.After(now) {
			continue
		}

		// Check if summary exists
		exists, _ := s.store.SummaryExists(string(SummaryHourly), hourStart)
		if exists {
			continue
		}

		// Create summary for this hour
		if err := s.createHourlySummary(ctx, hourStart, hourEnd); err != nil {
			log.Printf("[memory] Failed to summarize hour %s: %v", hourStart.Format("15:04"), err)
		}
	}
}

// summarizeYesterday creates a daily summary for yesterday if it doesn't exist.
func (s *Summarizer) summarizeYesterday(ctx context.Context) {
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	dayStart := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, now.Location())
	dayEnd := dayStart.AddDate(0, 0, 1)

	// Check if summary exists
	exists, _ := s.store.SummaryExists(string(SummaryDaily), dayStart)
	if exists {
		return
	}

	if err := s.createDailySummary(ctx, dayStart, dayEnd); err != nil {
		log.Printf("[memory] Failed to summarize day %s: %v", dayStart.Format("2006-01-02"), err)
	}
}

func (s *Summarizer) createHourlySummary(ctx context.Context, start, end time.Time) error {
	// Get captures for this hour
	captures, err := s.store.GetByTimeRange(start, end)
	if err != nil {
		return err
	}

	if len(captures) == 0 {
		return nil // No activity, no summary needed
	}

	// Build context from captures
	contextStr := s.buildCaptureContext(captures)

	// Generate summary using LLM
	summary, err := s.compressSummary(ctx, contextStr, "hour")
	if err != nil {
		return err
	}

	// Extract apps used
	apps := s.extractApps(captures)

	// Save to database
	return s.store.SaveSummary(&storage.SummaryRecord{
		Type:      string(SummaryHourly),
		StartTime: start,
		EndTime:   end,
		Content:   summary,
		Apps:      apps,
		Tokens:    len(strings.Fields(summary)) * 2, // Rough estimate
	})
}

func (s *Summarizer) createDailySummary(ctx context.Context, start, end time.Time) error {
	// Get hourly summaries for this day
	summaries, err := s.store.GetSummariesByRange(string(SummaryHourly), start, end)
	if err != nil || len(summaries) == 0 {
		// Fall back to raw captures if no hourly summaries
		captures, err := s.store.GetByTimeRange(start, end)
		if err != nil || len(captures) == 0 {
			return nil
		}
		contextStr := s.buildCaptureContext(captures)
		summary, err := s.compressSummary(ctx, contextStr, "day")
		if err != nil {
			return err
		}
		return s.store.SaveSummary(&storage.SummaryRecord{
			Type:      string(SummaryDaily),
			StartTime: start,
			EndTime:   end,
			Content:   summary,
			Apps:      s.extractApps(captures),
			Tokens:    len(strings.Fields(summary)) * 2,
		})
	}

	// Combine hourly summaries
	var combined strings.Builder
	var allApps []string
	for _, sum := range summaries {
		combined.WriteString(fmt.Sprintf("[%s] %s\n", sum.StartTime.Format("15:04"), sum.Content))
		if sum.Apps != "" {
			allApps = append(allApps, sum.Apps)
		}
	}

	// Compress into daily summary
	summary, err := s.compressSummary(ctx, combined.String(), "day")
	if err != nil {
		return err
	}

	return s.store.SaveSummary(&storage.SummaryRecord{
		Type:      string(SummaryDaily),
		StartTime: start,
		EndTime:   end,
		Content:   summary,
		Apps:      strings.Join(unique(allApps), ","),
		Tokens:    len(strings.Fields(summary)) * 2,
	})
}

func (s *Summarizer) buildCaptureContext(captures []storage.CaptureRecord) string {
	var sb strings.Builder

	// Group by source
	bySource := make(map[string][]storage.CaptureRecord)
	for _, c := range captures {
		bySource[c.Source] = append(bySource[c.Source], c)
	}

	// Windows - show app switches
	if windows := bySource["window"]; len(windows) > 0 {
		sb.WriteString("APPS: ")
		seen := make(map[string]bool)
		var apps []string
		for _, w := range windows {
			app := w.Metadata["app_class"]
			if app != "" && !seen[app] {
				seen[app] = true
				apps = append(apps, app)
			}
		}
		sb.WriteString(strings.Join(apps, ", "))
		sb.WriteString("\n")
	}

	// Screen OCR - key activities (sample)
	if screens := bySource["screen"]; len(screens) > 0 {
		sb.WriteString("ACTIVITY:\n")
		step := len(screens) / 5 // Sample ~5 screenshots
		if step < 1 {
			step = 1
		}
		for i := 0; i < len(screens) && i/step < 5; i += step {
			s := screens[i]
			if s.TextData != "" {
				text := s.TextData
				if len(text) > 150 {
					text = text[:147] + "..."
				}
				sb.WriteString(fmt.Sprintf("- %s: %s\n", s.Timestamp.Format("15:04"), text))
			}
		}
	}

	// Clipboard
	if clips := bySource["clipboard"]; len(clips) > 0 {
		sb.WriteString("CLIPBOARD: ")
		count := 0
		for _, c := range clips {
			if c.TextData != "" && count < 3 {
				text := c.TextData
				if len(text) > 50 {
					text = text[:47] + "..."
				}
				sb.WriteString(fmt.Sprintf("\"%s\" ", text))
				count++
			}
		}
		sb.WriteString("\n")
	}

	// Git
	if git := bySource["git"]; len(git) > 0 {
		sb.WriteString("GIT: ")
		seen := make(map[string]bool)
		for _, g := range git {
			repo := g.Metadata["repo_name"]
			if repo != "" && !seen[repo] {
				seen[repo] = true
				sb.WriteString(repo + " ")
			}
		}
		sb.WriteString("\n")
	}

	// Stress
	if bio := bySource["biometrics"]; len(bio) > 0 {
		highStress := 0
		for _, b := range bio {
			level := b.Metadata["stress_level"]
			if level == "high" || level == "anxious" || level == "elevated" {
				highStress++
			}
		}
		if highStress > 0 {
			sb.WriteString(fmt.Sprintf("STRESS: %d elevated periods\n", highStress))
		}
	}

	return sb.String()
}

func (s *Summarizer) extractApps(captures []storage.CaptureRecord) string {
	seen := make(map[string]bool)
	var apps []string
	for _, c := range captures {
		if c.Source == "window" {
			app := c.Metadata["app_class"]
			if app != "" && !seen[app] {
				seen[app] = true
				apps = append(apps, app)
			}
		}
	}
	return strings.Join(apps, ",")
}

const hourlyPrompt = `Compress this hour's computer activity into 1-2 sentences (max 50 words). Focus on: what was the main task, key accomplishments, any notable events. Be specific with app names and content.

Activity:
%s

Compressed summary (1-2 sentences):`

const dailyPrompt = `Compress this day's activity into a brief paragraph (max 100 words). Include: main projects/tasks worked on, key accomplishments, overall productivity pattern, any notable events.

Activity summaries:
%s

Daily summary (1 paragraph):`

func (s *Summarizer) compressSummary(ctx context.Context, content, period string) (string, error) {
	var prompt string
	if period == "hour" {
		prompt = fmt.Sprintf(hourlyPrompt, content)
	} else {
		prompt = fmt.Sprintf(dailyPrompt, content)
	}

	reqBody := map[string]any{
		"model": s.model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens":  150,
		"temperature": 0.3,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		"https://openrouter.ai/api/v1/chat/completions",
		bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("HTTP-Referer", "https://github.com/Atharva-Kanherkar/mnemosyne")
	req.Header.Set("X-Title", "Mnemosyne Memory")

	resp, err := s.httpClient.Do(req)
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

	summary := strings.TrimSpace(result.Choices[0].Message.Content)
	log.Printf("[memory] Created %s summary: %s", period, truncate(summary, 80))
	return summary, nil
}

// GetDayContext returns compressed context for a full day query.
func (s *Summarizer) GetDayContext(ctx context.Context, date time.Time) (string, error) {
	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	dayEnd := dayStart.AddDate(0, 0, 1)
	now := time.Now()

	var sb strings.Builder

	// Get hourly summaries for the day
	summaries, err := s.store.GetSummariesByRange(string(SummaryHourly), dayStart, dayEnd)
	if err == nil && len(summaries) > 0 {
		sb.WriteString("=== Activity Timeline ===\n")
		for _, sum := range summaries {
			sb.WriteString(fmt.Sprintf("[%s] %s\n", sum.StartTime.Format("15:04"), sum.Content))
		}
		sb.WriteString("\n")
	}

	// For recent hours (last 2 hours), include raw captures for detail
	twoHoursAgo := now.Add(-2 * time.Hour)
	if dayEnd.After(twoHoursAgo) {
		recentStart := twoHoursAgo
		if recentStart.Before(dayStart) {
			recentStart = dayStart
		}
		captures, err := s.store.GetByTimeRange(recentStart, now)
		if err == nil && len(captures) > 0 {
			sb.WriteString("=== Recent Detail ===\n")
			sb.WriteString(s.buildCaptureContext(captures))
		}
	}

	return sb.String(), nil
}

// ForceHourlySummary forces creation of a summary for the current hour (for testing).
func (s *Summarizer) ForceHourlySummary(ctx context.Context) error {
	now := time.Now()
	hourStart := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())
	hourEnd := now
	return s.createHourlySummary(ctx, hourStart, hourEnd)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func unique(slice []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range slice {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
