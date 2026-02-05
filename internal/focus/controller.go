package focus

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Controller handles Hyprland window control operations.
type Controller struct{}

// NewController creates a new window controller.
func NewController() *Controller {
	return &Controller{}
}

// GetActiveWindow returns information about the currently active window.
func (c *Controller) GetActiveWindow() (*Window, error) {
	cmd := exec.Command("hyprctl", "activewindow", "-j")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get active window: %w", err)
	}

	var window Window
	if err := json.Unmarshal(output, &window); err != nil {
		return nil, fmt.Errorf("failed to parse window info: %w", err)
	}

	return &window, nil
}

// CloseWindow closes a window by its address.
func (c *Controller) CloseWindow(address string) error {
	cmd := exec.Command("hyprctl", "dispatch", "closewindow", "address:"+address)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to close window: %w, output: %s", err, string(output))
	}
	return nil
}

// CloseBrowserTab closes the current tab in a browser using Ctrl+W.
// This is safer than closing the whole window for browsers.
func (c *Controller) CloseBrowserTab() error {
	// Use wtype (Wayland) or xdotool (X11) to send Ctrl+W
	// Try wtype first (for Wayland/Hyprland)
	cmd := exec.Command("wtype", "-M", "ctrl", "-P", "w", "-p", "w", "-m", "ctrl")
	if err := cmd.Run(); err != nil {
		// Fallback to ydotool if wtype fails
		cmd = exec.Command("ydotool", "key", "29:1", "17:1", "17:0", "29:0") // Ctrl+W
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to send Ctrl+W: %w (install wtype or ydotool)", err)
		}
	}
	return nil
}

// FocusWindow brings a window to focus by its address.
func (c *Controller) FocusWindow(address string) error {
	cmd := exec.Command("hyprctl", "dispatch", "focuswindow", "address:"+address)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to focus window: %w, output: %s", err, string(output))
	}
	return nil
}

// ListWindows returns all open windows.
func (c *Controller) ListWindows() ([]Window, error) {
	cmd := exec.Command("hyprctl", "clients", "-j")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list windows: %w", err)
	}

	var windows []Window
	if err := json.Unmarshal(output, &windows); err != nil {
		return nil, fmt.Errorf("failed to parse window list: %w", err)
	}

	return windows, nil
}

// FindWindowByClass finds windows by their class name.
func (c *Controller) FindWindowByClass(class string) ([]Window, error) {
	windows, err := c.ListWindows()
	if err != nil {
		return nil, err
	}

	classLower := strings.ToLower(class)
	var matches []Window
	for _, w := range windows {
		if strings.ToLower(w.Class) == classLower {
			matches = append(matches, w)
		}
	}

	return matches, nil
}

// IsHyprlandAvailable checks if Hyprland is running.
func (c *Controller) IsHyprlandAvailable() bool {
	cmd := exec.Command("hyprctl", "version")
	err := cmd.Run()
	return err == nil
}

// Border color constants (RGBA format for Hyprland)
const (
	BorderColorAllowed = "rgba(00ff00aa)" // Green with some transparency
	BorderColorWarned  = "rgba(ff5500ff)" // Orange/red, fully opaque
	BorderColorBlocked = "rgba(ff0000ff)" // Red, fully opaque
	BorderColorNormal  = "rgba(33ccffee)" // Default cyan-ish
)

// SetWindowBorder sets the border color of a window by address.
func (c *Controller) SetWindowBorder(address, color string) error {
	// Use hyprctl setprop to change the active border color for this specific window
	cmd := exec.Command("hyprctl", "setprop", "address:"+address, "activebordercolor", color)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set border: %w, output: %s", err, string(output))
	}
	return nil
}

// ResetWindowBorder resets a window's border to default.
func (c *Controller) ResetWindowBorder(address string) error {
	return c.SetWindowBorder(address, BorderColorNormal)
}

// FlashWindowRed makes a window border flash red (for warnings).
func (c *Controller) FlashWindowRed(address string) error {
	return c.SetWindowBorder(address, BorderColorBlocked)
}

// SetWindowAllowed sets a green border on an allowed window.
func (c *Controller) SetWindowAllowed(address string) error {
	return c.SetWindowBorder(address, BorderColorAllowed)
}

// SetWindowWarned sets an orange/red border on a warned window.
func (c *Controller) SetWindowWarned(address string) error {
	return c.SetWindowBorder(address, BorderColorWarned)
}
