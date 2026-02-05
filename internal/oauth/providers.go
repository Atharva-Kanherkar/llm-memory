package oauth

import (
	"os"
)

// Provider names
const (
	ProviderGmail    = "gmail"
	ProviderSlack    = "slack"
	ProviderCalendar = "google_calendar"
)

// NewGmailProvider creates a Gmail OAuth provider.
// SECURITY: Client secret loaded from environment variable.
func NewGmailProvider() *Provider {
	return &Provider{
		Name:         ProviderGmail,
		AuthURL:      "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:     "https://oauth2.googleapis.com/token",
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		Scopes: []string{
			"https://www.googleapis.com/auth/gmail.readonly",
		},
		RedirectURI: "http://localhost:8085/callback",
	}
}

// NewCalendarProvider creates a Google Calendar OAuth provider.
// SECURITY: Client secret loaded from environment variable.
func NewCalendarProvider() *Provider {
	return &Provider{
		Name:         ProviderCalendar,
		AuthURL:      "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:     "https://oauth2.googleapis.com/token",
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		Scopes: []string{
			"https://www.googleapis.com/auth/calendar.readonly",
		},
		RedirectURI: "http://localhost:8086/callback",
	}
}

// NewSlackProvider creates a Slack OAuth provider.
// SECURITY: Client secret loaded from environment variable.
func NewSlackProvider() *Provider {
	return &Provider{
		Name:         ProviderSlack,
		AuthURL:      "https://slack.com/oauth/v2/authorize",
		TokenURL:     "https://slack.com/api/oauth.v2.access",
		ClientID:     os.Getenv("SLACK_CLIENT_ID"),
		ClientSecret: os.Getenv("SLACK_CLIENT_SECRET"),
		Scopes: []string{
			"channels:history",
			"channels:read",
			"users:read",
		},
		RedirectURI: "http://localhost:8087/callback",
	}
}

// IsProviderConfigured checks if a provider has the required environment variables.
func IsProviderConfigured(provider string) bool {
	switch provider {
	case ProviderGmail, ProviderCalendar:
		return os.Getenv("GOOGLE_CLIENT_ID") != "" && os.Getenv("GOOGLE_CLIENT_SECRET") != ""
	case ProviderSlack:
		return os.Getenv("SLACK_CLIENT_ID") != "" && os.Getenv("SLACK_CLIENT_SECRET") != ""
	default:
		return false
	}
}

// GetProviderStatus returns whether each provider is configured and authenticated.
func GetProviderStatus(store *TokenStore) map[string]map[string]bool {
	status := make(map[string]map[string]bool)

	providers := []string{ProviderGmail, ProviderSlack, ProviderCalendar}
	for _, p := range providers {
		status[p] = map[string]bool{
			"configured":    IsProviderConfigured(p),
			"authenticated": store.HasToken(p),
		}
	}

	return status
}
