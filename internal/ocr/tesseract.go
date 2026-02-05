// Package ocr provides text extraction from images using Tesseract.
//
// Tesseract is an open-source OCR engine. We shell out to it
// rather than using CGO bindings for simplicity.
//
// Install on Arch: sudo pacman -S tesseract tesseract-data-eng
package ocr

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Engine provides OCR capabilities.
type Engine struct {
	// Language is the Tesseract language code (default: eng)
	Language string

	// available caches whether tesseract is installed
	available bool
}

// New creates a new OCR Engine.
func New() *Engine {
	e := &Engine{
		Language: "eng",
	}
	e.available = e.checkAvailable()
	return e
}

// checkAvailable checks if tesseract is installed.
func (e *Engine) checkAvailable() bool {
	_, err := exec.LookPath("tesseract")
	return err == nil
}

// Available returns whether OCR is available on this system.
func (e *Engine) Available() bool {
	return e.available
}

// ExtractText extracts text from image bytes (PNG, JPG, etc).
func (e *Engine) ExtractText(ctx context.Context, imageData []byte) (string, error) {
	if !e.available {
		return "", fmt.Errorf("tesseract not installed")
	}

	// tesseract reads from stdin with "-" and outputs to stdout with "stdout"
	// Usage: tesseract stdin stdout -l eng
	cmd := exec.CommandContext(ctx, "tesseract",
		"stdin", "stdout",
		"-l", e.Language,
		"--psm", "3", // Fully automatic page segmentation
	)

	cmd.Stdin = bytes.NewReader(imageData)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("tesseract failed: %w (stderr: %s)", err, stderr.String())
	}

	text := strings.TrimSpace(stdout.String())
	return text, nil
}

// ExtractTextFromFile extracts text from an image file.
func (e *Engine) ExtractTextFromFile(ctx context.Context, imagePath string) (string, error) {
	if !e.available {
		return "", fmt.Errorf("tesseract not installed")
	}

	cmd := exec.CommandContext(ctx, "tesseract",
		imagePath, "stdout",
		"-l", e.Language,
		"--psm", "3",
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("tesseract failed: %w (stderr: %s)", err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// SetLanguage sets the OCR language (e.g., "eng", "deu", "fra").
func (e *Engine) SetLanguage(lang string) {
	e.Language = lang
}
