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

// CalendarClient fetches events from Google Calendar API.
type CalendarClient struct {
	oauthClient *oauth.Client
	httpClient  *http.Client
}

// CalendarEvent represents a calendar event.
type CalendarEvent struct {
	ID          string
	Summary     string
	Description string
	Location    string
	Start       time.Time
	End         time.Time
	AllDay      bool
	Status      string // confirmed, tentative, cancelled
	Organizer   string
	Attendees   []string
	MeetLink    string
}

// NewCalendarClient creates a new Calendar client.
func NewCalendarClient(oauthClient *oauth.Client) *CalendarClient {
	return &CalendarClient{
		oauthClient: oauthClient,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
	}
}

// calendarRequest makes an authenticated request to Calendar API.
func (c *CalendarClient) calendarRequest(ctx context.Context, method, endpoint string, body io.Reader) ([]byte, error) {
	token, err := c.oauthClient.GetValidToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("authentication required: %w", err)
	}

	reqURL := "https://www.googleapis.com/calendar/v3" + endpoint

	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
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

// GetTodaysEvents fetches events for today.
func (c *CalendarClient) GetTodaysEvents(ctx context.Context) ([]CalendarEvent, error) {
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	return c.GetEventsBetween(ctx, startOfDay, endOfDay)
}

// GetUpcomingEvents fetches events for the next N days.
func (c *CalendarClient) GetUpcomingEvents(ctx context.Context, days int) ([]CalendarEvent, error) {
	now := time.Now()
	end := now.AddDate(0, 0, days)

	return c.GetEventsBetween(ctx, now, end)
}

// GetEventsBetween fetches events between two times.
func (c *CalendarClient) GetEventsBetween(ctx context.Context, start, end time.Time) ([]CalendarEvent, error) {
	params := url.Values{}
	params.Set("timeMin", start.Format(time.RFC3339))
	params.Set("timeMax", end.Format(time.RFC3339))
	params.Set("singleEvents", "true")
	params.Set("orderBy", "startTime")
	params.Set("maxResults", "50")

	endpoint := "/calendars/primary/events?" + params.Encode()
	body, err := c.calendarRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Items []struct {
			ID          string `json:"id"`
			Summary     string `json:"summary"`
			Description string `json:"description"`
			Location    string `json:"location"`
			Status      string `json:"status"`
			Start       struct {
				DateTime string `json:"dateTime"`
				Date     string `json:"date"`
			} `json:"start"`
			End struct {
				DateTime string `json:"dateTime"`
				Date     string `json:"date"`
			} `json:"end"`
			Organizer struct {
				Email string `json:"email"`
			} `json:"organizer"`
			Attendees []struct {
				Email          string `json:"email"`
				ResponseStatus string `json:"responseStatus"`
			} `json:"attendees"`
			HangoutLink    string `json:"hangoutLink"`
			ConferenceData struct {
				EntryPoints []struct {
					URI string `json:"uri"`
				} `json:"entryPoints"`
			} `json:"conferenceData"`
		} `json:"items"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse events: %w", err)
	}

	var events []CalendarEvent
	for _, item := range resp.Items {
		event := CalendarEvent{
			ID:          item.ID,
			Summary:     item.Summary,
			Description: item.Description,
			Location:    item.Location,
			Status:      item.Status,
			Organizer:   item.Organizer.Email,
		}

		// Parse start time
		if item.Start.DateTime != "" {
			if t, err := time.Parse(time.RFC3339, item.Start.DateTime); err == nil {
				event.Start = t
			}
		} else if item.Start.Date != "" {
			// All-day event
			if t, err := time.Parse("2006-01-02", item.Start.Date); err == nil {
				event.Start = t
				event.AllDay = true
			}
		}

		// Parse end time
		if item.End.DateTime != "" {
			if t, err := time.Parse(time.RFC3339, item.End.DateTime); err == nil {
				event.End = t
			}
		} else if item.End.Date != "" {
			if t, err := time.Parse("2006-01-02", item.End.Date); err == nil {
				event.End = t
			}
		}

		// Collect attendees
		for _, att := range item.Attendees {
			event.Attendees = append(event.Attendees, att.Email)
		}

		// Get meeting link
		if item.HangoutLink != "" {
			event.MeetLink = item.HangoutLink
		} else if len(item.ConferenceData.EntryPoints) > 0 {
			event.MeetLink = item.ConferenceData.EntryPoints[0].URI
		}

		events = append(events, event)
	}

	return events, nil
}

// GetNextEvent fetches the next upcoming event.
func (c *CalendarClient) GetNextEvent(ctx context.Context) (*CalendarEvent, error) {
	now := time.Now()
	end := now.Add(7 * 24 * time.Hour) // Look up to a week ahead

	params := url.Values{}
	params.Set("timeMin", now.Format(time.RFC3339))
	params.Set("timeMax", end.Format(time.RFC3339))
	params.Set("singleEvents", "true")
	params.Set("orderBy", "startTime")
	params.Set("maxResults", "1")

	endpoint := "/calendars/primary/events?" + params.Encode()
	body, err := c.calendarRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Items []struct {
			ID      string `json:"id"`
			Summary string `json:"summary"`
			Start   struct {
				DateTime string `json:"dateTime"`
				Date     string `json:"date"`
			} `json:"start"`
			End struct {
				DateTime string `json:"dateTime"`
			} `json:"end"`
			HangoutLink string `json:"hangoutLink"`
		} `json:"items"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	if len(resp.Items) == 0 {
		return nil, nil // No upcoming events
	}

	item := resp.Items[0]
	event := &CalendarEvent{
		ID:       item.ID,
		Summary:  item.Summary,
		MeetLink: item.HangoutLink,
	}

	if item.Start.DateTime != "" {
		if t, err := time.Parse(time.RFC3339, item.Start.DateTime); err == nil {
			event.Start = t
		}
	} else if item.Start.Date != "" {
		if t, err := time.Parse("2006-01-02", item.Start.Date); err == nil {
			event.Start = t
			event.AllDay = true
		}
	}

	if item.End.DateTime != "" {
		if t, err := time.Parse(time.RFC3339, item.End.DateTime); err == nil {
			event.End = t
		}
	}

	return event, nil
}

// FreeBusy checks if a time slot is free.
func (c *CalendarClient) FreeBusy(ctx context.Context, start, end time.Time) (bool, error) {
	events, err := c.GetEventsBetween(ctx, start, end)
	if err != nil {
		return false, err
	}

	// Filter out cancelled events
	for _, e := range events {
		if e.Status != "cancelled" {
			return false, nil // Time is busy
		}
	}

	return true, nil // Time is free
}
