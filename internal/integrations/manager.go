package integrations

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/Atharva-Kanherkar/mnemosyne/internal/capture"
	"github.com/Atharva-Kanherkar/mnemosyne/internal/oauth"
)

// Manager handles all external integrations.
type Manager struct {
	tokenStore *oauth.TokenStore
	baseDir    string

	// OAuth clients
	gmailOAuth    *oauth.Client
	slackOAuth    *oauth.Client
	calendarOAuth *oauth.Client

	// API clients
	gmail    *GmailClient
	slack    *SlackClient
	calendar *CalendarClient
}

// NewManager creates a new integrations manager.
func NewManager(baseDir string) (*Manager, error) {
	// Create token store
	tokenDir := filepath.Join(baseDir, "oauth")
	if err := os.MkdirAll(tokenDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create oauth directory: %w", err)
	}

	tokenStore, err := oauth.NewTokenStore(tokenDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create token store: %w", err)
	}

	m := &Manager{
		tokenStore: tokenStore,
		baseDir:    baseDir,
	}

	// Initialize OAuth clients for configured providers
	if oauth.IsProviderConfigured(oauth.ProviderGmail) {
		m.gmailOAuth = oauth.NewClient(oauth.NewGmailProvider(), tokenStore)
		m.gmail = NewGmailClient(m.gmailOAuth)
	}

	if oauth.IsProviderConfigured(oauth.ProviderSlack) {
		m.slackOAuth = oauth.NewClient(oauth.NewSlackProvider(), tokenStore)
		m.slack = NewSlackClient(m.slackOAuth)
	}

	if oauth.IsProviderConfigured(oauth.ProviderCalendar) {
		m.calendarOAuth = oauth.NewClient(oauth.NewCalendarProvider(), tokenStore)
		m.calendar = NewCalendarClient(m.calendarOAuth)
	}

	return m, nil
}

// GetProviderStatus returns the status of all providers.
func (m *Manager) GetProviderStatus() map[string]map[string]bool {
	status := oauth.GetProviderStatus(m.tokenStore)

	// Add authentication status from OAuth clients
	if m.gmailOAuth != nil {
		status[oauth.ProviderGmail]["authenticated"] = m.gmailOAuth.IsAuthenticated()
	}
	if m.slackOAuth != nil {
		status[oauth.ProviderSlack]["authenticated"] = m.slackOAuth.IsAuthenticated()
	}
	if m.calendarOAuth != nil {
		status[oauth.ProviderCalendar]["authenticated"] = m.calendarOAuth.IsAuthenticated()
	}

	return status
}

// AuthenticateProvider starts the OAuth flow for a provider.
// Returns the auth URL that the user needs to visit.
func (m *Manager) AuthenticateProvider(ctx context.Context, provider string) (string, error) {
	var client *oauth.Client

	switch provider {
	case oauth.ProviderGmail:
		if m.gmailOAuth == nil {
			return "", fmt.Errorf("Gmail not configured - set GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET")
		}
		client = m.gmailOAuth
	case oauth.ProviderSlack:
		if m.slackOAuth == nil {
			return "", fmt.Errorf("Slack not configured - set SLACK_CLIENT_ID and SLACK_CLIENT_SECRET")
		}
		client = m.slackOAuth
	case oauth.ProviderCalendar:
		if m.calendarOAuth == nil {
			return "", fmt.Errorf("Calendar not configured - set GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET")
		}
		client = m.calendarOAuth
	default:
		return "", fmt.Errorf("unknown provider: %s", provider)
	}

	// Get auth URL and start local callback server
	authURL, state, err := client.GetAuthURL()
	if err != nil {
		return "", err
	}

	// Start local server in background to handle callback
	go func() {
		_, err := client.StartLocalAuthServer(ctx, state)
		if err != nil {
			log.Printf("[oauth] Authentication failed for %s: %v", provider, err)
		} else {
			log.Printf("[oauth] Successfully authenticated with %s", provider)
		}
	}()

	return authURL, nil
}

// LogoutProvider removes authentication for a provider.
func (m *Manager) LogoutProvider(provider string) error {
	var client *oauth.Client

	switch provider {
	case oauth.ProviderGmail:
		client = m.gmailOAuth
	case oauth.ProviderSlack:
		client = m.slackOAuth
	case oauth.ProviderCalendar:
		client = m.calendarOAuth
	default:
		return fmt.Errorf("unknown provider: %s", provider)
	}

	if client == nil {
		return fmt.Errorf("provider not configured: %s", provider)
	}

	return client.Logout()
}

// CaptureGmail captures recent emails as a capture result.
func (m *Manager) CaptureGmail(ctx context.Context) (*capture.Result, error) {
	if m.gmail == nil || !m.gmailOAuth.IsAuthenticated() {
		return nil, fmt.Errorf("Gmail not authenticated")
	}

	emails, err := m.gmail.GetRecentEmails(ctx, 10)
	if err != nil {
		return nil, err
	}

	result := capture.NewResult("gmail")

	// Build text summary of emails
	var text string
	for _, e := range emails {
		status := ""
		if e.IsUnread {
			status = " [UNREAD]"
		}
		text += fmt.Sprintf("From: %s%s\nSubject: %s\nSnippet: %s\n\n",
			e.From, status, e.Subject, e.Snippet)
	}

	result.TextData = text
	result.SetMetadata("email_count", fmt.Sprintf("%d", len(emails)))

	unread, _ := m.gmail.GetUnreadCount(ctx)
	result.SetMetadata("unread_count", fmt.Sprintf("%d", unread))

	return result, nil
}

// CaptureSlack captures recent Slack messages as a capture result.
func (m *Manager) CaptureSlack(ctx context.Context) (*capture.Result, error) {
	if m.slack == nil || !m.slackOAuth.IsAuthenticated() {
		return nil, fmt.Errorf("Slack not authenticated")
	}

	messages, err := m.slack.GetAllRecentMessages(ctx, 5)
	if err != nil {
		return nil, err
	}

	result := capture.NewResult("slack")

	// Build text summary of messages
	var text string
	for _, msg := range messages {
		text += fmt.Sprintf("#%s - %s: %s\n",
			msg.ChannelName, msg.User, msg.Text)
	}

	result.TextData = text
	result.SetMetadata("message_count", fmt.Sprintf("%d", len(messages)))

	return result, nil
}

// CaptureCalendar captures today's events as a capture result.
func (m *Manager) CaptureCalendar(ctx context.Context) (*capture.Result, error) {
	if m.calendar == nil || !m.calendarOAuth.IsAuthenticated() {
		return nil, fmt.Errorf("Calendar not authenticated")
	}

	events, err := m.calendar.GetTodaysEvents(ctx)
	if err != nil {
		return nil, err
	}

	result := capture.NewResult("calendar")

	// Build text summary of events
	var text string
	for _, e := range events {
		timeStr := e.Start.Format("15:04")
		if e.AllDay {
			timeStr = "All day"
		}
		text += fmt.Sprintf("%s: %s", timeStr, e.Summary)
		if e.Location != "" {
			text += fmt.Sprintf(" @ %s", e.Location)
		}
		if e.MeetLink != "" {
			text += " [has meeting link]"
		}
		text += "\n"
	}

	result.TextData = text
	result.SetMetadata("event_count", fmt.Sprintf("%d", len(events)))

	// Also get next event
	next, _ := m.calendar.GetNextEvent(ctx)
	if next != nil {
		result.SetMetadata("next_event", next.Summary)
		result.SetMetadata("next_event_time", next.Start.Format(time.RFC3339))
	}

	return result, nil
}

// GetGmailClient returns the Gmail client if authenticated.
func (m *Manager) GetGmailClient() *GmailClient {
	if m.gmail != nil && m.gmailOAuth.IsAuthenticated() {
		return m.gmail
	}
	return nil
}

// GetSlackClient returns the Slack client if authenticated.
func (m *Manager) GetSlackClient() *SlackClient {
	if m.slack != nil && m.slackOAuth.IsAuthenticated() {
		return m.slack
	}
	return nil
}

// GetCalendarClient returns the Calendar client if authenticated.
func (m *Manager) GetCalendarClient() *CalendarClient {
	if m.calendar != nil && m.calendarOAuth.IsAuthenticated() {
		return m.calendar
	}
	return nil
}

// Close securely closes all resources.
func (m *Manager) Close() {
	if m.tokenStore != nil {
		m.tokenStore.Close()
	}
}
