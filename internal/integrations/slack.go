package integrations

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/Atharva-Kanherkar/mnemosyne/internal/oauth"
)

// SlackClient fetches messages from Slack API.
type SlackClient struct {
	oauthClient *oauth.Client
	httpClient  *http.Client
}

// SlackMessage represents a Slack message.
type SlackMessage struct {
	ChannelID   string
	ChannelName string
	User        string
	UserName    string
	Text        string
	Timestamp   time.Time
	ThreadTS    string
	IsThread    bool
}

// SlackChannel represents a Slack channel.
type SlackChannel struct {
	ID        string
	Name      string
	IsMember  bool
	IsPrivate bool
}

// NewSlackClient creates a new Slack client.
func NewSlackClient(oauthClient *oauth.Client) *SlackClient {
	return &SlackClient{
		oauthClient: oauthClient,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
	}
}

// slackRequest makes an authenticated request to Slack API.
func (s *SlackClient) slackRequest(ctx context.Context, method, endpoint string, params url.Values) ([]byte, error) {
	token, err := s.oauthClient.GetValidToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("authentication required: %w", err)
	}

	reqURL := "https://slack.com/api" + endpoint
	if params != nil && method == "GET" {
		reqURL += "?" + params.Encode()
	}

	var body io.Reader
	if params != nil && method == "POST" {
		body = nil // Slack uses query params even for POST
		reqURL += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Check Slack-specific error format
	var slackResp struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}

	if err := json.Unmarshal(respBody, &slackResp); err == nil {
		if !slackResp.OK {
			return nil, fmt.Errorf("Slack API error: %s", slackResp.Error)
		}
	}

	return respBody, nil
}

// GetChannels fetches the list of channels the user is a member of.
func (s *SlackClient) GetChannels(ctx context.Context) ([]SlackChannel, error) {
	params := url.Values{}
	params.Set("types", "public_channel,private_channel")
	params.Set("exclude_archived", "true")

	body, err := s.slackRequest(ctx, "GET", "/conversations.list", params)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Channels []struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			IsMember  bool   `json:"is_member"`
			IsPrivate bool   `json:"is_private"`
		} `json:"channels"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse channels: %w", err)
	}

	var channels []SlackChannel
	for _, ch := range resp.Channels {
		channels = append(channels, SlackChannel{
			ID:        ch.ID,
			Name:      ch.Name,
			IsMember:  ch.IsMember,
			IsPrivate: ch.IsPrivate,
		})
	}

	return channels, nil
}

// GetRecentMessages fetches recent messages from a channel.
func (s *SlackClient) GetRecentMessages(ctx context.Context, channelID string, limit int) ([]SlackMessage, error) {
	params := url.Values{}
	params.Set("channel", channelID)
	params.Set("limit", fmt.Sprintf("%d", limit))

	body, err := s.slackRequest(ctx, "GET", "/conversations.history", params)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Messages []struct {
			User     string `json:"user"`
			Text     string `json:"text"`
			TS       string `json:"ts"`
			ThreadTS string `json:"thread_ts"`
		} `json:"messages"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse messages: %w", err)
	}

	var messages []SlackMessage
	for _, msg := range resp.Messages {
		m := SlackMessage{
			ChannelID: channelID,
			User:      msg.User,
			Text:      msg.Text,
			ThreadTS:  msg.ThreadTS,
			IsThread:  msg.ThreadTS != "" && msg.ThreadTS != msg.TS,
		}

		// Parse timestamp
		if ts, err := parseSlackTimestamp(msg.TS); err == nil {
			m.Timestamp = ts
		}

		messages = append(messages, m)
	}

	return messages, nil
}

// GetAllRecentMessages fetches recent messages from all channels.
func (s *SlackClient) GetAllRecentMessages(ctx context.Context, messagesPerChannel int) ([]SlackMessage, error) {
	channels, err := s.GetChannels(ctx)
	if err != nil {
		return nil, err
	}

	var allMessages []SlackMessage

	for _, ch := range channels {
		if !ch.IsMember {
			continue
		}

		messages, err := s.GetRecentMessages(ctx, ch.ID, messagesPerChannel)
		if err != nil {
			continue // Skip channels we can't access
		}

		// Add channel name to messages
		for i := range messages {
			messages[i].ChannelName = ch.Name
		}

		allMessages = append(allMessages, messages...)
	}

	return allMessages, nil
}

// GetUserInfo fetches information about a user.
func (s *SlackClient) GetUserInfo(ctx context.Context, userID string) (string, error) {
	params := url.Values{}
	params.Set("user", userID)

	body, err := s.slackRequest(ctx, "GET", "/users.info", params)
	if err != nil {
		return "", err
	}

	var resp struct {
		User struct {
			Name     string `json:"name"`
			RealName string `json:"real_name"`
		} `json:"user"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return "", err
	}

	if resp.User.RealName != "" {
		return resp.User.RealName, nil
	}
	return resp.User.Name, nil
}

// SearchMessages searches for messages matching a query.
func (s *SlackClient) SearchMessages(ctx context.Context, query string, count int) ([]SlackMessage, error) {
	params := url.Values{}
	params.Set("query", query)
	params.Set("count", fmt.Sprintf("%d", count))

	body, err := s.slackRequest(ctx, "GET", "/search.messages", params)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Messages struct {
			Matches []struct {
				Channel struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"channel"`
				User string `json:"user"`
				Text string `json:"text"`
				TS   string `json:"ts"`
			} `json:"matches"`
		} `json:"messages"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse search results: %w", err)
	}

	var messages []SlackMessage
	for _, match := range resp.Messages.Matches {
		m := SlackMessage{
			ChannelID:   match.Channel.ID,
			ChannelName: match.Channel.Name,
			User:        match.User,
			Text:        match.Text,
		}

		if ts, err := parseSlackTimestamp(match.TS); err == nil {
			m.Timestamp = ts
		}

		messages = append(messages, m)
	}

	return messages, nil
}

// parseSlackTimestamp parses Slack's timestamp format (seconds.microseconds).
func parseSlackTimestamp(ts string) (time.Time, error) {
	var secs float64
	if _, err := fmt.Sscanf(ts, "%f", &secs); err != nil {
		return time.Time{}, err
	}
	return time.Unix(int64(secs), 0), nil
}
