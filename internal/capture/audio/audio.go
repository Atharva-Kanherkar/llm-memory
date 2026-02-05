// Package audio provides ambient audio capture.
//
// Uses PipeWire's pw-record to capture audio snippets.
// This is OPT-IN only - must be explicitly enabled in config.
//
// Captures short audio "anchors" (5-10 seconds) that can be:
// - Transcribed using Whisper for text extraction
// - Used as context cues ("what was I hearing when I worked on this?")
//
// Audio is captured from the default microphone input.
package audio

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/Atharva-Kanherkar/mnemosyne/internal/capture"
	"github.com/Atharva-Kanherkar/mnemosyne/internal/platform"
)

// Capturer captures audio snippets.
type Capturer struct {
	platform *platform.Platform

	// Duration is how long to record (default 5 seconds)
	Duration time.Duration

	// Format is the audio format (wav, flac, etc.)
	Format string

	// SampleRate is the audio sample rate
	SampleRate int

	// Enabled must be explicitly set to true (opt-in)
	Enabled bool
}

// New creates a new audio Capturer.
// Note: Audio capture is disabled by default for privacy.
func New(plat *platform.Platform) *Capturer {
	return &Capturer{
		platform:   plat,
		Duration:   5 * time.Second,
		Format:     "wav",
		SampleRate: 16000, // Good for speech recognition
		Enabled:    false, // Must be explicitly enabled
	}
}

// Name returns the capturer identifier.
func (c *Capturer) Name() string {
	return "audio"
}

// Available checks if audio capture is possible.
func (c *Capturer) Available() bool {
	if !c.Enabled {
		return false
	}

	// Check for pw-record (PipeWire)
	if _, err := exec.LookPath("pw-record"); err == nil {
		return true
	}

	// Check for parecord (PulseAudio)
	if _, err := exec.LookPath("parecord"); err == nil {
		return true
	}

	return false
}

// Enable turns on audio capture.
func (c *Capturer) Enable() {
	c.Enabled = true
}

// Disable turns off audio capture.
func (c *Capturer) Disable() {
	c.Enabled = false
}

// Capture records an audio snippet.
func (c *Capturer) Capture(ctx context.Context) (*capture.Result, error) {
	if !c.Enabled {
		return nil, fmt.Errorf("audio capture is disabled (privacy: must be explicitly enabled)")
	}

	// Try PipeWire first, fall back to PulseAudio
	if _, err := exec.LookPath("pw-record"); err == nil {
		return c.capturePipeWire(ctx)
	}

	if _, err := exec.LookPath("parecord"); err == nil {
		return c.capturePulseAudio(ctx)
	}

	return nil, fmt.Errorf("no audio capture tool available (need pw-record or parecord)")
}

// capturePipeWire records audio using PipeWire's pw-record.
func (c *Capturer) capturePipeWire(ctx context.Context) (*capture.Result, error) {
	// Create a context with timeout for the recording duration
	recordCtx, cancel := context.WithTimeout(ctx, c.Duration+2*time.Second)
	defer cancel()

	// pw-record arguments:
	// --rate: sample rate
	// --channels: mono for speech
	// --format: sample format
	// -: output to stdout
	cmd := exec.CommandContext(recordCtx, "pw-record",
		"--rate", fmt.Sprintf("%d", c.SampleRate),
		"--channels", "1",
		"--format", "s16", // 16-bit signed
		"-", // stdout
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Start recording
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start pw-record: %w", err)
	}

	// Wait for duration
	select {
	case <-time.After(c.Duration):
		// Duration elapsed, kill the process
		cmd.Process.Kill()
	case <-ctx.Done():
		// Context cancelled
		cmd.Process.Kill()
		return nil, ctx.Err()
	}

	// Wait for process to exit
	cmd.Wait() // Ignore error since we killed it

	audioData := stdout.Bytes()

	if len(audioData) == 0 {
		return nil, fmt.Errorf("no audio data captured (stderr: %s)", stderr.String())
	}

	result := capture.NewResult("audio")
	result.RawData = audioData
	result.SetMetadata("format", "raw")
	result.SetMetadata("sample_rate", fmt.Sprintf("%d", c.SampleRate))
	result.SetMetadata("channels", "1")
	result.SetMetadata("bits", "16")
	result.SetMetadata("duration_ms", fmt.Sprintf("%d", c.Duration.Milliseconds()))
	result.SetMetadata("size_bytes", fmt.Sprintf("%d", len(audioData)))

	return result, nil
}

// capturePulseAudio records audio using PulseAudio's parecord.
func (c *Capturer) capturePulseAudio(ctx context.Context) (*capture.Result, error) {
	recordCtx, cancel := context.WithTimeout(ctx, c.Duration+2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(recordCtx, "parecord",
		"--rate", fmt.Sprintf("%d", c.SampleRate),
		"--channels", "1",
		"--format", "s16le",
		"--raw",
	)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start parecord: %w", err)
	}

	select {
	case <-time.After(c.Duration):
		cmd.Process.Kill()
	case <-ctx.Done():
		cmd.Process.Kill()
		return nil, ctx.Err()
	}

	cmd.Wait()

	audioData := stdout.Bytes()

	result := capture.NewResult("audio")
	result.RawData = audioData
	result.SetMetadata("format", "raw")
	result.SetMetadata("sample_rate", fmt.Sprintf("%d", c.SampleRate))
	result.SetMetadata("channels", "1")
	result.SetMetadata("duration_ms", fmt.Sprintf("%d", c.Duration.Milliseconds()))

	return result, nil
}

// SetDuration sets the recording duration.
func (c *Capturer) SetDuration(d time.Duration) {
	if d < time.Second {
		d = time.Second
	}
	if d > 30*time.Second {
		d = 30 * time.Second
	}
	c.Duration = d
}
