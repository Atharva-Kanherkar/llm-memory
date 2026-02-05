// Package git provides git repository state capture.
//
// It captures:
// - Current repository path
// - Current branch
// - Recent commits (last 5)
// - Uncommitted changes (staged and unstaged)
// - Current file being edited (from window context)
//
// This is cross-platform since we just shell out to git.
package git

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Atharva-Kanherkar/mnemosyne/internal/capture"
)

// Capturer captures git repository state.
type Capturer struct {
	// WorkingDir is the directory to check for git repos.
	// If empty, uses current working directory.
	WorkingDir string

	// MaxCommits is how many recent commits to capture.
	MaxCommits int
}

// New creates a new git Capturer.
func New() *Capturer {
	return &Capturer{
		WorkingDir: "", // Will use cwd
		MaxCommits: 5,
	}
}

// Name returns the capturer identifier.
func (c *Capturer) Name() string {
	return "git"
}

// Available checks if git is installed.
func (c *Capturer) Available() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// Capture gets the current git state.
func (c *Capturer) Capture(ctx context.Context) (*capture.Result, error) {
	return c.CaptureInDir(ctx, c.WorkingDir)
}

// CaptureInDir captures git state for a specific directory.
// This is useful when you know which directory to check (e.g., from window title).
func (c *Capturer) CaptureInDir(ctx context.Context, dir string) (*capture.Result, error) {
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}
	}

	// Find the git root (might be in a subdirectory)
	gitRoot, err := c.findGitRoot(ctx, dir)
	if err != nil {
		// Not in a git repo
		result := capture.NewResult("git")
		result.SetMetadata("in_repo", "false")
		result.SetMetadata("checked_dir", dir)
		return result, nil
	}

	result := capture.NewResult("git")
	result.SetMetadata("in_repo", "true")
	result.SetMetadata("repo_root", gitRoot)
	result.SetMetadata("repo_name", filepath.Base(gitRoot))

	// Get branch
	branch, err := c.getBranch(ctx, gitRoot)
	if err == nil {
		result.SetMetadata("branch", branch)
	}

	// Get current commit
	commit, err := c.getCurrentCommit(ctx, gitRoot)
	if err == nil {
		result.SetMetadata("commit", commit)
	}

	// Get recent commits
	commits, err := c.getRecentCommits(ctx, gitRoot, c.MaxCommits)
	if err == nil {
		result.SetMetadata("recent_commits", strings.Join(commits, "\n"))
	}

	// Get status (modified/staged files)
	status, err := c.getStatus(ctx, gitRoot)
	if err == nil {
		result.SetMetadata("status", status)
		result.SetMetadata("has_changes", fmt.Sprintf("%t", status != ""))
	}

	// Get diff stats
	diffStats, err := c.getDiffStats(ctx, gitRoot)
	if err == nil {
		result.SetMetadata("diff_stats", diffStats)
	}

	// Build summary text
	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("Repo: %s\n", filepath.Base(gitRoot)))
	summary.WriteString(fmt.Sprintf("Branch: %s\n", branch))
	if status != "" {
		summary.WriteString(fmt.Sprintf("Changes:\n%s\n", status))
	}
	result.TextData = summary.String()

	return result, nil
}

// findGitRoot finds the root of the git repository containing dir.
func (c *Capturer) findGitRoot(ctx context.Context, dir string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository")
	}

	return strings.TrimSpace(string(output)), nil
}

// getBranch returns the current branch name.
func (c *Capturer) getBranch(ctx context.Context, gitRoot string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "branch", "--show-current")
	cmd.Dir = gitRoot

	output, err := cmd.Output()
	if err != nil {
		// Might be in detached HEAD state
		cmd = exec.CommandContext(ctx, "git", "rev-parse", "--short", "HEAD")
		cmd.Dir = gitRoot
		output, err = cmd.Output()
		if err != nil {
			return "", err
		}
		return "detached:" + strings.TrimSpace(string(output)), nil
	}

	return strings.TrimSpace(string(output)), nil
}

// getCurrentCommit returns the current commit hash.
func (c *Capturer) getCurrentCommit(ctx context.Context, gitRoot string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--short", "HEAD")
	cmd.Dir = gitRoot

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// getRecentCommits returns the last n commits (one-line format).
func (c *Capturer) getRecentCommits(ctx context.Context, gitRoot string, n int) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "log",
		fmt.Sprintf("-%d", n),
		"--oneline",
		"--no-decorate")
	cmd.Dir = gitRoot

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return []string{}, nil
	}
	return lines, nil
}

// getStatus returns git status in short format.
func (c *Capturer) getStatus(ctx context.Context, gitRoot string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "status", "--short")
	cmd.Dir = gitRoot

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// getDiffStats returns stats about uncommitted changes.
func (c *Capturer) getDiffStats(ctx context.Context, gitRoot string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "diff", "--stat", "--no-color")
	cmd.Dir = gitRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", err
	}

	return strings.TrimSpace(stdout.String()), nil
}

// GetRemoteURL returns the remote origin URL.
func (c *Capturer) GetRemoteURL(ctx context.Context, gitRoot string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "remote", "get-url", "origin")
	cmd.Dir = gitRoot

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}
