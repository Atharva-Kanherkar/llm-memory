// Package query provides the query engine for searching and retrieving captures.
//
// The engine combines:
// - Database queries (time range, source type)
// - Vision-based OCR for screenshot text extraction (via GPT-4o-mini)
// - LLM for natural language queries
package query

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Atharva-Kanherkar/mnemosyne/internal/llm"
	"github.com/Atharva-Kanherkar/mnemosyne/internal/ocr"
)

// Engine handles queries over captured data.
type Engine struct {
	db        *sql.DB
	llm       *llm.Client
	visionOCR *ocr.VisionOCR
	ocrCache  map[string]string // Cache OCR results by file path
	Debug     bool
}

// debugLog logs a message if debug mode is enabled.
func (e *Engine) debugLog(format string, args ...interface{}) {
	if e.Debug {
		log.Printf("[engine] "+format, args...)
	}
}

// CaptureRecord represents a capture from the database.
type CaptureRecord struct {
	ID          int64
	Source      string
	Timestamp   time.Time
	TextData    string
	RawDataPath string
	Metadata    map[string]string
}

// New creates a new query Engine.
func New(db *sql.DB, llmClient *llm.Client) *Engine {
	return &Engine{
		db:       db,
		llm:      llmClient,
		ocrCache: make(map[string]string),
	}
}

// NewWithOCR creates a new query Engine with vision OCR support.
func NewWithOCR(db *sql.DB, llmClient *llm.Client, apiKey string) *Engine {
	return &Engine{
		db:        db,
		llm:       llmClient,
		visionOCR: ocr.NewVisionOCR(apiKey),
		ocrCache:  make(map[string]string),
	}
}

// GetRecent retrieves the most recent captures.
func (e *Engine) GetRecent(ctx context.Context, limit int) ([]CaptureRecord, error) {
	rows, err := e.db.QueryContext(ctx, `
		SELECT id, source, timestamp, text_data, raw_data_path, metadata
		FROM captures
		ORDER BY timestamp DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return e.scanRecords(rows)
}

// GetByTimeRange retrieves captures within a time range.
func (e *Engine) GetByTimeRange(ctx context.Context, start, end time.Time) ([]CaptureRecord, error) {
	rows, err := e.db.QueryContext(ctx, `
		SELECT id, source, timestamp, text_data, raw_data_path, metadata
		FROM captures
		WHERE timestamp BETWEEN ? AND ?
		ORDER BY timestamp DESC
	`, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return e.scanRecords(rows)
}

// GetBySource retrieves captures from a specific source.
func (e *Engine) GetBySource(ctx context.Context, source string, limit int) ([]CaptureRecord, error) {
	rows, err := e.db.QueryContext(ctx, `
		SELECT id, source, timestamp, text_data, raw_data_path, metadata
		FROM captures
		WHERE source = ?
		ORDER BY timestamp DESC
		LIMIT ?
	`, source, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return e.scanRecords(rows)
}

// SearchText searches for captures containing the given text.
func (e *Engine) SearchText(ctx context.Context, searchText string, limit int) ([]CaptureRecord, error) {
	// Search in text_data and metadata
	pattern := "%" + searchText + "%"
	rows, err := e.db.QueryContext(ctx, `
		SELECT id, source, timestamp, text_data, raw_data_path, metadata
		FROM captures
		WHERE text_data LIKE ? OR metadata LIKE ?
		ORDER BY timestamp DESC
		LIMIT ?
	`, pattern, pattern, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return e.scanRecords(rows)
}

// scanRecords scans database rows into CaptureRecord slice.
func (e *Engine) scanRecords(rows *sql.Rows) ([]CaptureRecord, error) {
	var records []CaptureRecord
	for rows.Next() {
		var r CaptureRecord
		var metadataJSON string
		var rawDataPath, textData sql.NullString

		err := rows.Scan(&r.ID, &r.Source, &r.Timestamp, &textData, &rawDataPath, &metadataJSON)
		if err != nil {
			return nil, err
		}

		r.TextData = textData.String
		r.RawDataPath = rawDataPath.String

		if metadataJSON != "" {
			json.Unmarshal([]byte(metadataJSON), &r.Metadata)
		}

		records = append(records, r)
	}

	return records, nil
}

// BuildContext builds a text context from captures for LLM queries.
func (e *Engine) BuildContext(ctx context.Context, records []CaptureRecord, includeOCR bool) string {
	var sb strings.Builder

	// Limit OCR to 5 most recent screenshots to avoid long waits
	ocrCount := 0
	const maxOCR = 5

	for _, r := range records {
		timestamp := r.Timestamp.Format("2006-01-02 15:04:05")
		sb.WriteString(fmt.Sprintf("\n[%s] %s:\n", timestamp, r.Source))

		switch r.Source {
		case "window":
			app := r.Metadata["app_class"]
			title := r.Metadata["window_title"]
			workspace := r.Metadata["workspace_name"]
			sb.WriteString(fmt.Sprintf("  App: %s\n  Title: %s\n  Workspace: %s\n", app, title, workspace))

		case "clipboard":
			contentType := r.Metadata["content_type"]
			if r.TextData != "" {
				// Truncate long clipboard content
				text := r.TextData
				if len(text) > 500 {
					text = text[:500] + "..."
				}
				sb.WriteString(fmt.Sprintf("  Type: %s\n  Content: %s\n", contentType, text))
			}

		case "git":
			repo := r.Metadata["repo_name"]
			branch := r.Metadata["branch"]
			commit := r.Metadata["commit"]
			sb.WriteString(fmt.Sprintf("  Repo: %s\n  Branch: %s\n  Commit: %s\n", repo, branch, commit))
			if r.TextData != "" {
				// Include uncommitted changes summary
				changes := r.TextData
				if len(changes) > 300 {
					changes = changes[:300] + "..."
				}
				sb.WriteString(fmt.Sprintf("  Changes: %s\n", changes))
			}

		case "screen":
			// Check for pre-computed OCR text first
			if r.TextData != "" {
				ocrText := r.TextData
				if len(ocrText) > 2000 {
					ocrText = ocrText[:2000] + "..."
				}
				sb.WriteString(fmt.Sprintf("  Screen content (pre-computed OCR):\n%s\n", ocrText))
			} else if includeOCR && r.RawDataPath != "" && ocrCount < maxOCR {
				// Fall back to on-demand OCR for old screenshots without pre-computed text
				ocrCount++
				e.debugLog("Processing screenshot %d/%d (no pre-computed OCR)", ocrCount, maxOCR)
				ocrText := e.getOCRText(ctx, r.RawDataPath)
				if ocrText != "" {
					if len(ocrText) > 2000 {
						ocrText = ocrText[:2000] + "..."
					}
					sb.WriteString(fmt.Sprintf("  Screen content (vision OCR):\n%s\n", ocrText))
				}
			} else if r.RawDataPath != "" {
				sb.WriteString("  [Screenshot captured - OCR skipped]\n")
			} else {
				sb.WriteString("  [Screenshot captured]\n")
			}

		case "activity":
			state := r.Metadata["state"]
			idleSeconds := r.Metadata["idle_seconds"]
			sb.WriteString(fmt.Sprintf("  State: %s (idle: %ss)\n", state, idleSeconds))

		case "biometrics":
			level := r.Metadata["stress_level"]
			score := r.Metadata["stress_score"]
			sb.WriteString(fmt.Sprintf("  Stress level: %s (score: %s/100)\n", level, score))

			// Include detailed metrics
			if jitter := r.Metadata["mouse_jitter"]; jitter != "" {
				sb.WriteString(fmt.Sprintf("  Mouse jitter: %s (>0.3 indicates stress)\n", jitter))
			}
			if pauses := r.Metadata["typing_pauses"]; pauses != "" {
				sb.WriteString(fmt.Sprintf("  Typing pauses: %s (>10 indicates stress)\n", pauses))
			}
			if errorRate := r.Metadata["typing_error_rate"]; errorRate != "" {
				sb.WriteString(fmt.Sprintf("  Typing error rate: %s (>0.15 indicates stress)\n", errorRate))
			}
			if switches := r.Metadata["window_switches_pm"]; switches != "" {
				sb.WriteString(fmt.Sprintf("  Window switches/min: %s (>3 indicates fragmented attention)\n", switches))
			}
			if rapid := r.Metadata["rapid_switches"]; rapid != "" {
				sb.WriteString(fmt.Sprintf("  Rapid switches (<5s): %s (>10 indicates anxiety)\n", rapid))
			}
			if r.TextData != "" {
				sb.WriteString(fmt.Sprintf("  Indicators: %s\n", r.TextData))
			}
		}
	}

	return sb.String()
}

// getOCRText extracts text from an image using vision model.
func (e *Engine) getOCRText(ctx context.Context, imagePath string) string {
	// Check cache first
	if text, ok := e.ocrCache[imagePath]; ok {
		e.debugLog("OCR cache hit for %s", imagePath)
		return text
	}

	// Check if file exists
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		return ""
	}

	// Use vision OCR if available
	if e.visionOCR != nil && e.visionOCR.Available() {
		e.debugLog("OCR extracting text from %s...", imagePath)
		text, err := e.visionOCR.ExtractTextFromFile(ctx, imagePath)
		if err != nil {
			e.debugLog("OCR error: %v", err)
			return fmt.Sprintf("[Vision OCR error: %v]", err)
		}
		e.debugLog("OCR extracted %d chars", len(text))
		// Cache the result
		e.ocrCache[imagePath] = text
		return text
	}

	return "[OCR not available - no API key]"
}

// Ask answers a natural language question about past activity.
func (e *Engine) Ask(ctx context.Context, question string) (string, error) {
	if e.llm == nil {
		return "", fmt.Errorf("LLM not configured")
	}

	context, err := e.buildQueryContext(ctx, question)
	if err != nil {
		return "", err
	}
	if context == "" {
		return "I don't have any captures for that time period.", nil
	}

	return e.llm.AnswerQuery(ctx, question, context)
}

// AskStream answers a question with streaming output.
func (e *Engine) AskStream(ctx context.Context, question string, onChunk func(string)) (string, error) {
	e.debugLog("AskStream called")
	if e.llm == nil {
		return "", fmt.Errorf("LLM not configured")
	}

	e.debugLog("Building query context...")
	context, err := e.buildQueryContext(ctx, question)
	if err != nil {
		e.debugLog("Error building context: %v", err)
		return "", err
	}
	if context == "" {
		e.debugLog("No captures found")
		msg := "I don't have any captures for that time period."
		if onChunk != nil {
			onChunk(msg)
		}
		return msg, nil
	}

	e.debugLog("Context built: %d chars. Calling LLM...", len(context))
	return e.llm.AnswerQueryStream(ctx, question, context, onChunk)
}

// buildQueryContext builds the context for a query.
func (e *Engine) buildQueryContext(ctx context.Context, question string) (string, error) {
	// Parse the question to determine time range
	timeRange := e.parseTimeRange(question)
	e.debugLog("Parsed time range: %v to %v", timeRange.start, timeRange.end)

	// Get relevant captures
	var records []CaptureRecord
	var err error

	if timeRange.start.IsZero() {
		e.debugLog("No time range, fetching last 100 captures")
		records, err = e.GetRecent(ctx, 100)
	} else {
		e.debugLog("Fetching captures in time range")
		records, err = e.GetByTimeRange(ctx, timeRange.start, timeRange.end)
	}

	if err != nil {
		return "", fmt.Errorf("failed to fetch captures: %w", err)
	}

	e.debugLog("Found %d records", len(records))
	if len(records) == 0 {
		return "", nil
	}

	// Count screenshots for OCR
	screenCount := 0
	for _, r := range records {
		if r.Source == "screen" {
			screenCount++
		}
	}
	e.debugLog("Building context (%d screenshots will use OCR)...", screenCount)

	// Build context (include OCR for screenshots)
	return e.BuildContext(ctx, records, true), nil
}

// timeRange represents a parsed time range from a query.
type timeRange struct {
	start time.Time
	end   time.Time
}

// parseTimeRange extracts time references from a question.
// This is a simple parser - could be enhanced with LLM parsing.
func (e *Engine) parseTimeRange(question string) timeRange {
	now := time.Now()
	lower := strings.ToLower(question)

	// Check for common patterns
	if strings.Contains(lower, "today") {
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		return timeRange{start: start, end: now}
	}

	if strings.Contains(lower, "yesterday") {
		yesterday := now.AddDate(0, 0, -1)
		start := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, now.Location())
		end := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 23, 59, 59, 0, now.Location())
		return timeRange{start: start, end: end}
	}

	if strings.Contains(lower, "last hour") || strings.Contains(lower, "past hour") {
		return timeRange{start: now.Add(-1 * time.Hour), end: now}
	}

	if strings.Contains(lower, "last 30 minutes") || strings.Contains(lower, "past 30 minutes") {
		return timeRange{start: now.Add(-30 * time.Minute), end: now}
	}

	if strings.Contains(lower, "this morning") {
		start := time.Date(now.Year(), now.Month(), now.Day(), 6, 0, 0, 0, now.Location())
		end := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, now.Location())
		return timeRange{start: start, end: end}
	}

	if strings.Contains(lower, "this afternoon") {
		start := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, now.Location())
		end := time.Date(now.Year(), now.Month(), now.Day(), 18, 0, 0, 0, now.Location())
		return timeRange{start: start, end: end}
	}

	if strings.Contains(lower, "this week") {
		// Go back to start of week (Monday)
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7 // Sunday
		}
		start := now.AddDate(0, 0, -(weekday - 1))
		start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, now.Location())
		return timeRange{start: start, end: now}
	}

	// Default: empty (will use recent captures)
	return timeRange{}
}

// Summarize generates a summary of recent activity.
func (e *Engine) Summarize(ctx context.Context, duration time.Duration) (string, error) {
	if e.llm == nil {
		return "", fmt.Errorf("LLM not configured")
	}

	context, err := e.buildSummaryContext(ctx, duration)
	if err != nil {
		return "", err
	}
	if context == "" {
		return "No activity captured in this time period.", nil
	}

	return e.llm.Summarize(ctx, context)
}

// SummarizeStream generates a summary with streaming output.
func (e *Engine) SummarizeStream(ctx context.Context, duration time.Duration, onChunk func(string)) (string, error) {
	if e.llm == nil {
		return "", fmt.Errorf("LLM not configured")
	}

	context, err := e.buildSummaryContext(ctx, duration)
	if err != nil {
		return "", err
	}
	if context == "" {
		msg := "No activity captured in this time period."
		if onChunk != nil {
			onChunk(msg)
		}
		return msg, nil
	}

	return e.llm.SummarizeStream(ctx, context, onChunk)
}

// buildSummaryContext builds context for a summary.
func (e *Engine) buildSummaryContext(ctx context.Context, duration time.Duration) (string, error) {
	start := time.Now().Add(-duration)
	records, err := e.GetByTimeRange(ctx, start, time.Now())
	if err != nil {
		return "", err
	}

	if len(records) == 0 {
		return "", nil
	}

	return e.BuildContext(ctx, records, false), nil // Skip OCR for speed
}

// Stats returns statistics about captured data.
func (e *Engine) Stats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total captures
	var total int64
	e.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM captures").Scan(&total)
	stats["total_captures"] = total

	// Captures by source
	bySource := make(map[string]int64)
	rows, err := e.db.QueryContext(ctx, "SELECT source, COUNT(*) FROM captures GROUP BY source")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var source string
			var count int64
			rows.Scan(&source, &count)
			bySource[source] = count
		}
	}
	stats["by_source"] = bySource

	// Time range
	var oldest, newest time.Time
	e.db.QueryRowContext(ctx, "SELECT MIN(timestamp), MAX(timestamp) FROM captures").Scan(&oldest, &newest)
	stats["oldest_capture"] = oldest
	stats["newest_capture"] = newest

	return stats, nil
}
