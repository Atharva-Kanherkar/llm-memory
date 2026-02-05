package insights

import (
	"fmt"
	"strings"
	"time"

	"github.com/Atharva-Kanherkar/mnemosyne/internal/capture"
	"github.com/Atharva-Kanherkar/mnemosyne/internal/capture/biometrics"
)

// Rule evaluates captures and produces insights.
type Rule interface {
	Name() string
	Evaluate(ctx *RuleContext) *Insight
}

// RuleContext provides data for rule evaluation.
type RuleContext struct {
	LatestCapture  *capture.Result
	RecentCaptures []capture.Result
	StressSnapshot *biometrics.StressSnapshot
	CurrentWindow  string
	CurrentApp     string
	IdleSeconds    int
	WasIdle        bool
}

// StressSpikeRule detects rapid stress increases.
type StressSpikeRule struct {
	lastScore float64
	lastLevel biometrics.StressLevel
	lastTime  time.Time
}

func NewStressSpikeRule() *StressSpikeRule {
	return &StressSpikeRule{}
}

func (r *StressSpikeRule) Name() string {
	return "stress_spike"
}

func (r *StressSpikeRule) Evaluate(ctx *RuleContext) *Insight {
	if ctx.StressSnapshot == nil {
		return nil
	}

	currentLevel := ctx.StressSnapshot.Level
	currentScore := ctx.StressSnapshot.OverallScore

	// Detect spike: from calm/normal (<35) to elevated/high (>55) within 2 minutes
	if currentScore > 55 && r.lastScore < 35 && !r.lastTime.IsZero() && time.Since(r.lastTime) < 2*time.Minute {
		insight := &Insight{
			Type:          InsightTypeStressAlert,
			Severity:      SeverityUrgent,
			Title:         "Stress spike detected",
			Body:          fmt.Sprintf("Stress jumped from %.0f to %.0f. %s", r.lastScore, currentScore, formatIndicators(ctx.StressSnapshot.TopIndicators)),
			TriggerSource: TriggerRule,
			CreatedAt:     time.Now(),
		}

		r.lastScore = currentScore
		r.lastLevel = currentLevel
		r.lastTime = time.Now()

		return insight
	}

	// Update tracking
	r.lastScore = currentScore
	r.lastLevel = currentLevel
	r.lastTime = time.Now()
	return nil
}

// SustainedStressRule detects prolonged high stress.
type SustainedStressRule struct {
	highStressSince time.Time
	alerted         bool
}

func NewSustainedStressRule() *SustainedStressRule {
	return &SustainedStressRule{}
}

func (r *SustainedStressRule) Name() string {
	return "sustained_stress"
}

func (r *SustainedStressRule) Evaluate(ctx *RuleContext) *Insight {
	if ctx.StressSnapshot == nil {
		return nil
	}

	isHighStress := ctx.StressSnapshot.OverallScore > 55

	if isHighStress {
		if r.highStressSince.IsZero() {
			r.highStressSince = time.Now()
		} else if !r.alerted && time.Since(r.highStressSince) > 10*time.Minute {
			r.alerted = true
			duration := int(time.Since(r.highStressSince).Minutes())
			return &Insight{
				Type:          InsightTypeStressAlert,
				Severity:      SeverityWarning,
				Title:         "Take a break?",
				Body:          fmt.Sprintf("You've been stressed for %d+ minutes. %s", duration, formatIndicators(ctx.StressSnapshot.TopIndicators)),
				TriggerSource: TriggerRule,
				CreatedAt:     time.Now(),
			}
		}
	} else {
		// Reset when stress drops
		r.highStressSince = time.Time{}
		r.alerted = false
	}

	return nil
}

// ContextSwitchRule provides reminders when returning from break.
type ContextSwitchRule struct {
	lastActiveWindow string
	lastActiveApp    string
	lastActiveTime   time.Time
	wasIdle          bool
	idleStartTime    time.Time
}

func NewContextSwitchRule() *ContextSwitchRule {
	return &ContextSwitchRule{}
}

func (r *ContextSwitchRule) Name() string {
	return "context_switch"
}

func (r *ContextSwitchRule) Evaluate(ctx *RuleContext) *Insight {
	// Track idle state
	isIdle := ctx.IdleSeconds > 30

	if isIdle && !r.wasIdle {
		// Just went idle - save context
		r.wasIdle = true
		r.idleStartTime = time.Now()
		r.lastActiveWindow = ctx.CurrentWindow
		r.lastActiveApp = ctx.CurrentApp
		return nil
	}

	if !isIdle && r.wasIdle {
		// Just returned from idle
		idleDuration := time.Since(r.idleStartTime)
		r.wasIdle = false

		// Only notify if idle for 5+ minutes
		if idleDuration > 5*time.Minute && r.lastActiveWindow != "" {
			insight := &Insight{
				Type:          InsightTypeContextReminder,
				Severity:      SeverityInfo,
				Title:         "Welcome back!",
				Body:          fmt.Sprintf("You were away for %d minutes. Before: %s", int(idleDuration.Minutes()), truncate(r.lastActiveWindow, 50)),
				TriggerSource: TriggerRule,
				CreatedAt:     time.Now(),
			}
			return insight
		}
	}

	// Update current context if not idle
	if !isIdle {
		r.lastActiveWindow = ctx.CurrentWindow
		r.lastActiveApp = ctx.CurrentApp
		r.lastActiveTime = time.Now()
	}

	return nil
}

// DeepWorkRule detects sustained focus.
type DeepWorkRule struct {
	singleAppSince time.Time
	lastApp        string
	notified       bool
}

func NewDeepWorkRule() *DeepWorkRule {
	return &DeepWorkRule{}
}

func (r *DeepWorkRule) Name() string {
	return "deep_work"
}

func (r *DeepWorkRule) Evaluate(ctx *RuleContext) *Insight {
	if ctx.CurrentApp == "" {
		return nil
	}

	// Same app as before
	if ctx.CurrentApp == r.lastApp {
		if r.singleAppSince.IsZero() {
			r.singleAppSince = time.Now()
		}

		// Notify after 30 minutes of focus (once)
		if !r.notified && time.Since(r.singleAppSince) > 30*time.Minute {
			r.notified = true
			duration := int(time.Since(r.singleAppSince).Minutes())
			return &Insight{
				Type:          InsightTypeDeepWork,
				Severity:      SeverityInfo,
				Title:         "Deep work session",
				Body:          fmt.Sprintf("Focused on %s for %d minutes. Great flow!", r.lastApp, duration),
				TriggerSource: TriggerRule,
				CreatedAt:     time.Now(),
			}
		}
	} else {
		// App changed - reset
		r.lastApp = ctx.CurrentApp
		r.singleAppSince = time.Now()
		r.notified = false
	}

	return nil
}

// RapidSwitchingRule detects anxiety-like window switching.
type RapidSwitchingRule struct {
	recentSwitches []time.Time
	alerted        bool
	lastAlertTime  time.Time
}

func NewRapidSwitchingRule() *RapidSwitchingRule {
	return &RapidSwitchingRule{
		recentSwitches: make([]time.Time, 0),
	}
}

func (r *RapidSwitchingRule) Name() string {
	return "rapid_switching"
}

func (r *RapidSwitchingRule) Evaluate(ctx *RuleContext) *Insight {
	if ctx.StressSnapshot == nil {
		return nil
	}

	rapidSwitches := ctx.StressSnapshot.Context.RapidSwitches

	// Alert if >10 rapid switches and haven't alerted in last 30 min
	if rapidSwitches > 10 && (r.lastAlertTime.IsZero() || time.Since(r.lastAlertTime) > 30*time.Minute) {
		r.lastAlertTime = time.Now()
		return &Insight{
			Type:          InsightTypeStressAlert,
			Severity:      SeverityWarning,
			Title:         "Fragmented attention",
			Body:          fmt.Sprintf("%d rapid window switches detected. Try focusing on one task.", rapidSwitches),
			TriggerSource: TriggerRule,
			CreatedAt:     time.Now(),
		}
	}

	return nil
}

func formatIndicators(indicators []string) string {
	if len(indicators) == 0 {
		return ""
	}
	return "Signs: " + strings.Join(indicators, ", ")
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
