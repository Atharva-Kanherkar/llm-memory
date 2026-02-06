package timetable

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Atharva-Kanherkar/mnemosyne/internal/llm"
	"github.com/Atharva-Kanherkar/mnemosyne/internal/storage"
)

const (
	defaultBuilderModel = "openai/gpt-4o-mini"
	minQuestionAnswers  = 12
)

// GeneratedPlan is the persisted result of timetable planning.
type GeneratedPlan struct {
	Plan  storage.TimetablePlanRecord
	Items []storage.TimetableItemRecord
}

// Builder runs a detailed LLM interview and produces a timetable.
type Builder struct {
	store          *storage.Store
	llmClient      *llm.Client
	llmModel       string
	defaultEmailTo string
	editPlanID     string
	editCreatedAt  time.Time
	editActive     bool

	messages    []llm.Message
	phase       int
	answerCount int
	result      *GeneratedPlan
}

// NewBuilder creates a new timetable builder.
func NewBuilder(store *storage.Store, apiKey, llmModel, defaultEmailTo string) *Builder {
	if llmModel == "" {
		llmModel = defaultBuilderModel
	}

	client := llm.NewClient(apiKey)
	client.ChatModel = llmModel

	return &Builder{
		store:          store,
		llmClient:      client,
		llmModel:       llmModel,
		defaultEmailTo: defaultEmailTo,
		messages:       make([]llm.Message, 0, 24),
	}
}

const builderSystemPrompt = `You are Mnemosyne's timetable planning agent.

Your job is to run a detailed interview and then build a realistic timetable.

Hard requirements:
1. Ask one question at a time.
2. Ask at least 12 detailed questions before finalizing.
3. Cover all categories:
   - goals and priorities
   - fixed commitments (meetings/classes/meals) with exact times
   - task durations and dependencies
   - preferred focus blocks and break cadence
   - energy peaks/lows and sleep window
   - location/commute constraints
   - contingency/buffer preferences
   - reminder preferences (desktop + email)
   - weekday/weekend/holiday differences
   - desired plan horizon/date
4. Keep each question concise and specific.
5. Only output final JSON once enough detail is collected.

When finalizing, output JSON only inside a fenced block and use this exact schema:
` + "```json" + `
{
  "plan_name": "string",
  "goal": "string",
  "timezone": "IANA timezone, e.g. America/Los_Angeles",
  "email_to": "user@example.com or empty",
  "applicable_weekdays": [0,1,2,3,4,5,6],
  "recurrence": {
    "enabled": true,
    "interval_days": 1
  },
  "summary": "short planning summary",
  "tasks": [
    {
      "title": "string",
      "details": "string",
      "start_time": "RFC3339 datetime with timezone offset",
      "end_time": "RFC3339 datetime with timezone offset",
      "notify_at": "RFC3339 datetime with timezone offset",
      "notify_desktop": true,
      "notify_email": false
    }
  ]
}
` + "```" + `

Scheduling constraints:
- No overlapping tasks.
- Include short breaks for long focus periods.
- Include buffer blocks around risky transitions.
- notify_at must be at or before start_time.
- If user did not explicitly request email reminders, default notify_email to false.
- If user asks for daily recurring plan, set recurrence.enabled=true and interval_days=1.
- Use applicable_weekdays:
  - weekdays = [1,2,3,4,5]
  - weekends = [0,6]
  - all days = [0,1,2,3,4,5,6]
`

// Start initializes the interview and returns the first question.
func (b *Builder) Start() string {
	b.messages = []llm.Message{
		{Role: "system", Content: builderSystemPrompt},
	}
	b.phase = 1
	b.answerCount = 0
	b.result = nil

	response, err := b.chat("Start the detailed interview now. Ask question 1.")
	if err != nil {
		return "What date should this timetable cover, and what are your top 1-3 priorities?"
	}

	return response
}

// StartEdit begins an edit conversation for an existing plan.
func (b *Builder) StartEdit(plan storage.TimetablePlanRecord, items []storage.TimetableItemRecord) string {
	b.messages = []llm.Message{
		{Role: "system", Content: builderSystemPrompt},
	}
	b.phase = 1
	b.answerCount = 0
	b.result = nil
	b.editPlanID = plan.ID
	b.editCreatedAt = plan.CreatedAt
	b.editActive = plan.Active

	type recurrence struct {
		Enabled      bool `json:"enabled"`
		IntervalDays int  `json:"interval_days"`
	}
	type itemPayload struct {
		Title         string `json:"title"`
		Details       string `json:"details"`
		StartTime     string `json:"start_time"`
		EndTime       string `json:"end_time"`
		NotifyAt      string `json:"notify_at"`
		NotifyDesktop bool   `json:"notify_desktop"`
		NotifyEmail   bool   `json:"notify_email"`
	}
	payload := struct {
		PlanName           string        `json:"plan_name"`
		Goal               string        `json:"goal"`
		Timezone           string        `json:"timezone"`
		EmailTo            string        `json:"email_to"`
		ApplicableWeekdays []int         `json:"applicable_weekdays"`
		Recurrence         recurrence    `json:"recurrence"`
		Tasks              []itemPayload `json:"tasks"`
	}{
		PlanName:           plan.Name,
		Goal:               plan.Goal,
		Timezone:           plan.Timezone,
		EmailTo:            plan.EmailTo,
		ApplicableWeekdays: storage.WeekdayMaskToSlice(plan.WeekdayMask),
		Recurrence: recurrence{
			Enabled:      plan.RecurrenceEnabled,
			IntervalDays: plan.RecurrenceDays,
		},
		Tasks: make([]itemPayload, 0, len(items)),
	}

	for _, it := range items {
		payload.Tasks = append(payload.Tasks, itemPayload{
			Title:         it.Title,
			Details:       it.Details,
			StartTime:     it.StartTime.Format(time.RFC3339),
			EndTime:       it.EndTime.Format(time.RFC3339),
			NotifyAt:      it.NotifyAt.Format(time.RFC3339),
			NotifyDesktop: it.NotifyDesktop,
			NotifyEmail:   it.NotifyEmail,
		})
	}

	raw, _ := json.MarshalIndent(payload, "", "  ")
	seedPrompt := fmt.Sprintf(
		"We are editing an existing timetable. Current plan JSON:\n%s\n\nAsk specific follow-up questions to modify this plan. After enough detail, output the full updated JSON in the required format.",
		string(raw),
	)

	response, err := b.chat(seedPrompt)
	if err != nil {
		return "What exactly would you like to change in the current timetable?"
	}
	return response
}

// Chat sends a user answer and returns the next agent message.
func (b *Builder) Chat(userInput string) (string, *GeneratedPlan, error) {
	if strings.TrimSpace(userInput) != "" {
		b.answerCount++
	}

	response, err := b.chat(userInput)
	if err != nil {
		return "", nil, err
	}

	draft := b.extractDraft(response)
	if draft == nil {
		return response, nil, nil
	}

	if b.answerCount < minQuestionAnswers {
		followup, followErr := b.chat(
			"Do not finalize yet. Ask another specific question to collect more constraints.",
		)
		if followErr != nil {
			return "I still need more detail. What fixed commitments do you have with exact start/end times?", nil, nil
		}
		return followup, nil, nil
	}

	generated, err := b.persistDraft(draft)
	if err != nil {
		return "", nil, err
	}

	b.phase = 2
	b.result = generated
	return b.cleanResponse(response), generated, nil
}

func (b *Builder) chat(userInput string) (string, error) {
	trimmed := strings.TrimSpace(userInput)
	if trimmed != "" {
		b.messages = append(b.messages, llm.Message{Role: "user", Content: trimmed})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	response, err := b.llmClient.Chat(ctx, b.messages)
	if err != nil {
		return "", err
	}

	response = strings.TrimSpace(response)
	b.messages = append(b.messages, llm.Message{Role: "assistant", Content: response})
	return response, nil
}

type draftPlan struct {
	PlanName           string `json:"plan_name"`
	Goal               string `json:"goal"`
	Timezone           string `json:"timezone"`
	EmailTo            string `json:"email_to"`
	ApplicableWeekdays []int  `json:"applicable_weekdays"`
	Recurrence         struct {
		Enabled      bool `json:"enabled"`
		IntervalDays int  `json:"interval_days"`
	} `json:"recurrence"`
	Summary string      `json:"summary"`
	Tasks   []draftTask `json:"tasks"`
}

type draftTask struct {
	Title         string `json:"title"`
	Details       string `json:"details"`
	StartTime     string `json:"start_time"`
	EndTime       string `json:"end_time"`
	NotifyAt      string `json:"notify_at"`
	NotifyDesktop bool   `json:"notify_desktop"`
	NotifyEmail   bool   `json:"notify_email"`
}

func (b *Builder) extractDraft(response string) *draftPlan {
	startMarker := "```json"
	endMarker := "```"

	start := strings.Index(response, startMarker)
	if start == -1 {
		return nil
	}
	jsonStart := start + len(startMarker)
	rest := response[jsonStart:]
	end := strings.Index(rest, endMarker)
	if end == -1 {
		return nil
	}

	jsonStr := strings.TrimSpace(rest[:end])
	if jsonStr == "" {
		return nil
	}

	var draft draftPlan
	if err := json.Unmarshal([]byte(jsonStr), &draft); err != nil {
		return nil
	}

	if strings.TrimSpace(draft.PlanName) == "" || len(draft.Tasks) == 0 {
		return nil
	}

	return &draft
}

func (b *Builder) persistDraft(draft *draftPlan) (*GeneratedPlan, error) {
	loc := time.UTC
	tz := strings.TrimSpace(draft.Timezone)
	if tz == "" {
		tz = "UTC"
	}
	if loaded, err := time.LoadLocation(tz); err == nil {
		loc = loaded
	} else {
		tz = "UTC"
	}

	planID := strings.TrimSpace(b.editPlanID)
	createdAt := b.editCreatedAt
	active := b.editActive
	if planID == "" {
		var err error
		planID, err = randomID()
		if err != nil {
			return nil, err
		}
		createdAt = time.Now()
		active = true
	}

	emailTo := strings.TrimSpace(draft.EmailTo)
	if emailTo == "" {
		emailTo = strings.TrimSpace(b.defaultEmailTo)
	}

	questionnairePayload := map[string]any{
		"summary":       draft.Summary,
		"answers_count": b.answerCount,
		"messages":      b.messages,
	}
	questionnaireJSON, _ := json.Marshal(questionnairePayload)
	recurrenceDays := draft.Recurrence.IntervalDays
	if recurrenceDays <= 0 {
		recurrenceDays = 1
	}
	weekdayMask := storage.WeekdayMaskFromSlice(draft.ApplicableWeekdays)

	planRecord := storage.TimetablePlanRecord{
		ID:                planID,
		Name:              strings.TrimSpace(draft.PlanName),
		Goal:              strings.TrimSpace(draft.Goal),
		Timezone:          tz,
		EmailTo:           emailTo,
		RecurrenceEnabled: draft.Recurrence.Enabled,
		RecurrenceDays:    recurrenceDays,
		WeekdayMask:       weekdayMask,
		QuestionnaireJSON: string(questionnaireJSON),
		Active:            active,
		CreatedAt:         createdAt,
		UpdatedAt:         time.Now(),
	}

	items := make([]storage.TimetableItemRecord, 0, len(draft.Tasks))
	for _, t := range draft.Tasks {
		title := strings.TrimSpace(t.Title)
		if title == "" {
			continue
		}

		startTime, err := parseFlexibleTime(t.StartTime, loc)
		if err != nil {
			return nil, fmt.Errorf("invalid start_time for task %q: %w", title, err)
		}
		endTime, err := parseFlexibleTime(t.EndTime, loc)
		if err != nil {
			return nil, fmt.Errorf("invalid end_time for task %q: %w", title, err)
		}
		if !endTime.After(startTime) {
			return nil, fmt.Errorf("end_time must be after start_time for task %q", title)
		}

		notifyAt, err := parseFlexibleTime(t.NotifyAt, loc)
		if err != nil {
			notifyAt = startTime.Add(-10 * time.Minute)
		}
		if notifyAt.After(startTime) {
			notifyAt = startTime.Add(-1 * time.Minute)
		}

		items = append(items, storage.TimetableItemRecord{
			PlanID:        planID,
			Title:         title,
			Details:       strings.TrimSpace(t.Details),
			StartTime:     startTime,
			EndTime:       endTime,
			NotifyAt:      notifyAt,
			NotifyDesktop: t.NotifyDesktop,
			NotifyEmail:   t.NotifyEmail,
		})
	}

	if len(items) == 0 {
		return nil, fmt.Errorf("no valid timetable tasks generated")
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].StartTime.Before(items[j].StartTime)
	})
	for i := 1; i < len(items); i++ {
		if items[i].StartTime.Before(items[i-1].EndTime) {
			return nil, fmt.Errorf("generated tasks overlap: %q and %q", items[i-1].Title, items[i].Title)
		}
	}

	if err := b.store.SaveTimetablePlan(&planRecord, items); err != nil {
		return nil, err
	}

	return &GeneratedPlan{Plan: planRecord, Items: items}, nil
}

func parseFlexibleTime(raw string, loc *time.Location) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, fmt.Errorf("empty datetime")
	}

	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
	}

	for _, layout := range layouts {
		if ts, err := time.Parse(layout, raw); err == nil {
			return ts.In(loc), nil
		}
		if ts, err := time.ParseInLocation(layout, raw, loc); err == nil {
			return ts, nil
		}
	}

	return time.Time{}, fmt.Errorf("unsupported datetime format: %q", raw)
}

func randomID() (string, error) {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (b *Builder) cleanResponse(response string) string {
	startMarker := "```json"
	start := strings.Index(response, startMarker)
	if start == -1 {
		return response
	}

	end := strings.Index(response[start+len(startMarker):], "```")
	if end == -1 {
		return strings.TrimSpace(response[:start])
	}

	before := strings.TrimSpace(response[:start])
	afterStart := start + len(startMarker) + end + len("```")
	after := strings.TrimSpace(response[afterStart:])

	if before == "" {
		before = "Timetable ready."
	}
	if after != "" {
		return before + "\n\n" + after
	}
	return before
}

// IsComplete indicates whether the builder has produced a plan.
func (b *Builder) IsComplete() bool {
	return b.phase == 2 && b.result != nil
}

// GetResult returns the generated plan if available.
func (b *Builder) GetResult() *GeneratedPlan {
	return b.result
}

// Reset clears conversation state.
func (b *Builder) Reset() {
	b.messages = nil
	b.phase = 0
	b.answerCount = 0
	b.result = nil
	b.editPlanID = ""
	b.editCreatedAt = time.Time{}
	b.editActive = false
}
