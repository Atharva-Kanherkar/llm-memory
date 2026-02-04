// Package platform handles detection of the operating system and display server.
//
// This is important because different platforms need different methods to:
// - Capture screenshots (grim on Wayland, scrot on X11, screencapture on macOS)
// - Get active window info (hyprctl on Hyprland, xdotool on X11, etc.)
// - Access clipboard (wl-paste on Wayland, xclip on X11, pbpaste on macOS)
package platform

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// DisplayServer represents the display server type.
// In Go, we often use string constants instead of enums.
type DisplayServer string

const (
	DisplayServerHyprland DisplayServer = "hyprland"
	DisplayServerSway     DisplayServer = "sway"
	DisplayServerWayland  DisplayServer = "wayland"  // Generic Wayland (GNOME, KDE)
	DisplayServerX11      DisplayServer = "x11"
	DisplayServerMacOS    DisplayServer = "macos"
	DisplayServerUnknown  DisplayServer = "unknown"
)

// Platform holds information about the detected platform.
type Platform struct {
	// OS is the operating system: "linux", "darwin" (macOS), "windows"
	OS string

	// DisplayServer is the specific display server being used
	DisplayServer DisplayServer

	// Available tools - these are set based on what's installed
	HasHyprctl  bool // Hyprland control tool
	HasGrim     bool // Wayland screenshot tool
	HasSlurp    bool // Wayland region selector
	HasWlPaste  bool // Wayland clipboard
	HasXdotool  bool // X11 window info
	HasScrot    bool // X11 screenshot
	HasXclip    bool // X11 clipboard
	HasTesseract bool // OCR
}

// String returns a human-readable description of the platform.
// This is Go's equivalent of toString() in other languages.
// When you call fmt.Printf("%s", plat), this method is called.
func (p *Platform) String() string {
	return fmt.Sprintf("%s/%s", p.OS, p.DisplayServer)
}

// Detect figures out what platform we're running on.
// It checks the OS, then probes for display server and available tools.
func Detect() (*Platform, error) {
	p := &Platform{
		OS: runtime.GOOS, // "linux", "darwin", "windows"
	}

	// Detect display server
	p.DisplayServer = detectDisplayServer()

	// Check for available tools
	p.HasHyprctl = commandExists("hyprctl")
	p.HasGrim = commandExists("grim")
	p.HasSlurp = commandExists("slurp")
	p.HasWlPaste = commandExists("wl-paste")
	p.HasXdotool = commandExists("xdotool")
	p.HasScrot = commandExists("scrot")
	p.HasXclip = commandExists("xclip")
	p.HasTesseract = commandExists("tesseract")

	return p, nil
}

// detectDisplayServer figures out which display server is running.
func detectDisplayServer() DisplayServer {
	// On macOS, it's always macOS
	if runtime.GOOS == "darwin" {
		return DisplayServerMacOS
	}

	// Check for Hyprland first (most specific)
	// Hyprland sets HYPRLAND_INSTANCE_SIGNATURE environment variable
	if os.Getenv("HYPRLAND_INSTANCE_SIGNATURE") != "" {
		return DisplayServerHyprland
	}

	// Check for Sway
	if os.Getenv("SWAYSOCK") != "" {
		return DisplayServerSway
	}

	// Check for generic Wayland
	// XDG_SESSION_TYPE is set by systemd/login managers
	sessionType := os.Getenv("XDG_SESSION_TYPE")
	if sessionType == "wayland" {
		return DisplayServerWayland
	}

	// Check WAYLAND_DISPLAY as another indicator
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		return DisplayServerWayland
	}

	// Check for X11
	if sessionType == "x11" || os.Getenv("DISPLAY") != "" {
		return DisplayServerX11
	}

	return DisplayServerUnknown
}

// commandExists checks if a command is available in PATH.
// exec.LookPath searches for the command like `which` does.
func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// IsWayland returns true if we're on any Wayland compositor.
func (p *Platform) IsWayland() bool {
	switch p.DisplayServer {
	case DisplayServerHyprland, DisplayServerSway, DisplayServerWayland:
		return true
	default:
		return false
	}
}

// CanCaptureWindow returns true if we have tools to capture window info.
func (p *Platform) CanCaptureWindow() bool {
	switch p.DisplayServer {
	case DisplayServerHyprland:
		return p.HasHyprctl
	case DisplayServerX11:
		return p.HasXdotool
	case DisplayServerMacOS:
		return true // macOS always has Accessibility APIs
	default:
		return false
	}
}

// CanCaptureScreen returns true if we have tools to capture screenshots.
func (p *Platform) CanCaptureScreen() bool {
	switch p.DisplayServer {
	case DisplayServerHyprland, DisplayServerSway, DisplayServerWayland:
		return p.HasGrim
	case DisplayServerX11:
		return p.HasScrot
	case DisplayServerMacOS:
		return true // macOS always has screencapture
	default:
		return false
	}
}

// GetWindowInfo executes the appropriate command to get active window info.
// Returns the raw output - parsing happens in the window capture module.
func (p *Platform) GetWindowInfo() ([]byte, error) {
	switch p.DisplayServer {
	case DisplayServerHyprland:
		// hyprctl activewindow -j returns JSON
		return exec.Command("hyprctl", "activewindow", "-j").Output()
	case DisplayServerX11:
		// xdotool getactivewindow followed by xprop
		// This is more complex, we'll handle it in the X11 capturer
		return nil, fmt.Errorf("X11 window capture not implemented yet")
	case DisplayServerMacOS:
		return nil, fmt.Errorf("macOS window capture not implemented yet")
	default:
		return nil, fmt.Errorf("unsupported display server: %s", p.DisplayServer)
	}
}

// SupportedFeatures returns a human-readable list of what we can capture.
func (p *Platform) SupportedFeatures() []string {
	features := []string{}

	if p.CanCaptureWindow() {
		features = append(features, "window tracking")
	}
	if p.CanCaptureScreen() {
		features = append(features, "screen capture")
	}
	if p.HasWlPaste || p.HasXclip {
		features = append(features, "clipboard")
	}
	if p.HasTesseract {
		features = append(features, "OCR")
	}

	if len(features) == 0 {
		return []string{"none - missing required tools"}
	}

	return features
}

// CheckRequirements logs warnings about missing tools.
func (p *Platform) CheckRequirements() []string {
	var missing []string

	switch p.DisplayServer {
	case DisplayServerHyprland, DisplayServerSway, DisplayServerWayland:
		if !p.HasGrim {
			missing = append(missing, "grim (install: sudo pacman -S grim)")
		}
		if !p.HasWlPaste {
			missing = append(missing, "wl-paste (install: sudo pacman -S wl-clipboard)")
		}
	case DisplayServerX11:
		if !p.HasXdotool {
			missing = append(missing, "xdotool (install: sudo pacman -S xdotool)")
		}
		if !p.HasScrot {
			missing = append(missing, "scrot (install: sudo pacman -S scrot)")
		}
	}

	if !p.HasTesseract {
		missing = append(missing, "tesseract (install: sudo pacman -S tesseract tesseract-data-eng)")
	}

	return missing
}
