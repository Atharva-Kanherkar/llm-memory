// Package config handles configuration loading and defaults.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for the daemon.
type Config struct {
	CaptureIntervalSeconds  int  `yaml:"capture_interval_seconds"`
	ScreenCaptureEnabled    bool `yaml:"screen_capture_enabled"`
	WindowCaptureEnabled    bool `yaml:"window_capture_enabled"`
	GitCaptureEnabled       bool `yaml:"git_capture_enabled"`
	ClipboardCaptureEnabled bool `yaml:"clipboard_capture_enabled"`
	OCREnabled              bool `yaml:"ocr_enabled"` // Pre-compute OCR on screenshots (uses LLM API)

	StoragePath string `yaml:"storage_path"`

	// Privacy settings
	BlockedApps     []string `yaml:"blocked_apps"`
	BlockedURLs     []string `yaml:"blocked_urls"`
	BlockedKeywords []string `yaml:"blocked_keywords"`
	Paused          bool     `yaml:"paused"`

	LLM       LLMConfig       `yaml:"llm"`
	Insights  InsightsConfig  `yaml:"insights"`
	Timetable TimetableConfig `yaml:"timetable"`
}

// InsightsConfig holds settings for proactive insights.
type InsightsConfig struct {
	Enabled              bool   `yaml:"enabled"`
	DesktopNotifications bool   `yaml:"desktop_notifications"`
	BatchIntervalMinutes int    `yaml:"batch_interval_minutes"`
	StressAlertsEnabled  bool   `yaml:"stress_alerts_enabled"`
	ContextReminders     bool   `yaml:"context_reminders"`
	LLMModel             string `yaml:"llm_model"`
}

// TimetableConfig holds settings for timetable generation and reminder delivery.
type TimetableConfig struct {
	Enabled                 bool   `yaml:"enabled"`
	ReminderCheckSeconds    int    `yaml:"reminder_check_seconds"`
	ReminderLookbackMinutes int    `yaml:"reminder_lookback_minutes"`
	DesktopNotifications    bool   `yaml:"desktop_notifications"`
	EmailNotifications      bool   `yaml:"email_notifications"`
	DefaultEmailTo          string `yaml:"default_email_to"`
	EmailFrom               string `yaml:"email_from"`
	SMTPHost                string `yaml:"smtp_host"`
	SMTPPort                int    `yaml:"smtp_port"`
	SMTPUser                string `yaml:"smtp_user"`
	SMTPPass                string `yaml:"smtp_pass"`
}

// LLMConfig holds settings for the LLM provider.
type LLMConfig struct {
	Provider       string `yaml:"provider"`
	OpenRouterKey  string `yaml:"openrouter_api_key"`
	OpenRouterBase string `yaml:"openrouter_base_url"`
	ChatModel      string `yaml:"chat_model"`
	EmbeddingModel string `yaml:"embedding_model"`
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() *Config {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "/tmp"
	}

	return &Config{
		CaptureIntervalSeconds:  10,
		ScreenCaptureEnabled:    true,
		WindowCaptureEnabled:    true,
		GitCaptureEnabled:       true,
		ClipboardCaptureEnabled: true,
		OCREnabled:              true, // Enable by default, disable to save API costs

		StoragePath: filepath.Join(home, ".local", "share", "mnemosyne"),

		// Default privacy blocklist - sensitive apps
		BlockedApps: []string{
			"1password", "keepassxc", "bitwarden", "lastpass",
			"gnome-keyring", "seahorse", "wallet",
		},
		BlockedURLs: []string{
			"*bank*", "*banking*", "*paypal*", "*venmo*",
			"*password*", "*login*", "*signin*",
		},
		BlockedKeywords: []string{
			"password", "secret", "api_key", "apikey", "token",
			"private_key", "ssh_key", "credential",
		},
		Paused: false,

		LLM: LLMConfig{
			Provider:       "openrouter",
			OpenRouterKey:  os.Getenv("OPENROUTER_API_KEY"),
			OpenRouterBase: "https://openrouter.ai/api/v1",
			ChatModel:      "openai/gpt-4o-mini",
			EmbeddingModel: "openai/text-embedding-3-small",
		},

		Insights: InsightsConfig{
			Enabled:              true,
			DesktopNotifications: true,
			BatchIntervalMinutes: 30,
			StressAlertsEnabled:  true,
			ContextReminders:     true,
			LLMModel:             "deepseek/deepseek-chat",
		},

		Timetable: TimetableConfig{
			Enabled:                 true,
			ReminderCheckSeconds:    30,
			ReminderLookbackMinutes: 360,
			DesktopNotifications:    true,
			EmailNotifications:      false,
			DefaultEmailTo:          os.Getenv("MNEMOSYNE_DEFAULT_EMAIL_TO"),
			EmailFrom:               os.Getenv("MNEMOSYNE_SMTP_FROM"),
			SMTPHost:                os.Getenv("MNEMOSYNE_SMTP_HOST"),
			SMTPPort:                envInt("MNEMOSYNE_SMTP_PORT", 587),
			SMTPUser:                os.Getenv("MNEMOSYNE_SMTP_USER"),
			SMTPPass:                os.Getenv("MNEMOSYNE_SMTP_PASS"),
		},
	}
}

// Load loads configuration from the default path, falling back to defaults.
func Load() (*Config, error) {
	cfg := DefaultConfig()

	home, err := os.UserHomeDir()
	if err != nil {
		return cfg, nil
	}

	configPaths := []string{
		filepath.Join(home, ".config", "mnemosyne", "config.yaml"),
		filepath.Join(home, ".local", "share", "mnemosyne", "config.yaml"),
	}

	for _, path := range configPaths {
		if err := loadFromFile(cfg, path); err == nil {
			return cfg, nil
		}
	}

	return cfg, nil
}

// loadFromFile reads a YAML config file and merges it into cfg.
func loadFromFile(cfg *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return err
	}
	// Expand ~ in storage path
	cfg.StoragePath = expandTilde(cfg.StoragePath)
	return nil
}

// expandTilde expands ~ to the user's home directory.
func expandTilde(path string) string {
	if len(path) == 0 || path[0] != '~' {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[1:])
}

// Save writes the current config to disk.
func (c *Config) Save() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(home, ".config", "mnemosyne")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(configDir, "config.yaml"), data, 0600)
}

// EnsureStorageDir creates the storage directory if it doesn't exist.
func (c *Config) EnsureStorageDir() error {
	return os.MkdirAll(c.StoragePath, 0700) // More restrictive permissions
}

// IsAppBlocked checks if an app should be blocked from capture.
func (c *Config) IsAppBlocked(appName string) bool {
	appLower := strings.ToLower(appName)
	for _, blocked := range c.BlockedApps {
		if strings.Contains(appLower, strings.ToLower(blocked)) {
			return true
		}
	}
	return false
}

// IsURLBlocked checks if a URL should be blocked from capture.
func (c *Config) IsURLBlocked(url string) bool {
	urlLower := strings.ToLower(url)
	for _, pattern := range c.BlockedURLs {
		pattern = strings.ToLower(strings.ReplaceAll(pattern, "*", ""))
		if strings.Contains(urlLower, pattern) {
			return true
		}
	}
	return false
}

// ContainsBlockedKeyword checks if text contains sensitive keywords.
func (c *Config) ContainsBlockedKeyword(text string) bool {
	textLower := strings.ToLower(text)
	for _, keyword := range c.BlockedKeywords {
		if strings.Contains(textLower, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

func envInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}

	var value int
	if _, err := fmt.Sscanf(raw, "%d", &value); err != nil {
		return fallback
	}
	if value <= 0 {
		return fallback
	}
	return value
}
