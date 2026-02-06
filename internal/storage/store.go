// Package storage handles persistence of captures.
//
// Architecture:
// - SQLite database for metadata (searchable, indexed)
// - File system for raw data (screenshots, audio)
//
// Directory structure:
// ~/.local/share/mnemosyne/
// ├── mnemosyne.db              # SQLite database
// ├── captures/
// │   ├── 2025/
// │   │   ├── 02/
// │   │   │   ├── 05/
// │   │   │   │   ├── screen_143022_abc123.png
// │   │   │   │   ├── audio_143100_def456.wav
package storage

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Atharva-Kanherkar/mnemosyne/internal/capture"
	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// Store handles persistence of captures.
type Store struct {
	db      *sql.DB
	dataDir string
}

// New creates a new Store.
func New(baseDir string) (*Store, error) {
	// Ensure directories exist
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	capturesDir := filepath.Join(baseDir, "captures")
	if err := os.MkdirAll(capturesDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create captures directory: %w", err)
	}

	// Open SQLite database
	dbPath := filepath.Join(baseDir, "mnemosyne.db")
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &Store{
		db:      db,
		dataDir: capturesDir,
	}

	// Initialize schema
	if err := store.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return store, nil
}

// initSchema creates the database tables.
func (s *Store) initSchema() error {
	// First create base schema
	if err := s.createBaseSchema(); err != nil {
		return err
	}
	// Then run migrations for schema updates
	return s.migrateSchema()
}

// createBaseSchema creates the initial database tables.
func (s *Store) createBaseSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS captures (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		source TEXT NOT NULL,
		timestamp DATETIME NOT NULL,
		text_data TEXT,
		raw_data_path TEXT,
		metadata JSON,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_captures_timestamp ON captures(timestamp);
	CREATE INDEX IF NOT EXISTS idx_captures_source ON captures(source);

	CREATE TABLE IF NOT EXISTS sessions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		start_time DATETIME NOT NULL,
		end_time DATETIME,
		summary TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_sessions_start ON sessions(start_time);

	CREATE TABLE IF NOT EXISTS insights (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		insight_type TEXT NOT NULL,
		severity TEXT NOT NULL,
		title TEXT NOT NULL,
		body TEXT NOT NULL,
		trigger_source TEXT,
		related_captures TEXT,
		metadata JSON,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		time_range_start DATETIME,
		time_range_end DATETIME,
		acknowledged_at DATETIME,
		notified_desktop INTEGER DEFAULT 0,
		notified_tui INTEGER DEFAULT 0
	);

	CREATE INDEX IF NOT EXISTS idx_insights_type ON insights(insight_type);
	CREATE INDEX IF NOT EXISTS idx_insights_created ON insights(created_at);
	CREATE INDEX IF NOT EXISTS idx_insights_severity ON insights(severity);

	CREATE TABLE IF NOT EXISTS focus_modes (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		purpose TEXT,
		allowed_apps TEXT,
		blocked_apps TEXT,
		blocked_patterns TEXT,
		allowed_sites TEXT,
		browser_policy TEXT DEFAULT 'ask_llm',
		duration_minutes INT DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS focus_sessions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		mode_id TEXT,
		started_at DATETIME NOT NULL,
		ended_at DATETIME,
		blocks_count INT DEFAULT 0,
		heartbeat DATETIME DEFAULT CURRENT_TIMESTAMP,
		quit_reason TEXT,
		planned_duration_minutes INT DEFAULT 0,
		actual_duration_minutes INT DEFAULT 0,
		FOREIGN KEY (mode_id) REFERENCES focus_modes(id)
	);

	CREATE INDEX IF NOT EXISTS idx_focus_sessions_mode ON focus_sessions(mode_id);
	CREATE INDEX IF NOT EXISTS idx_focus_sessions_started ON focus_sessions(started_at);

	CREATE TABLE IF NOT EXISTS focus_session_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id INTEGER NOT NULL,
		event_type TEXT NOT NULL,
		app_class TEXT,
		window_title TEXT,
		llm_decision TEXT,
		reason TEXT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (session_id) REFERENCES focus_sessions(id)
	);

	CREATE INDEX IF NOT EXISTS idx_focus_events_session ON focus_session_events(session_id);
	CREATE INDEX IF NOT EXISTS idx_focus_events_timestamp ON focus_session_events(timestamp);

	CREATE TABLE IF NOT EXISTS summaries (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		summary_type TEXT NOT NULL,
		start_time DATETIME NOT NULL,
		end_time DATETIME NOT NULL,
		content TEXT NOT NULL,
		apps TEXT,
		tokens INT DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_summaries_type ON summaries(summary_type);
	CREATE INDEX IF NOT EXISTS idx_summaries_start ON summaries(start_time);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_summaries_unique ON summaries(summary_type, start_time);
	`

	_, err := s.db.Exec(schema)
	return err
}

// migrateSchema handles schema migrations for existing databases.
func (s *Store) migrateSchema() error {
	// Migration: Add focus session tracking columns (v2)
	migrations := []string{
		// Add new columns to focus_sessions if they don't exist
		`ALTER TABLE focus_sessions ADD COLUMN quit_reason TEXT`,
		`ALTER TABLE focus_sessions ADD COLUMN planned_duration_minutes INTEGER DEFAULT 0`,
		`ALTER TABLE focus_sessions ADD COLUMN actual_duration_minutes INTEGER DEFAULT 0`,
	}

	for _, migration := range migrations {
		// Try to execute migration - will fail if column already exists, which is fine
		_, _ = s.db.Exec(migration)
	}

	return nil
}

// Save persists a capture result.
func (s *Store) Save(result *capture.Result) (int64, error) {
	var rawDataPath string

	// If there's raw data, save it to a file
	if len(result.RawData) > 0 {
		path, err := s.saveRawData(result)
		if err != nil {
			return 0, fmt.Errorf("failed to save raw data: %w", err)
		}
		rawDataPath = path
	}

	// Serialize metadata to JSON
	metadataJSON, err := json.Marshal(result.Metadata)
	if err != nil {
		return 0, fmt.Errorf("failed to serialize metadata: %w", err)
	}

	// Insert into database
	res, err := s.db.Exec(`
		INSERT INTO captures (source, timestamp, text_data, raw_data_path, metadata)
		VALUES (?, ?, ?, ?, ?)
	`, result.Source, result.Timestamp, result.TextData, rawDataPath, string(metadataJSON))

	if err != nil {
		return 0, fmt.Errorf("failed to insert capture: %w", err)
	}

	return res.LastInsertId()
}

// saveRawData saves binary data to a file and returns the path.
func (s *Store) saveRawData(result *capture.Result) (string, error) {
	// Generate path: captures/YYYY/MM/DD/source_HHMMSS_random.ext
	t := result.Timestamp
	dir := filepath.Join(s.dataDir,
		fmt.Sprintf("%04d", t.Year()),
		fmt.Sprintf("%02d", t.Month()),
		fmt.Sprintf("%02d", t.Day()))

	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	// Determine file extension
	ext := ".bin"
	if format, ok := result.Metadata["format"]; ok {
		switch format {
		case "png":
			ext = ".png"
		case "jpg", "jpeg":
			ext = ".jpg"
		case "wav":
			ext = ".wav"
		case "raw":
			ext = ".raw"
		}
	}

	// Generate filename
	randomBytes := make([]byte, 4)
	rand.Read(randomBytes)
	randomHex := hex.EncodeToString(randomBytes)

	filename := fmt.Sprintf("%s_%s_%s%s",
		result.Source,
		t.Format("150405"), // HHMMSS
		randomHex,
		ext)

	path := filepath.Join(dir, filename)

	// Write file
	if err := os.WriteFile(path, result.RawData, 0644); err != nil {
		return "", err
	}

	return path, nil
}

// GetRecent retrieves the most recent captures.
func (s *Store) GetRecent(limit int) ([]CaptureRecord, error) {
	rows, err := s.db.Query(`
		SELECT id, source, timestamp, text_data, raw_data_path, metadata
		FROM captures
		ORDER BY timestamp DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []CaptureRecord
	for rows.Next() {
		var r CaptureRecord
		var metadataJSON string
		var rawDataPath sql.NullString
		var textData sql.NullString

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

// GetBySource retrieves captures from a specific source.
func (s *Store) GetBySource(source string, limit int) ([]CaptureRecord, error) {
	rows, err := s.db.Query(`
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

// GetByTimeRange retrieves captures within a time range.
func (s *Store) GetByTimeRange(start, end time.Time) ([]CaptureRecord, error) {
	rows, err := s.db.Query(`
		SELECT id, source, timestamp, text_data, raw_data_path, metadata
		FROM captures
		WHERE timestamp BETWEEN ? AND ?
		ORDER BY timestamp DESC
	`, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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

// CaptureRecord represents a capture stored in the database.
type CaptureRecord struct {
	ID          int64
	Source      string
	Timestamp   time.Time
	TextData    string
	RawDataPath string
	Metadata    map[string]string
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// Stats returns statistics about stored captures.
func (s *Store) Stats() (Stats, error) {
	var stats Stats

	// Total captures
	row := s.db.QueryRow("SELECT COUNT(*) FROM captures")
	row.Scan(&stats.TotalCaptures)

	// Captures by source
	rows, err := s.db.Query("SELECT source, COUNT(*) FROM captures GROUP BY source")
	if err == nil {
		stats.BySource = make(map[string]int64)
		for rows.Next() {
			var source string
			var count int64
			rows.Scan(&source, &count)
			stats.BySource[source] = count
		}
		rows.Close()
	}

	// Database size
	if info, err := os.Stat(filepath.Join(filepath.Dir(s.dataDir), "mnemosyne.db")); err == nil {
		stats.DatabaseSize = info.Size()
	}

	// Data directory size (approximate - just count files)
	filepath.Walk(s.dataDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			stats.DataSize += info.Size()
		}
		return nil
	})

	return stats, nil
}

// Stats holds storage statistics.
type Stats struct {
	TotalCaptures int64
	BySource      map[string]int64
	DatabaseSize  int64
	DataSize      int64
}

// InsightRecord represents an insight stored in the database.
type InsightRecord struct {
	ID              int64
	Type            string
	Severity        string
	Title           string
	Body            string
	TriggerSource   string
	RelatedCaptures []int64
	Metadata        map[string]any
	CreatedAt       time.Time
	TimeRangeStart  *time.Time
	TimeRangeEnd    *time.Time
	AcknowledgedAt  *time.Time
	NotifiedDesktop bool
	NotifiedTUI     bool
}

// SaveInsight stores an insight in the database.
func (s *Store) SaveInsight(insight *InsightRecord) (int64, error) {
	var metadataJSON, relatedJSON []byte
	var err error

	if insight.Metadata != nil {
		metadataJSON, err = json.Marshal(insight.Metadata)
		if err != nil {
			return 0, fmt.Errorf("failed to serialize metadata: %w", err)
		}
	}

	if len(insight.RelatedCaptures) > 0 {
		relatedJSON, err = json.Marshal(insight.RelatedCaptures)
		if err != nil {
			return 0, fmt.Errorf("failed to serialize related captures: %w", err)
		}
	}

	res, err := s.db.Exec(`
		INSERT INTO insights
		(insight_type, severity, title, body, trigger_source, related_captures, metadata, time_range_start, time_range_end)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, insight.Type, insight.Severity, insight.Title, insight.Body,
		insight.TriggerSource, string(relatedJSON), string(metadataJSON),
		insight.TimeRangeStart, insight.TimeRangeEnd)

	if err != nil {
		return 0, fmt.Errorf("failed to insert insight: %w", err)
	}

	return res.LastInsertId()
}

// GetRecentInsights retrieves recent unacknowledged insights.
func (s *Store) GetRecentInsights(limit int) ([]InsightRecord, error) {
	rows, err := s.db.Query(`
		SELECT id, insight_type, severity, title, body, trigger_source,
		       related_captures, metadata, created_at, time_range_start,
		       time_range_end, acknowledged_at, notified_desktop, notified_tui
		FROM insights
		WHERE acknowledged_at IS NULL
		ORDER BY created_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanInsights(rows)
}

// GetInsightsByTimeRange retrieves insights within a time range.
func (s *Store) GetInsightsByTimeRange(start, end time.Time) ([]InsightRecord, error) {
	rows, err := s.db.Query(`
		SELECT id, insight_type, severity, title, body, trigger_source,
		       related_captures, metadata, created_at, time_range_start,
		       time_range_end, acknowledged_at, notified_desktop, notified_tui
		FROM insights
		WHERE created_at BETWEEN ? AND ?
		ORDER BY created_at DESC
	`, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanInsights(rows)
}

// AcknowledgeInsight marks an insight as acknowledged.
func (s *Store) AcknowledgeInsight(id int64) error {
	_, err := s.db.Exec(`
		UPDATE insights SET acknowledged_at = CURRENT_TIMESTAMP WHERE id = ?
	`, id)
	return err
}

// MarkInsightNotified updates notification status.
func (s *Store) MarkInsightNotified(id int64, desktop, tui bool) error {
	_, err := s.db.Exec(`
		UPDATE insights SET notified_desktop = ?, notified_tui = ? WHERE id = ?
	`, desktop, tui, id)
	return err
}

func (s *Store) scanInsights(rows *sql.Rows) ([]InsightRecord, error) {
	var records []InsightRecord
	for rows.Next() {
		var r InsightRecord
		var relatedJSON, metadataJSON sql.NullString
		var triggerSource sql.NullString
		var timeRangeStart, timeRangeEnd, acknowledgedAt sql.NullTime

		err := rows.Scan(&r.ID, &r.Type, &r.Severity, &r.Title, &r.Body,
			&triggerSource, &relatedJSON, &metadataJSON, &r.CreatedAt,
			&timeRangeStart, &timeRangeEnd, &acknowledgedAt,
			&r.NotifiedDesktop, &r.NotifiedTUI)
		if err != nil {
			return nil, err
		}

		r.TriggerSource = triggerSource.String
		if timeRangeStart.Valid {
			r.TimeRangeStart = &timeRangeStart.Time
		}
		if timeRangeEnd.Valid {
			r.TimeRangeEnd = &timeRangeEnd.Time
		}
		if acknowledgedAt.Valid {
			r.AcknowledgedAt = &acknowledgedAt.Time
		}

		if relatedJSON.Valid && relatedJSON.String != "" {
			json.Unmarshal([]byte(relatedJSON.String), &r.RelatedCaptures)
		}
		if metadataJSON.Valid && metadataJSON.String != "" {
			json.Unmarshal([]byte(metadataJSON.String), &r.Metadata)
		}

		records = append(records, r)
	}

	return records, nil
}

// SearchText searches captures by text content.
func (s *Store) SearchText(searchText string, limit int) ([]CaptureRecord, error) {
	rows, err := s.db.Query(`
		SELECT id, source, timestamp, text_data, raw_data_path, metadata
		FROM captures
		WHERE text_data LIKE ?
		ORDER BY timestamp DESC
		LIMIT ?
	`, "%"+searchText+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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

// FocusModeRecord represents a focus mode in the database.
type FocusModeRecord struct {
	ID              string
	Name            string
	Purpose         string
	AllowedApps     string // JSON array
	BlockedApps     string // JSON array
	BlockedPatterns string // JSON array
	AllowedSites    string // JSON array
	BrowserPolicy   string
	DurationMinutes int
	CreatedAt       time.Time
}

// FocusSessionRecord represents a focus session in the database.
type FocusSessionRecord struct {
	ID              int64
	ModeID          string
	StartedAt       time.Time
	EndedAt         *time.Time
	BlocksCount     int
	Heartbeat       time.Time // Last heartbeat from TUI
	PlannedDuration int       // Planned duration in minutes (0 = no limit)
}

// SaveFocusMode saves a focus mode to the database.
func (s *Store) SaveFocusMode(mode *FocusModeRecord) error {
	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO focus_modes
		(id, name, purpose, allowed_apps, blocked_apps, blocked_patterns,
		 allowed_sites, browser_policy, duration_minutes, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, mode.ID, mode.Name, mode.Purpose, mode.AllowedApps, mode.BlockedApps,
		mode.BlockedPatterns, mode.AllowedSites, mode.BrowserPolicy,
		mode.DurationMinutes, mode.CreatedAt)
	return err
}

// GetFocusMode retrieves a focus mode by ID.
func (s *Store) GetFocusMode(id string) (*FocusModeRecord, error) {
	row := s.db.QueryRow(`
		SELECT id, name, purpose, allowed_apps, blocked_apps, blocked_patterns,
		       allowed_sites, browser_policy, duration_minutes, created_at
		FROM focus_modes WHERE id = ?
	`, id)

	var m FocusModeRecord
	var allowedApps, blockedApps, blockedPatterns, allowedSites sql.NullString
	var purpose sql.NullString

	err := row.Scan(&m.ID, &m.Name, &purpose, &allowedApps, &blockedApps,
		&blockedPatterns, &allowedSites, &m.BrowserPolicy, &m.DurationMinutes, &m.CreatedAt)
	if err != nil {
		return nil, err
	}

	m.Purpose = purpose.String
	m.AllowedApps = allowedApps.String
	m.BlockedApps = blockedApps.String
	m.BlockedPatterns = blockedPatterns.String
	m.AllowedSites = allowedSites.String

	return &m, nil
}

// ListFocusModes retrieves all focus modes.
func (s *Store) ListFocusModes() ([]FocusModeRecord, error) {
	rows, err := s.db.Query(`
		SELECT id, name, purpose, allowed_apps, blocked_apps, blocked_patterns,
		       allowed_sites, browser_policy, duration_minutes, created_at
		FROM focus_modes ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var modes []FocusModeRecord
	for rows.Next() {
		var m FocusModeRecord
		var allowedApps, blockedApps, blockedPatterns, allowedSites sql.NullString
		var purpose sql.NullString

		err := rows.Scan(&m.ID, &m.Name, &purpose, &allowedApps, &blockedApps,
			&blockedPatterns, &allowedSites, &m.BrowserPolicy, &m.DurationMinutes, &m.CreatedAt)
		if err != nil {
			return nil, err
		}

		m.Purpose = purpose.String
		m.AllowedApps = allowedApps.String
		m.BlockedApps = blockedApps.String
		m.BlockedPatterns = blockedPatterns.String
		m.AllowedSites = allowedSites.String

		modes = append(modes, m)
	}

	return modes, nil
}

// DeleteFocusMode deletes a focus mode by ID.
func (s *Store) DeleteFocusMode(id string) error {
	_, err := s.db.Exec("DELETE FROM focus_modes WHERE id = ?", id)
	return err
}

// StartFocusSession starts a new focus session.
func (s *Store) StartFocusSession(modeID string, plannedDurationMinutes int) (int64, error) {
	res, err := s.db.Exec(`
		INSERT INTO focus_sessions (mode_id, started_at, heartbeat, planned_duration_minutes)
		VALUES (?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, ?)
	`, modeID, plannedDurationMinutes)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateFocusSessionHeartbeat updates the heartbeat timestamp for a session.
func (s *Store) UpdateFocusSessionHeartbeat(sessionID int64) error {
	_, err := s.db.Exec(`
		UPDATE focus_sessions SET heartbeat = CURRENT_TIMESTAMP WHERE id = ?
	`, sessionID)
	return err
}

// EndFocusSession ends a focus session.
func (s *Store) EndFocusSession(sessionID int64) error {
	_, err := s.db.Exec(`
		UPDATE focus_sessions SET ended_at = CURRENT_TIMESTAMP WHERE id = ?
	`, sessionID)
	return err
}

// EndAllActiveFocusSessions ends all active focus sessions (cleanup orphans).
func (s *Store) EndAllActiveFocusSessions() (int64, error) {
	result, err := s.db.Exec(`
		UPDATE focus_sessions SET ended_at = CURRENT_TIMESTAMP WHERE ended_at IS NULL
	`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// EndFocusSessionWithReason ends a session and records why it ended.
func (s *Store) EndFocusSessionWithReason(sessionID int64, quitReason string, plannedDuration, actualDuration int) error {
	_, err := s.db.Exec(`
		UPDATE focus_sessions 
		SET ended_at = CURRENT_TIMESTAMP,
		    quit_reason = ?,
		    planned_duration_minutes = ?,
		    actual_duration_minutes = ?
		WHERE id = ?
	`, quitReason, plannedDuration, actualDuration, sessionID)
	return err
}

// EndFocusSessionExpired ends a session that ran out of time.
func (s *Store) EndFocusSessionExpired(sessionID int64) error {
	_, err := s.db.Exec(`
		UPDATE focus_sessions 
		SET ended_at = CURRENT_TIMESTAMP,
		    quit_reason = 'time_up',
		    actual_duration_minutes = CAST((julianday(CURRENT_TIMESTAMP) - julianday(started_at)) * 24 * 60 AS INTEGER)
		WHERE id = ?
	`, sessionID)
	return err
}

// IncrementFocusSessionBlocks increments the block count for a session.
func (s *Store) IncrementFocusSessionBlocks(sessionID int64) error {
	_, err := s.db.Exec(`
		UPDATE focus_sessions SET blocks_count = blocks_count + 1 WHERE id = ?
	`, sessionID)
	return err
}

// FocusSessionEvent represents a behavioral event during a focus session.
type FocusSessionEvent struct {
	ID          int64
	SessionID   int64
	EventType   string // 'block', 'warn', 'allow', 'switch', 'llm_check'
	AppClass    string
	WindowTitle string
	LLMDecision string // 'ALLOW', 'BLOCK', ''
	Reason      string
	Timestamp   time.Time
}

// SaveFocusSessionEvent logs an event during a focus session.
func (s *Store) SaveFocusSessionEvent(event *FocusSessionEvent) (int64, error) {
	res, err := s.db.Exec(`
		INSERT INTO focus_session_events (session_id, event_type, app_class, window_title, llm_decision, reason, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, event.SessionID, event.EventType, event.AppClass, event.WindowTitle, event.LLMDecision, event.Reason, event.Timestamp)
	if err != nil {
		return 0, fmt.Errorf("failed to save focus event: %w", err)
	}
	return res.LastInsertId()
}

// GetFocusSessionEvents retrieves all events for a session.
func (s *Store) GetFocusSessionEvents(sessionID int64) ([]FocusSessionEvent, error) {
	rows, err := s.db.Query(`
		SELECT id, session_id, event_type, app_class, window_title, llm_decision, reason, timestamp
		FROM focus_session_events
		WHERE session_id = ?
		ORDER BY timestamp ASC
	`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []FocusSessionEvent
	for rows.Next() {
		var e FocusSessionEvent
		var llmDecision, reason sql.NullString
		err := rows.Scan(&e.ID, &e.SessionID, &e.EventType, &e.AppClass, &e.WindowTitle, &llmDecision, &reason, &e.Timestamp)
		if err != nil {
			return nil, err
		}
		e.LLMDecision = llmDecision.String
		e.Reason = reason.String
		events = append(events, e)
	}
	return events, nil
}

// GetActiveFocusSession returns the currently active session if any.
// Only returns sessions with a heartbeat within the last 2 minutes.
func (s *Store) GetActiveFocusSession() (*FocusSessionRecord, error) {
	row := s.db.QueryRow(`
		SELECT id, mode_id, started_at, ended_at, blocks_count, planned_duration_minutes
		FROM focus_sessions
		WHERE ended_at IS NULL
		  AND (heartbeat IS NULL OR heartbeat > datetime('now', '-2 minutes'))
		ORDER BY started_at DESC LIMIT 1
	`)

	var session FocusSessionRecord
	var endedAt sql.NullTime
	var plannedDuration sql.NullInt64

	err := row.Scan(&session.ID, &session.ModeID, &session.StartedAt, &endedAt, &session.BlocksCount, &plannedDuration)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if endedAt.Valid {
		session.EndedAt = &endedAt.Time
	}
	if plannedDuration.Valid {
		session.PlannedDuration = int(plannedDuration.Int64)
	}

	return &session, nil
}

// GetFocusSessionStats returns statistics for focus sessions.
func (s *Store) GetFocusSessionStats(modeID string) (totalSessions int, totalMinutes int, totalBlocks int, err error) {
	row := s.db.QueryRow(`
		SELECT COUNT(*),
		       COALESCE(SUM(CAST((julianday(COALESCE(ended_at, CURRENT_TIMESTAMP)) - julianday(started_at)) * 24 * 60 AS INTEGER)), 0),
		       COALESCE(SUM(blocks_count), 0)
		FROM focus_sessions WHERE mode_id = ?
	`, modeID)

	err = row.Scan(&totalSessions, &totalMinutes, &totalBlocks)
	return
}

// SummaryRecord represents a memory summary in the database.
type SummaryRecord struct {
	ID        int64
	Type      string
	StartTime time.Time
	EndTime   time.Time
	Content   string
	Apps      string
	Tokens    int
	CreatedAt time.Time
}

// SaveSummary saves a memory summary to the database.
func (s *Store) SaveSummary(summary *SummaryRecord) error {
	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO summaries
		(summary_type, start_time, end_time, content, apps, tokens)
		VALUES (?, ?, ?, ?, ?, ?)
	`, summary.Type, summary.StartTime, summary.EndTime, summary.Content, summary.Apps, summary.Tokens)
	return err
}

// SummaryExists checks if a summary exists for a given type and start time.
func (s *Store) SummaryExists(summaryType string, startTime time.Time) (bool, error) {
	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM summaries
		WHERE summary_type = ? AND start_time = ?
	`, summaryType, startTime).Scan(&count)
	return count > 0, err
}

// GetSummariesByRange retrieves summaries within a time range.
func (s *Store) GetSummariesByRange(summaryType string, start, end time.Time) ([]SummaryRecord, error) {
	rows, err := s.db.Query(`
		SELECT id, summary_type, start_time, end_time, content, apps, tokens, created_at
		FROM summaries
		WHERE summary_type = ? AND start_time >= ? AND start_time < ?
		ORDER BY start_time ASC
	`, summaryType, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []SummaryRecord
	for rows.Next() {
		var sum SummaryRecord
		var apps sql.NullString
		err := rows.Scan(&sum.ID, &sum.Type, &sum.StartTime, &sum.EndTime,
			&sum.Content, &apps, &sum.Tokens, &sum.CreatedAt)
		if err != nil {
			return nil, err
		}
		sum.Apps = apps.String
		summaries = append(summaries, sum)
	}
	return summaries, nil
}

// GetRecentSummaries retrieves recent summaries of a given type.
func (s *Store) GetRecentSummaries(summaryType string, limit int) ([]SummaryRecord, error) {
	rows, err := s.db.Query(`
		SELECT id, summary_type, start_time, end_time, content, apps, tokens, created_at
		FROM summaries
		WHERE summary_type = ?
		ORDER BY start_time DESC
		LIMIT ?
	`, summaryType, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []SummaryRecord
	for rows.Next() {
		var sum SummaryRecord
		var apps sql.NullString
		err := rows.Scan(&sum.ID, &sum.Type, &sum.StartTime, &sum.EndTime,
			&sum.Content, &apps, &sum.Tokens, &sum.CreatedAt)
		if err != nil {
			return nil, err
		}
		sum.Apps = apps.String
		summaries = append(summaries, sum)
	}
	return summaries, nil
}
