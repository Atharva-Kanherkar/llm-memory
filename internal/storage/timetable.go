package storage

import (
	"database/sql"
	"fmt"
	"sort"
	"time"
)

const (
	// WeekdayMaskAll means plan is applicable every day.
	// Bit positions use Go's time.Weekday: Sunday=0 ... Saturday=6.
	WeekdayMaskAll = (1 << 7) - 1
)

// TimetablePlanRecord represents a generated timetable plan.
type TimetablePlanRecord struct {
	ID                string
	Name              string
	Goal              string
	Timezone          string
	EmailTo           string
	RecurrenceEnabled bool
	RecurrenceDays    int
	WeekdayMask       int
	QuestionnaireJSON string
	Active            bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// TimetableItemRecord represents a single scheduled task in a timetable plan.
type TimetableItemRecord struct {
	ID                int64
	PlanID            string
	Title             string
	Details           string
	StartTime         time.Time
	EndTime           time.Time
	NotifyAt          time.Time
	NotifyDesktop     bool
	NotifyEmail       bool
	NotifiedDesktopAt *time.Time
	NotifiedEmailAt   *time.Time
	CreatedAt         time.Time
}

// TimetableEventRecord is a timetable item joined with plan metadata.
type TimetableEventRecord struct {
	ID                 int64
	PlanID             string
	PlanName           string
	PlanGoal           string
	PlanTimezone       string
	PlanEmailTo        string
	PlanRecurring      bool
	PlanRecurrenceDays int
	PlanWeekdayMask    int
	Title              string
	Details            string
	StartTime          time.Time
	EndTime            time.Time
	NotifyAt           time.Time
	NotifyDesktop      bool
	NotifyEmail        bool
	NotifiedDesktopAt  *time.Time
	NotifiedEmailAt    *time.Time
}

// TimetableDayOverrideRecord maps one date to a specific plan.
type TimetableDayOverrideRecord struct {
	Date      string
	PlanID    string
	PlanName  string
	CreatedAt time.Time
}

// WeekdayMaskFromSlice converts weekday numbers (0=Sunday..6=Saturday) to bitmask.
func WeekdayMaskFromSlice(days []int) int {
	mask := 0
	for _, day := range days {
		if day < 0 || day > 6 {
			continue
		}
		mask |= 1 << day
	}
	if mask == 0 {
		return WeekdayMaskAll
	}
	return mask
}

// WeekdayMaskToSlice converts bitmask to sorted weekday numbers.
func WeekdayMaskToSlice(mask int) []int {
	if mask <= 0 {
		mask = WeekdayMaskAll
	}
	out := make([]int, 0, 7)
	for day := 0; day <= 6; day++ {
		if mask&(1<<day) != 0 {
			out = append(out, day)
		}
	}
	sort.Ints(out)
	return out
}

// WeekdayInMask checks whether weekday is allowed by mask.
func WeekdayInMask(mask int, weekday time.Weekday) bool {
	if mask <= 0 {
		mask = WeekdayMaskAll
	}
	day := int(weekday)
	return mask&(1<<day) != 0
}

// SaveTimetablePlan stores a timetable plan and replaces all items for that plan.
func (s *Store) SaveTimetablePlan(plan *TimetablePlanRecord, items []TimetableItemRecord) error {
	if plan == nil {
		return fmt.Errorf("plan is nil")
	}
	if plan.ID == "" {
		return fmt.Errorf("plan id is required")
	}
	if plan.Name == "" {
		return fmt.Errorf("plan name is required")
	}
	if plan.Timezone == "" {
		plan.Timezone = "UTC"
	}
	if plan.RecurrenceDays <= 0 {
		plan.RecurrenceDays = 1
	}
	if plan.WeekdayMask <= 0 {
		plan.WeekdayMask = WeekdayMaskAll
	}

	now := time.Now()
	if plan.CreatedAt.IsZero() {
		plan.CreatedAt = now
	}
	if plan.UpdatedAt.IsZero() {
		plan.UpdatedAt = now
	}

	activeInt := 0
	if plan.Active {
		activeInt = 1
	}
	recurringInt := 0
	if plan.RecurrenceEnabled {
		recurringInt = 1
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO timetable_plans
		(id, name, goal, timezone, email_to, recurrence_enabled, recurrence_interval_days, weekday_mask, questionnaire_json, active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name=excluded.name,
			goal=excluded.goal,
			timezone=excluded.timezone,
			email_to=excluded.email_to,
			recurrence_enabled=excluded.recurrence_enabled,
			recurrence_interval_days=excluded.recurrence_interval_days,
			weekday_mask=excluded.weekday_mask,
			questionnaire_json=excluded.questionnaire_json,
			active=excluded.active,
			updated_at=excluded.updated_at
	`, plan.ID, plan.Name, plan.Goal, plan.Timezone, plan.EmailTo,
		recurringInt, plan.RecurrenceDays, plan.WeekdayMask, plan.QuestionnaireJSON, activeInt, plan.CreatedAt, plan.UpdatedAt)
	if err != nil {
		return err
	}

	// Replace all items atomically.
	if _, err := tx.Exec(`DELETE FROM timetable_items WHERE plan_id = ?`, plan.ID); err != nil {
		return err
	}

	for _, item := range items {
		notifyDesktop := 0
		notifyEmail := 0
		if item.NotifyDesktop {
			notifyDesktop = 1
		}
		if item.NotifyEmail {
			notifyEmail = 1
		}

		_, err := tx.Exec(`
			INSERT INTO timetable_items
			(plan_id, title, details, start_time, end_time, notify_at, notify_desktop, notify_email)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, plan.ID, item.Title, item.Details, item.StartTime, item.EndTime,
			item.NotifyAt, notifyDesktop, notifyEmail)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// ListTimetablePlans returns recent timetable plans.
func (s *Store) ListTimetablePlans(limit int) ([]TimetablePlanRecord, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := s.db.Query(`
		SELECT id, name, goal, timezone, email_to, recurrence_enabled, recurrence_interval_days, weekday_mask, questionnaire_json, active, created_at, updated_at
		FROM timetable_plans
		ORDER BY updated_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plans []TimetablePlanRecord
	for rows.Next() {
		var p TimetablePlanRecord
		var active, recurring int
		var recurrenceDays sql.NullInt64
		var weekdayMask sql.NullInt64
		var goal, emailTo, questionnaire sql.NullString

		if err := rows.Scan(&p.ID, &p.Name, &goal, &p.Timezone, &emailTo,
			&recurring, &recurrenceDays, &weekdayMask, &questionnaire, &active,
			&p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		p.Goal = goal.String
		p.EmailTo = emailTo.String
		p.QuestionnaireJSON = questionnaire.String
		p.RecurrenceEnabled = recurring == 1
		if recurrenceDays.Valid {
			p.RecurrenceDays = int(recurrenceDays.Int64)
		}
		if p.RecurrenceDays <= 0 {
			p.RecurrenceDays = 1
		}
		if weekdayMask.Valid {
			p.WeekdayMask = int(weekdayMask.Int64)
		}
		if p.WeekdayMask <= 0 {
			p.WeekdayMask = WeekdayMaskAll
		}
		p.Active = active == 1
		plans = append(plans, p)
	}

	return plans, nil
}

// GetTimetablePlan retrieves a timetable plan by id.
func (s *Store) GetTimetablePlan(id string) (*TimetablePlanRecord, error) {
	row := s.db.QueryRow(`
		SELECT id, name, goal, timezone, email_to, recurrence_enabled, recurrence_interval_days, weekday_mask, questionnaire_json, active, created_at, updated_at
		FROM timetable_plans
		WHERE id = ?
	`, id)

	var p TimetablePlanRecord
	var active, recurring int
	var recurrenceDays sql.NullInt64
	var weekdayMask sql.NullInt64
	var goal, emailTo, questionnaire sql.NullString

	if err := row.Scan(&p.ID, &p.Name, &goal, &p.Timezone, &emailTo,
		&recurring, &recurrenceDays, &weekdayMask, &questionnaire, &active,
		&p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, err
	}
	p.Goal = goal.String
	p.EmailTo = emailTo.String
	p.QuestionnaireJSON = questionnaire.String
	p.RecurrenceEnabled = recurring == 1
	if recurrenceDays.Valid {
		p.RecurrenceDays = int(recurrenceDays.Int64)
	}
	if p.RecurrenceDays <= 0 {
		p.RecurrenceDays = 1
	}
	if weekdayMask.Valid {
		p.WeekdayMask = int(weekdayMask.Int64)
	}
	if p.WeekdayMask <= 0 {
		p.WeekdayMask = WeekdayMaskAll
	}
	p.Active = active == 1
	return &p, nil
}

// GetTimetableItems returns all items for a plan.
func (s *Store) GetTimetableItems(planID string) ([]TimetableItemRecord, error) {
	rows, err := s.db.Query(`
		SELECT id, plan_id, title, details, start_time, end_time, notify_at,
		       notify_desktop, notify_email, notified_desktop_at, notified_email_at, created_at
		FROM timetable_items
		WHERE plan_id = ?
		ORDER BY start_time ASC
	`, planID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTimetableItems(rows)
}

// GetUpcomingTimetableItems returns upcoming timetable events across all active plans.
func (s *Store) GetUpcomingTimetableItems(start, end time.Time, limit int) ([]TimetableEventRecord, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := s.db.Query(`
		SELECT i.id, i.plan_id, p.name, p.goal, p.timezone, p.email_to,
		       p.recurrence_enabled, p.recurrence_interval_days, p.weekday_mask,
		       i.title, i.details, i.start_time, i.end_time, i.notify_at,
		       i.notify_desktop, i.notify_email, i.notified_desktop_at, i.notified_email_at
		FROM timetable_items i
		JOIN timetable_plans p ON p.id = i.plan_id
		WHERE p.active = 1
		  AND i.start_time BETWEEN ? AND ?
		ORDER BY i.start_time ASC
		LIMIT ?
	`, start, end, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTimetableEvents(rows)
}

// GetDueTimetableItems returns reminders that should be sent now.
func (s *Store) GetDueTimetableItems(now time.Time, lookback time.Duration) ([]TimetableEventRecord, error) {
	if lookback <= 0 {
		lookback = 24 * time.Hour
	}
	from := now.Add(-lookback)

	rows, err := s.db.Query(`
		SELECT i.id, i.plan_id, p.name, p.goal, p.timezone, p.email_to,
		       p.recurrence_enabled, p.recurrence_interval_days, p.weekday_mask,
		       i.title, i.details, i.start_time, i.end_time, i.notify_at,
		       i.notify_desktop, i.notify_email, i.notified_desktop_at, i.notified_email_at
		FROM timetable_items i
		JOIN timetable_plans p ON p.id = i.plan_id
		WHERE p.active = 1
		  AND i.notify_at BETWEEN ? AND ?
		  AND (
		       (i.notify_desktop = 1 AND i.notified_desktop_at IS NULL)
		    OR (i.notify_email = 1 AND i.notified_email_at IS NULL)
		  )
		ORDER BY i.notify_at ASC
	`, from, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTimetableEvents(rows)
}

// MarkTimetableDesktopNotified marks desktop reminder as delivered.
func (s *Store) MarkTimetableDesktopNotified(itemID int64) error {
	_, err := s.db.Exec(`
		UPDATE timetable_items
		SET notified_desktop_at = COALESCE(notified_desktop_at, CURRENT_TIMESTAMP)
		WHERE id = ?
	`, itemID)
	return err
}

// MarkTimetableEmailNotified marks email reminder as delivered.
func (s *Store) MarkTimetableEmailNotified(itemID int64) error {
	_, err := s.db.Exec(`
		UPDATE timetable_items
		SET notified_email_at = COALESCE(notified_email_at, CURRENT_TIMESTAMP)
		WHERE id = ?
	`, itemID)
	return err
}

// SetTimetablePlanActive toggles a timetable plan.
func (s *Store) SetTimetablePlanActive(planID string, active bool) error {
	activeInt := 0
	if active {
		activeInt = 1
	}

	_, err := s.db.Exec(`
		UPDATE timetable_plans
		SET active = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, activeInt, planID)
	return err
}

// SetTimetablePlanWeekdayMask updates the day applicability mask for a plan.
func (s *Store) SetTimetablePlanWeekdayMask(planID string, weekdayMask int) error {
	if weekdayMask <= 0 {
		weekdayMask = WeekdayMaskAll
	}

	_, err := s.db.Exec(`
		UPDATE timetable_plans
		SET weekday_mask = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, weekdayMask, planID)
	return err
}

// SetTimetableDayOverride forces a specific plan for a calendar date (YYYY-MM-DD).
func (s *Store) SetTimetableDayOverride(date, planID string) error {
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return fmt.Errorf("invalid date %q (expected YYYY-MM-DD)", date)
	}

	_, err := s.db.Exec(`
		INSERT INTO timetable_day_overrides (override_date, plan_id)
		VALUES (?, ?)
		ON CONFLICT(override_date) DO UPDATE SET
			plan_id = excluded.plan_id
	`, date, planID)
	return err
}

// ClearTimetableDayOverride removes override for a date.
func (s *Store) ClearTimetableDayOverride(date string) error {
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return fmt.Errorf("invalid date %q (expected YYYY-MM-DD)", date)
	}
	_, err := s.db.Exec(`DELETE FROM timetable_day_overrides WHERE override_date = ?`, date)
	return err
}

// GetTimetableDayOverride returns plan id override for a date, if any.
func (s *Store) GetTimetableDayOverride(date string) (string, error) {
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return "", fmt.Errorf("invalid date %q (expected YYYY-MM-DD)", date)
	}

	var planID string
	err := s.db.QueryRow(`
		SELECT plan_id FROM timetable_day_overrides WHERE override_date = ?
	`, date).Scan(&planID)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return planID, nil
}

// ListTimetableDayOverrides lists overrides in date range (inclusive).
func (s *Store) ListTimetableDayOverrides(startDate, endDate string, limit int) ([]TimetableDayOverrideRecord, error) {
	if _, err := time.Parse("2006-01-02", startDate); err != nil {
		return nil, fmt.Errorf("invalid start date %q (expected YYYY-MM-DD)", startDate)
	}
	if _, err := time.Parse("2006-01-02", endDate); err != nil {
		return nil, fmt.Errorf("invalid end date %q (expected YYYY-MM-DD)", endDate)
	}
	if limit <= 0 {
		limit = 50
	}

	rows, err := s.db.Query(`
		SELECT o.override_date, o.plan_id, p.name, o.created_at
		FROM timetable_day_overrides o
		JOIN timetable_plans p ON p.id = o.plan_id
		WHERE o.override_date BETWEEN ? AND ?
		ORDER BY o.override_date ASC
		LIMIT ?
	`, startDate, endDate, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := []TimetableDayOverrideRecord{}
	for rows.Next() {
		var r TimetableDayOverrideRecord
		if err := rows.Scan(&r.Date, &r.PlanID, &r.PlanName, &r.CreatedAt); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, nil
}

// DeleteTimetablePlan deletes a timetable plan and all its items.
func (s *Store) DeleteTimetablePlan(planID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM timetable_day_overrides WHERE plan_id = ?`, planID); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM timetable_items WHERE plan_id = ?`, planID); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM timetable_plans WHERE id = ?`, planID); err != nil {
		return err
	}
	return tx.Commit()
}

// RollRecurringTimetableItem shifts a completed recurring item into the future.
func (s *Store) RollRecurringTimetableItem(itemID int64, intervalDays int, weekdayMask int) error {
	return s.shiftRecurringTimetableItem(itemID, intervalDays, weekdayMask, true)
}

// ForceRollRecurringTimetableItem shifts a recurring item even if reminder wasn't delivered.
func (s *Store) ForceRollRecurringTimetableItem(itemID int64, intervalDays int, weekdayMask int) error {
	return s.shiftRecurringTimetableItem(itemID, intervalDays, weekdayMask, false)
}

func (s *Store) shiftRecurringTimetableItem(itemID int64, intervalDays int, weekdayMask int, requireDelivered bool) error {
	if intervalDays <= 0 {
		intervalDays = 1
	}
	if weekdayMask <= 0 {
		weekdayMask = WeekdayMaskAll
	}

	row := s.db.QueryRow(`
		SELECT start_time, end_time, notify_at, notify_desktop, notify_email,
		       notified_desktop_at, notified_email_at
		FROM timetable_items
		WHERE id = ?
	`, itemID)

	var startTime, endTime, notifyAt time.Time
	var notifyDesktop, notifyEmail int
	var notifiedDesktopAt, notifiedEmailAt sql.NullTime
	if err := row.Scan(&startTime, &endTime, &notifyAt, &notifyDesktop, &notifyEmail, &notifiedDesktopAt, &notifiedEmailAt); err != nil {
		return err
	}

	if requireDelivered {
		desktopDone := notifyDesktop == 0 || notifiedDesktopAt.Valid
		emailDone := notifyEmail == 0 || notifiedEmailAt.Valid
		if !desktopDone || !emailDone {
			return nil
		}
	}

	nextStart := startTime.AddDate(0, 0, intervalDays)
	for !WeekdayInMask(weekdayMask, nextStart.Weekday()) {
		nextStart = nextStart.AddDate(0, 0, 1)
	}
	dayShift := int(nextStart.Sub(startTime).Hours() / 24)
	if dayShift < 1 {
		dayShift = intervalDays
	}
	nextStart = startTime.AddDate(0, 0, dayShift)
	nextEnd := endTime.AddDate(0, 0, intervalDays)
	nextNotify := notifyAt.AddDate(0, 0, intervalDays)
	if dayShift != intervalDays {
		nextEnd = endTime.AddDate(0, 0, dayShift)
		nextNotify = notifyAt.AddDate(0, 0, dayShift)
	}

	_, err := s.db.Exec(`
		UPDATE timetable_items
		SET start_time = ?, end_time = ?, notify_at = ?,
		    notified_desktop_at = NULL, notified_email_at = NULL
		WHERE id = ?
	`, nextStart, nextEnd, nextNotify, itemID)
	return err
}

func scanTimetableItems(rows *sql.Rows) ([]TimetableItemRecord, error) {
	var items []TimetableItemRecord
	for rows.Next() {
		var item TimetableItemRecord
		var notifyDesktop, notifyEmail int
		var details sql.NullString
		var notifiedDesktopAt, notifiedEmailAt sql.NullTime

		if err := rows.Scan(&item.ID, &item.PlanID, &item.Title, &details,
			&item.StartTime, &item.EndTime, &item.NotifyAt,
			&notifyDesktop, &notifyEmail, &notifiedDesktopAt, &notifiedEmailAt, &item.CreatedAt); err != nil {
			return nil, err
		}

		item.Details = details.String
		item.NotifyDesktop = notifyDesktop == 1
		item.NotifyEmail = notifyEmail == 1
		if notifiedDesktopAt.Valid {
			item.NotifiedDesktopAt = &notifiedDesktopAt.Time
		}
		if notifiedEmailAt.Valid {
			item.NotifiedEmailAt = &notifiedEmailAt.Time
		}
		items = append(items, item)
	}

	return items, nil
}

func scanTimetableEvents(rows *sql.Rows) ([]TimetableEventRecord, error) {
	var events []TimetableEventRecord
	for rows.Next() {
		var event TimetableEventRecord
		var notifyDesktop, notifyEmail, recurring int
		var planGoal, planEmailTo, details sql.NullString
		var recurrenceDays, weekdayMask sql.NullInt64
		var notifiedDesktopAt, notifiedEmailAt sql.NullTime

		if err := rows.Scan(&event.ID, &event.PlanID, &event.PlanName, &planGoal,
			&event.PlanTimezone, &planEmailTo, &recurring, &recurrenceDays, &weekdayMask,
			&event.Title, &details,
			&event.StartTime, &event.EndTime, &event.NotifyAt,
			&notifyDesktop, &notifyEmail, &notifiedDesktopAt, &notifiedEmailAt); err != nil {
			return nil, err
		}

		event.PlanGoal = planGoal.String
		event.PlanEmailTo = planEmailTo.String
		event.PlanRecurring = recurring == 1
		if recurrenceDays.Valid {
			event.PlanRecurrenceDays = int(recurrenceDays.Int64)
		}
		if event.PlanRecurrenceDays <= 0 {
			event.PlanRecurrenceDays = 1
		}
		if weekdayMask.Valid {
			event.PlanWeekdayMask = int(weekdayMask.Int64)
		}
		if event.PlanWeekdayMask <= 0 {
			event.PlanWeekdayMask = WeekdayMaskAll
		}
		event.Details = details.String
		event.NotifyDesktop = notifyDesktop == 1
		event.NotifyEmail = notifyEmail == 1
		if notifiedDesktopAt.Valid {
			event.NotifiedDesktopAt = &notifiedDesktopAt.Time
		}
		if notifiedEmailAt.Valid {
			event.NotifiedEmailAt = &notifiedEmailAt.Time
		}

		events = append(events, event)
	}

	return events, nil
}
