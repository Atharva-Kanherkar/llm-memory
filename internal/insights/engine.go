package insights

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Atharva-Kanherkar/mnemosyne/internal/capture"
	"github.com/Atharva-Kanherkar/mnemosyne/internal/capture/biometrics"
	"github.com/Atharva-Kanherkar/mnemosyne/internal/notify"
	"github.com/Atharva-Kanherkar/mnemosyne/internal/storage"
)

// Engine orchestrates insight generation and delivery.
type Engine struct {
	store           *storage.Store
	desktopNotifier *notify.DesktopNotifier
	socketServer    *notify.SocketServer

	// Rule engine
	rules       []Rule
	ruleContext *RuleContext

	// Batch analyzer (optional, needs LLM)
	batchAnalyzer *BatchAnalyzer
	batchInterval time.Duration

	// Configuration
	desktopEnabled bool
	socketPath     string

	mu sync.RWMutex
}

// EngineConfig configures the insight engine.
type EngineConfig struct {
	Store           *storage.Store
	SocketPath      string
	DesktopEnabled  bool
	BatchInterval   time.Duration
	LLMAPIKey       string
	LLMModel        string
}

// NewEngine creates a new insight engine.
func NewEngine(cfg EngineConfig) *Engine {
	e := &Engine{
		store:          cfg.Store,
		socketPath:     cfg.SocketPath,
		desktopEnabled: cfg.DesktopEnabled,
		batchInterval:  cfg.BatchInterval,
		ruleContext:    &RuleContext{},
	}

	// Initialize desktop notifier
	if cfg.DesktopEnabled {
		e.desktopNotifier = notify.NewDesktopNotifier()
	}

	// Initialize socket server
	if cfg.SocketPath != "" {
		e.socketServer = notify.NewSocketServer(cfg.SocketPath)
	}

	// Register rules
	e.rules = []Rule{
		NewStressSpikeRule(),
		NewSustainedStressRule(),
		NewContextSwitchRule(),
		NewDeepWorkRule(),
		NewRapidSwitchingRule(),
	}

	// Initialize batch analyzer if LLM configured
	if cfg.LLMAPIKey != "" && cfg.BatchInterval > 0 {
		model := cfg.LLMModel
		if model == "" {
			model = "deepseek/deepseek-chat" // Default cheap model
		}
		e.batchAnalyzer = NewBatchAnalyzer(cfg.Store, cfg.LLMAPIKey, model, cfg.BatchInterval)
	}

	return e
}

// Start begins the insight engine.
func (e *Engine) Start(ctx context.Context) error {
	// Start socket server
	if e.socketServer != nil {
		if err := e.socketServer.Start(); err != nil {
			log.Printf("[insights] Socket server failed to start: %v", err)
			// Continue anyway - desktop notifications still work
		} else {
			log.Printf("[insights] Socket server listening on %s", e.socketPath)
		}
	}

	// Start batch analyzer
	if e.batchAnalyzer != nil {
		go e.batchAnalyzer.Run(ctx, func(insight *Insight) {
			e.processInsight(insight)
		})
		log.Printf("[insights] Batch analyzer started (interval: %v)", e.batchInterval)
	}

	log.Printf("[insights] Engine started with %d rules", len(e.rules))
	return nil
}

// Stop shuts down the insight engine.
func (e *Engine) Stop() {
	if e.socketServer != nil {
		e.socketServer.Stop()
	}
}

// ProcessCapture is called by the daemon for each new capture.
// This runs the rule engine for immediate alerts.
func (e *Engine) ProcessCapture(result *capture.Result, stressSnapshot *biometrics.StressSnapshot) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Update rule context
	e.ruleContext.LatestCapture = result
	e.ruleContext.StressSnapshot = stressSnapshot

	if result != nil {
		if result.Source == "window" {
			if app, ok := result.Metadata["app_class"]; ok {
				e.ruleContext.CurrentApp = app
			}
			if title, ok := result.Metadata["window_title"]; ok {
				e.ruleContext.CurrentWindow = title
			}
		}
		if result.Source == "activity" {
			if idle, ok := result.Metadata["idle_seconds"]; ok {
				// Parse idle seconds
				var idleSec int
				fmt.Sscanf(idle, "%d", &idleSec)
				e.ruleContext.IdleSeconds = idleSec
			}
		}
	}

	// Evaluate all rules
	for _, rule := range e.rules {
		if insight := rule.Evaluate(e.ruleContext); insight != nil {
			e.processInsight(insight)
		}
	}
}

// processInsight handles a generated insight.
func (e *Engine) processInsight(insight *Insight) {
	// Store in database
	record := &storage.InsightRecord{
		Type:            string(insight.Type),
		Severity:        string(insight.Severity),
		Title:           insight.Title,
		Body:            insight.Body,
		TriggerSource:   string(insight.TriggerSource),
		RelatedCaptures: insight.RelatedCaptures,
		Metadata:        insight.Metadata,
		TimeRangeStart:  insight.TimeRangeStart,
		TimeRangeEnd:    insight.TimeRangeEnd,
	}

	id, err := e.store.SaveInsight(record)
	if err != nil {
		log.Printf("[insights] Failed to save insight: %v", err)
		return
	}
	insight.ID = id

	log.Printf("[insights] Generated: [%s] %s", insight.Severity, insight.Title)

	// Desktop notification for urgent/warning alerts
	if e.desktopNotifier != nil {
		switch insight.Severity {
		case SeverityUrgent:
			e.desktopNotifier.Send(insight.Title, insight.Body, notify.UrgencyCritical)
			e.store.MarkInsightNotified(id, true, false)
		case SeverityWarning:
			e.desktopNotifier.Send(insight.Title, insight.Body, notify.UrgencyNormal)
			e.store.MarkInsightNotified(id, true, false)
		}
	}

	// Push to connected TUI clients
	if e.socketServer != nil {
		e.socketServer.Broadcast(SocketMessage{
			Type:      MsgTypeInsight,
			Timestamp: time.Now(),
			Payload:   insight,
		})
	}
}

// GetRecentInsights returns recent unacknowledged insights.
func (e *Engine) GetRecentInsights(limit int) ([]Insight, error) {
	records, err := e.store.GetRecentInsights(limit)
	if err != nil {
		return nil, err
	}

	insights := make([]Insight, len(records))
	for i, r := range records {
		insights[i] = Insight{
			ID:              r.ID,
			Type:            InsightType(r.Type),
			Severity:        Severity(r.Severity),
			Title:           r.Title,
			Body:            r.Body,
			TriggerSource:   TriggerSource(r.TriggerSource),
			RelatedCaptures: r.RelatedCaptures,
			Metadata:        r.Metadata,
			CreatedAt:       r.CreatedAt,
			TimeRangeStart:  r.TimeRangeStart,
			TimeRangeEnd:    r.TimeRangeEnd,
			AcknowledgedAt:  r.AcknowledgedAt,
			NotifiedDesktop: r.NotifiedDesktop,
			NotifiedTUI:     r.NotifiedTUI,
		}
	}

	return insights, nil
}

// AcknowledgeInsight marks an insight as seen.
func (e *Engine) AcknowledgeInsight(id int64) error {
	return e.store.AcknowledgeInsight(id)
}
