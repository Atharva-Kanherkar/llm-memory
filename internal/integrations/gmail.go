// Package integrations provides connectors to external services.
//
// SECURITY: All API calls use encrypted OAuth tokens.
// No sensitive data is logged.
package integrations

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Atharva-Kanherkar/mnemosyne/internal/oauth"
)

// GmailClient fetches emails from Gmail API.
type GmailClient struct {
	oauthClient *oauth.Client
	httpClient  *http.Client
}

// Email represents a simplified email message.
type Email struct {
	ID        string
	ThreadID  string
	From      string
	To        string
	Subject   string
	Snippet   string
	Body      string
	Timestamp time.Time
	Labels    []string
	IsUnread  bool
}

// NewGmailClient creates a new Gmail client.
func NewGmailClient(oauthClient *oauth.Client) *GmailClient {
	return &GmailClient{
		oauthClient: oauthClient,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
	}
}

// gmailRequest makes an authenticated request to Gmail API.
func (g *GmailClient) gmailRequest(ctx context.Context, method, endpoint string, body io.Reader) ([]byte, error) {
	token, err := g.oauthClient.GetValidToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("authentication required: %w", err)
	}

	url := "https://gmail.googleapis.com/gmail/v1/users/me" + endpoint

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// GetRecentEmails fetches recent emails from the inbox.
func (g *GmailClient) GetRecentEmails(ctx context.Context, maxResults int) ([]Email, error) {
	// List messages
	endpoint := fmt.Sprintf("/messages?maxResults=%d&q=in:inbox", maxResults)
	listBody, err := g.gmailRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var listResp struct {
		Messages []struct {
			ID       string `json:"id"`
			ThreadID string `json:"threadId"`
		} `json:"messages"`
	}

	if err := json.Unmarshal(listBody, &listResp); err != nil {
		return nil, fmt.Errorf("failed to parse message list: %w", err)
	}

	// Fetch each message
	var emails []Email
	for _, msg := range listResp.Messages {
		email, err := g.getMessage(ctx, msg.ID)
		if err != nil {
			continue // Skip messages we can't fetch
		}
		emails = append(emails, *email)
	}

	return emails, nil
}

// getMessage fetches a single message by ID.
func (g *GmailClient) getMessage(ctx context.Context, messageID string) (*Email, error) {
	endpoint := fmt.Sprintf("/messages/%s?format=full", messageID)
	body, err := g.gmailRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var msgResp struct {
		ID           string   `json:"id"`
		ThreadID     string   `json:"threadId"`
		Snippet      string   `json:"snippet"`
		LabelIDs     []string `json:"labelIds"`
		InternalDate string   `json:"internalDate"`
		Payload      struct {
			Headers []struct {
				Name  string `json:"name"`
				Value string `json:"value"`
			} `json:"headers"`
			Body struct {
				Data string `json:"data"`
			} `json:"body"`
			Parts []struct {
				MimeType string `json:"mimeType"`
				Body     struct {
					Data string `json:"data"`
				} `json:"body"`
			} `json:"parts"`
		} `json:"payload"`
	}

	if err := json.Unmarshal(body, &msgResp); err != nil {
		return nil, fmt.Errorf("failed to parse message: %w", err)
	}

	email := &Email{
		ID:       msgResp.ID,
		ThreadID: msgResp.ThreadID,
		Snippet:  msgResp.Snippet,
		Labels:   msgResp.LabelIDs,
	}

	// Check if unread
	for _, label := range msgResp.LabelIDs {
		if label == "UNREAD" {
			email.IsUnread = true
			break
		}
	}

	// Parse timestamp
	if ts, err := parseGmailTimestamp(msgResp.InternalDate); err == nil {
		email.Timestamp = ts
	}

	// Extract headers
	for _, header := range msgResp.Payload.Headers {
		switch header.Name {
		case "From":
			email.From = header.Value
		case "To":
			email.To = header.Value
		case "Subject":
			email.Subject = header.Value
		}
	}

	// Extract body (prefer plain text)
	if msgResp.Payload.Body.Data != "" {
		email.Body = decodeBase64URL(msgResp.Payload.Body.Data)
	} else {
		for _, part := range msgResp.Payload.Parts {
			if part.MimeType == "text/plain" && part.Body.Data != "" {
				email.Body = decodeBase64URL(part.Body.Data)
				break
			}
		}
	}

	// Truncate body if too long
	if len(email.Body) > 1000 {
		email.Body = email.Body[:1000] + "..."
	}

	return email, nil
}

// GetUnreadCount returns the number of unread emails.
func (g *GmailClient) GetUnreadCount(ctx context.Context) (int, error) {
	endpoint := "/messages?q=is:unread+in:inbox&maxResults=1"
	body, err := g.gmailRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return 0, err
	}

	var resp struct {
		ResultSizeEstimate int `json:"resultSizeEstimate"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, err
	}

	return resp.ResultSizeEstimate, nil
}

// SearchEmails searches for emails matching a query.
func (g *GmailClient) SearchEmails(ctx context.Context, query string, maxResults int) ([]Email, error) {
	endpoint := fmt.Sprintf("/messages?maxResults=%d&q=%s", maxResults, query)
	listBody, err := g.gmailRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var listResp struct {
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
	}

	if err := json.Unmarshal(listBody, &listResp); err != nil {
		return nil, err
	}

	var emails []Email
	for _, msg := range listResp.Messages {
		email, err := g.getMessage(ctx, msg.ID)
		if err != nil {
			continue
		}
		emails = append(emails, *email)
	}

	return emails, nil
}

// decodeBase64URL decodes base64 URL-safe encoded data.
func decodeBase64URL(data string) string {
	// Replace URL-safe characters
	data = strings.ReplaceAll(data, "-", "+")
	data = strings.ReplaceAll(data, "_", "/")

	// Add padding if needed
	switch len(data) % 4 {
	case 2:
		data += "=="
	case 3:
		data += "="
	}

	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return ""
	}

	return string(decoded)
}

// parseGmailTimestamp parses Gmail's internal date format (milliseconds since epoch).
func parseGmailTimestamp(ts string) (time.Time, error) {
	var ms int64
	if _, err := fmt.Sscanf(ts, "%d", &ms); err != nil {
		return time.Time{}, err
	}
	return time.UnixMilli(ms), nil
}
