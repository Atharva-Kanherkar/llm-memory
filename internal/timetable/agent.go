package timetable

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Atharva-Kanherkar/mnemosyne/internal/config"
	"github.com/Atharva-Kanherkar/mnemosyne/internal/notify"
	"github.com/Atharva-Kanherkar/mnemosyne/internal/storage"
)

// Agent monitors due timetable items and dispatches reminders.
type Agent struct {
	store            *storage.Store
	notifier         *notify.DesktopNotifier
	emailSender      EmailSender
	pollInterval     time.Duration
	lookbackWindow   time.Duration
	desktopEnabled   bool
	emailEnabled     bool
	defaultEmailTo   string
	emailUnavailable bool
}

// NewAgent creates a new timetable reminder agent.
func NewAgent(store *storage.Store, cfg config.TimetableConfig) *Agent {
	pollSeconds := cfg.ReminderCheckSeconds
	if pollSeconds <= 0 {
		pollSeconds = 30
	}

	lookback := time.Duration(cfg.ReminderLookbackMinutes) * time.Minute
	if lookback <= 0 {
		lookback = 6 * time.Hour
	}

	return &Agent{
		store:          store,
		notifier:       notify.NewDesktopNotifier(),
		emailSender:    NewSMTPSender(cfg),
		pollInterval:   time.Duration(pollSeconds) * time.Second,
		lookbackWindow: lookback,
		desktopEnabled: cfg.DesktopNotifications,
		emailEnabled:   cfg.EmailNotifications,
		defaultEmailTo: strings.TrimSpace(cfg.DefaultEmailTo),
	}
}

// Run starts the reminder loop.
func (a *Agent) Run(ctx context.Context) {
	if a.store == nil {
		log.Println("[timetable] Agent disabled: storage is nil")
		return
	}

	log.Printf("[timetable] Reminder agent started (interval: %s)", a.pollInterval)
	a.runOnce(ctx)

	ticker := time.NewTicker(a.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[timetable] Reminder agent stopped")
			return
		case <-ticker.C:
			a.runOnce(ctx)
		}
	}
}

func (a *Agent) runOnce(ctx context.Context) {
	now := time.Now()
	events, err := a.store.GetDueTimetableItems(now, a.lookbackWindow)
	if err != nil {
		log.Printf("[timetable] Failed to load due reminders: %v", err)
		return
	}

	if len(events) == 0 {
		return
	}

	senderAvailable := a.emailSender != nil && a.emailSender.Available()
	if a.emailEnabled && !senderAvailable && !a.emailUnavailable {
		a.emailUnavailable = true
		log.Printf("[timetable] Email reminders enabled but SMTP sender is unavailable")
	}
	if senderAvailable {
		a.emailUnavailable = false
	}

	overrideCache := map[string]string{}
	for _, event := range events {
		applicable, err := a.isEventApplicable(event, overrideCache)
		if err != nil {
			log.Printf("[timetable] applicability check failed for item %d: %v", event.ID, err)
			continue
		}
		if !applicable {
			a.skipEvent(event)
			continue
		}

		desktopDone := a.sendDesktopReminder(event)
		emailDone := a.sendEmailReminder(ctx, event, senderAvailable)
		a.rollRecurringIfNeeded(event, desktopDone, emailDone)
	}
}

func (a *Agent) sendDesktopReminder(event storage.TimetableEventRecord) bool {
	if event.NotifiedDesktopAt != nil {
		return true
	}
	if !event.NotifyDesktop {
		return true
	}

	if !a.desktopEnabled {
		_ = a.store.MarkTimetableDesktopNotified(event.ID)
		return true
	}

	title := fmt.Sprintf("Time for %s", event.Title)
	body := fmt.Sprintf("%s (%s)", event.PlanName, event.StartTime.Format("15:04"))
	if strings.TrimSpace(event.Details) != "" {
		body = event.Details
	}

	if err := a.notifier.Send(title, body, notify.UrgencyNormal); err != nil {
		log.Printf("[timetable] Desktop notification failed for item %d: %v", event.ID, err)
		return false
	}

	if err := a.store.MarkTimetableDesktopNotified(event.ID); err != nil {
		log.Printf("[timetable] Failed to mark desktop reminder sent for item %d: %v", event.ID, err)
		return false
	}
	return true
}

func (a *Agent) sendEmailReminder(ctx context.Context, event storage.TimetableEventRecord, senderAvailable bool) bool {
	if event.NotifiedEmailAt != nil {
		return true
	}
	if !event.NotifyEmail {
		return true
	}

	if !a.emailEnabled {
		_ = a.store.MarkTimetableEmailNotified(event.ID)
		return true
	}

	to := strings.TrimSpace(event.PlanEmailTo)
	if to == "" {
		to = a.defaultEmailTo
	}
	if to == "" {
		// No recipient available, treat as opted-out for this item.
		_ = a.store.MarkTimetableEmailNotified(event.ID)
		return true
	}

	if !senderAvailable {
		return false
	}

	subject := fmt.Sprintf("[Mnemosyne] %s", event.Title)
	body := a.buildEmailBody(event)

	sendCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	if err := a.emailSender.Send(sendCtx, to, subject, body); err != nil {
		log.Printf("[timetable] Email reminder failed for item %d: %v", event.ID, err)
		return false
	}

	if err := a.store.MarkTimetableEmailNotified(event.ID); err != nil {
		log.Printf("[timetable] Failed to mark email reminder sent for item %d: %v", event.ID, err)
		return false
	}
	return true
}

func (a *Agent) buildEmailBody(event storage.TimetableEventRecord) string {
	lines := []string{
		"Mnemosyne timetable reminder",
		"",
		"Task: " + event.Title,
		"Plan: " + event.PlanName,
		"Start: " + event.StartTime.Format(time.RFC1123),
		"End: " + event.EndTime.Format(time.RFC1123),
	}

	if strings.TrimSpace(event.Details) != "" {
		lines = append(lines, "Details: "+event.Details)
	}

	if strings.TrimSpace(event.PlanGoal) != "" {
		lines = append(lines, "Goal: "+event.PlanGoal)
	}

	lines = append(lines, "", "Sent by Mnemosyne timetable agent.")
	return strings.Join(lines, "\n")
}

func (a *Agent) rollRecurringIfNeeded(event storage.TimetableEventRecord, desktopDone, emailDone bool) {
	if !event.PlanRecurring {
		return
	}
	if !desktopDone || !emailDone {
		return
	}

	if err := a.store.RollRecurringTimetableItem(event.ID, event.PlanRecurrenceDays, event.PlanWeekdayMask); err != nil {
		log.Printf("[timetable] Failed to roll recurring item %d: %v", event.ID, err)
	}
}

func (a *Agent) isEventApplicable(event storage.TimetableEventRecord, overrideCache map[string]string) (bool, error) {
	dateKey, weekday := eventDateKey(event)

	overridePlanID, ok := overrideCache[dateKey]
	if !ok {
		var err error
		overridePlanID, err = a.store.GetTimetableDayOverride(dateKey)
		if err != nil {
			return false, err
		}
		overrideCache[dateKey] = overridePlanID
	}

	if overridePlanID != "" {
		return overridePlanID == event.PlanID, nil
	}

	return storage.WeekdayInMask(event.PlanWeekdayMask, weekday), nil
}

func (a *Agent) skipEvent(event storage.TimetableEventRecord) {
	if event.PlanRecurring {
		if err := a.store.ForceRollRecurringTimetableItem(event.ID, event.PlanRecurrenceDays, event.PlanWeekdayMask); err != nil {
			log.Printf("[timetable] Failed to defer recurring item %d: %v", event.ID, err)
		}
		return
	}

	// One-off items that are not applicable today should not keep retrying.
	if event.NotifyDesktop && event.NotifiedDesktopAt == nil {
		_ = a.store.MarkTimetableDesktopNotified(event.ID)
	}
	if event.NotifyEmail && event.NotifiedEmailAt == nil {
		_ = a.store.MarkTimetableEmailNotified(event.ID)
	}
}

func eventDateKey(event storage.TimetableEventRecord) (string, time.Weekday) {
	loc := time.Local
	if tz := strings.TrimSpace(event.PlanTimezone); tz != "" {
		if loaded, err := time.LoadLocation(tz); err == nil {
			loc = loaded
		}
	}

	ts := event.StartTime.In(loc)
	return ts.Format("2006-01-02"), ts.Weekday()
}
