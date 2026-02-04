// Package config handles configuration loading and defaults.
//
// Go convention: packages in internal/ are private to this module.
// They can't be imported by other modules - only by code in this repo.
package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for the daemon.
// The struct tags (e.g., `yaml:"capture_interval_seconds"`) tell the
// YAML parser which field maps to which YAML key.
type Config struct {
	// Capture settings
	CaptureIntervalSeconds int `yaml:"capture_interval_seconds"`
	ScreenCaptureEnabled   bool `yaml:"screen_capture_enabled"`
	WindowCaptureEnabled   bool `yaml:"window_capture_enabled"`
	GitCaptureEnabled      bool `yaml:"git_capture_enabled"`
	ClipboardCaptureEnabled bool `yaml:"clipboard_capture_enabled"`

	// Storage settings
	StoragePath string `yaml:"storage_path"`

	// Privacy settings
	BlockedApps    []string `yaml:"blocked_apps"`
	BlockedURLs    []string `yaml:"blocked_urls"`

	// LLM settings - using OpenRouter to access models
	// OpenRouter is a unified API that lets you use OpenAI, Anthropic, etc.
	// through one endpoint: https://openrouter.ai/api/v1
	LLM LLMConfig `yaml:"llm"`
}

// LLMConfig holds settings for the LLM provider.
type LLMConfig struct {
	// Provider can be "openrouter", "ollama", "openai", "anthropic"
	// We're using openrouter to access OpenAI models
	Provider string `yaml:"provider"`

	// OpenRouter settings
	OpenRouterKey  string `yaml:"openrouter_api_key"`
	OpenRouterBase string `yaml:"openrouter_base_url"`

	// Model selection (OpenRouter model names)
	// For OpenAI via OpenRouter: "openai/gpt-4o-mini"
	// For embeddings: "openai/text-embedding-3-small"
	ChatModel      string `yaml:"chat_model"`
	EmbeddingModel string `yaml:"embedding_model"`
}

// DefaultConfig returns sensible defaults.
// In Go, functions that create new instances often start with "New" or "Default".
func DefaultConfig() *Config {
	// Get home directory for default storage path
	home, err := os.UserHomeDir()
	if err != nil {
		home = "/tmp" // Fallback if we can't get home dir
	}

	return &Config{
		CaptureIntervalSeconds:  10, // Every 10 seconds for window
		ScreenCaptureEnabled:    true,
		WindowCaptureEnabled:    true,
		GitCaptureEnabled:       true,
		ClipboardCaptureEnabled: true,

		StoragePath: filepath.Join(home, ".local", "share", "mnemosyne"),

		BlockedApps: []string{
			"1password",
			"keepassxc",
			"bitwarden",
		},
		BlockedURLs: []string{
			"*bank*",
			"*banking*",
		},

		LLM: LLMConfig{
			Provider:       "openrouter",
			OpenRouterKey:  os.Getenv("OPENROUTER_API_KEY"), // Read from environment
			OpenRouterBase: "https://openrouter.ai/api/v1",
			ChatModel:      "openai/gpt-4o-mini",            // OpenAI model via OpenRouter
			EmbeddingModel: "openai/text-embedding-3-small", // For vector embeddings
		},
	}
}

// Load loads configuration from the default path, falling back to defaults.
// It returns a pointer to Config - in Go, we often return pointers to avoid
// copying large structs.
func Load() (*Config, error) {
	cfg := DefaultConfig()

	// Try to load from config file
	// First check ~/.config/mnemosyne/config.yaml (XDG standard)
	home, err := os.UserHomeDir()
	if err != nil {
		return cfg, nil // Just use defaults if we can't get home
	}

	configPaths := []string{
		filepath.Join(home, ".config", "mnemosyne", "config.yaml"),
		filepath.Join(home, ".local", "share", "mnemosyne", "config.yaml"),
	}

	for _, path := range configPaths {
		if err := loadFromFile(cfg, path); err == nil {
			// Successfully loaded
			return cfg, nil
		}
		// If file doesn't exist, that's fine - continue to next path
	}

	// No config file found - use defaults
	return cfg, nil
}

// loadFromFile reads a YAML config file and merges it into cfg.
func loadFromFile(cfg *Config, path string) error {
	// os.ReadFile reads the entire file into memory
	// This is fine for small config files
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// yaml.Unmarshal parses YAML bytes into a Go struct
	// It uses the struct tags to match field names
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return err
	}

	return nil
}

// EnsureStorageDir creates the storage directory if it doesn't exist.
func (c *Config) EnsureStorageDir() error {
	// os.MkdirAll creates the directory and all parent directories
	// 0755 is the permission mode (rwxr-xr-x)
	return os.MkdirAll(c.StoragePath, 0755)
}
