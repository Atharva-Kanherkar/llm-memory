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

// Built-in OAuth credentials for easy setup.
// These are for desktop/native apps where client secrets cannot be truly confidential.
// Users can override with their own credentials via environment variables.
// To use your own: set GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET, SLACK_CLIENT_ID, SLACK_CLIENT_SECRET
var (
	// Default Google OAuth - users should create their own at console.cloud.google.com
	defaultGoogleClientID     = "" // Set via GOOGLE_CLIENT_ID
	defaultGoogleClientSecret = "" // Set via GOOGLE_CLIENT_SECRET

	// Default Slack OAuth - users should create their own at api.slack.com
	defaultSlackClientID     = "" // Set via SLACK_CLIENT_ID
	defaultSlackClientSecret = "" // Set via SLACK_CLIENT_SECRET
)

// getGoogleCredentials returns Google OAuth credentials.
func getGoogleCredentials() (clientID, clientSecret string) {
	clientID = os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret = os.Getenv("GOOGLE_CLIENT_SECRET")

	if clientID == "" {
		clientID = defaultGoogleClientID
	}
	if clientSecret == "" {
		clientSecret = defaultGoogleClientSecret
	}
	return
}

// getSlackCredentials returns Slack OAuth credentials.
func getSlackCredentials() (clientID, clientSecret string) {
	clientID = os.Getenv("SLACK_CLIENT_ID")
	clientSecret = os.Getenv("SLACK_CLIENT_SECRET")

	if clientID == "" {
		clientID = defaultSlackClientID
	}
	if clientSecret == "" {
		clientSecret = defaultSlackClientSecret
	}
	return
}

// NewGmailProvider creates a Gmail OAuth provider.
func NewGmailProvider() *Provider {
	clientID, clientSecret := getGoogleCredentials()
	return &Provider{
		Name:         ProviderGmail,
		AuthURL:      "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:     "https://oauth2.googleapis.com/token",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes: []string{
			"https://www.googleapis.com/auth/gmail.readonly",
		},
		RedirectURI: "http://localhost:8085/callback",
	}
}

// NewCalendarProvider creates a Google Calendar OAuth provider.
func NewCalendarProvider() *Provider {
	clientID, clientSecret := getGoogleCredentials()
	return &Provider{
		Name:         ProviderCalendar,
		AuthURL:      "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:     "https://oauth2.googleapis.com/token",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes: []string{
			"https://www.googleapis.com/auth/calendar.readonly",
		},
		RedirectURI: "http://localhost:8086/callback",
	}
}

// NewSlackProvider creates a Slack OAuth provider.
func NewSlackProvider() *Provider {
	clientID, clientSecret := getSlackCredentials()
	return &Provider{
		Name:         ProviderSlack,
		AuthURL:      "https://slack.com/oauth/v2/authorize",
		TokenURL:     "https://slack.com/api/oauth.v2.access",
		ClientID:     clientID,
		ClientSecret: clientSecret,
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
