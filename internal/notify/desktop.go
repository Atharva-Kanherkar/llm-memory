// Package notify handles notifications to the user.
package notify

import (
	"fmt"
	"os/exec"
)

// Urgency levels for desktop notifications.
type Urgency string

const (
	UrgencyLow      Urgency = "low"
	UrgencyNormal   Urgency = "normal"
	UrgencyCritical Urgency = "critical"
)

// DesktopNotifier sends desktop notifications via notify-send.
type DesktopNotifier struct {
	appName string
}

// NewDesktopNotifier creates a new desktop notifier.
func NewDesktopNotifier() *DesktopNotifier {
	return &DesktopNotifier{
		appName: "Mnemosyne",
	}
}

// Available checks if notify-send is available.
func (n *DesktopNotifier) Available() bool {
	_, err := exec.LookPath("notify-send")
	return err == nil
}

// Send sends a desktop notification.
func (n *DesktopNotifier) Send(title, body string, urgency Urgency) error {
	if !n.Available() {
		return nil // Silently skip if not available
	}

	args := []string{
		"--app-name=" + n.appName,
		"--urgency=" + string(urgency),
	}

	// Add icon hint for different urgency levels
	switch urgency {
	case UrgencyCritical:
		args = append(args, "--icon=dialog-warning")
	case UrgencyNormal:
		args = append(args, "--icon=dialog-information")
	default:
		args = append(args, "--icon=dialog-information")
	}

	args = append(args, title, body)

	cmd := exec.Command("notify-send", args...)
	return cmd.Run()
}

// SendWithTimeout sends a notification that expires after given milliseconds.
func (n *DesktopNotifier) SendWithTimeout(title, body string, urgency Urgency, timeoutMs int) error {
	if !n.Available() {
		return nil
	}

	args := []string{
		"--app-name=" + n.appName,
		"--urgency=" + string(urgency),
		fmt.Sprintf("--expire-time=%d", timeoutMs),
		title,
		body,
	}

	cmd := exec.Command("notify-send", args...)
	return cmd.Run()
}
