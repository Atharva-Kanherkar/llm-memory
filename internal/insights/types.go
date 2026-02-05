// Package insights provides proactive analysis and notifications.
package insights

import (
	"time"
)

// InsightType categorizes insights
type InsightType string

const (
	InsightTypeStressAlert     InsightType = "stress_alert"
	InsightTypeContextReminder InsightType = "context_reminder"
	InsightTypePattern         InsightType = "pattern"
	InsightTypeLLMSummary      InsightType = "llm_summary"
	InsightTypeDeepWork        InsightType = "deep_work"
)

// Severity indicates urgency level
type Severity string

const (
	SeverityInfo    Severity = "info"    // Blue - informational
	SeverityWarning Severity = "warning" // Yellow - attention needed
	SeverityUrgent  Severity = "urgent"  // Red - immediate action
)

// TriggerSource indicates what generated the insight
type TriggerSource string

const (
	TriggerRule     TriggerSource = "rule"
	TriggerLLMBatch TriggerSource = "llm_batch"
)

// Insight represents a proactive notification or observation
type Insight struct {
	ID              int64         `json:"id"`
	Type            InsightType   `json:"type"`
	Severity        Severity      `json:"severity"`
	Title           string        `json:"title"`
	Body            string        `json:"body"`
	TriggerSource   TriggerSource `json:"trigger_source"`
	RelatedCaptures []int64       `json:"related_captures,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	CreatedAt       time.Time     `json:"created_at"`
	TimeRangeStart  *time.Time    `json:"time_range_start,omitempty"`
	TimeRangeEnd    *time.Time    `json:"time_range_end,omitempty"`
	AcknowledgedAt  *time.Time    `json:"acknowledged_at,omitempty"`
	NotifiedDesktop bool          `json:"notified_desktop"`
	NotifiedTUI     bool          `json:"notified_tui"`
}

// SocketMessage is used for daemon-TUI communication
type SocketMessage struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Payload   any       `json:"payload"`
}

// Message types for socket communication
const (
	MsgTypeInsight     = "insight"
	MsgTypeStressAlert = "stress"
	MsgTypeHeartbeat   = "heartbeat"
	MsgTypeSubscribe   = "subscribe"
	MsgTypeAck         = "ack"
)
