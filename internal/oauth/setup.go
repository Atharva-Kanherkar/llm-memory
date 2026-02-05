package oauth

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// SetupWizard handles CLI-based OAuth setup.
type SetupWizard struct {
	reader *bufio.Reader
}

// NewSetupWizard creates a new setup wizard.
func NewSetupWizard() *SetupWizard {
	return &SetupWizard{
		reader: bufio.NewReader(os.Stdin),
	}
}

// HasGcloud checks if gcloud CLI is installed.
func HasGcloud() bool {
	_, err := exec.LookPath("gcloud")
	return err == nil
}

// SetupGoogle guides through Google OAuth setup via CLI.
func (w *SetupWizard) SetupGoogle(ctx context.Context) error {
	fmt.Println("\n=== Google OAuth Setup ===\n")

	if !HasGcloud() {
		fmt.Println("gcloud CLI not found. Install it from: https://cloud.google.com/sdk/docs/install")
		fmt.Println("\nOr set environment variables manually:")
		fmt.Println("  export GOOGLE_CLIENT_ID=\"your-client-id\"")
		fmt.Println("  export GOOGLE_CLIENT_SECRET=\"your-client-secret\"")
		return fmt.Errorf("gcloud not installed")
	}

	// Check if already logged in
	fmt.Println("Checking gcloud authentication...")
	checkCmd := exec.CommandContext(ctx, "gcloud", "auth", "list", "--filter=status:ACTIVE", "--format=value(account)")
	output, err := checkCmd.Output()
	if err != nil || len(strings.TrimSpace(string(output))) == 0 {
		fmt.Println("Not logged into gcloud. Running 'gcloud auth login'...")
		loginCmd := exec.CommandContext(ctx, "gcloud", "auth", "login")
		loginCmd.Stdin = os.Stdin
		loginCmd.Stdout = os.Stdout
		loginCmd.Stderr = os.Stderr
		if err := loginCmd.Run(); err != nil {
			return fmt.Errorf("gcloud auth failed: %w", err)
		}
	} else {
		fmt.Printf("Logged in as: %s\n", strings.TrimSpace(string(output)))
	}

	// Get or create project
	projectID, err := w.getOrCreateProject(ctx)
	if err != nil {
		return err
	}

	// Enable APIs
	fmt.Println("\nEnabling Gmail and Calendar APIs...")
	apis := []string{"gmail.googleapis.com", "calendar-json.googleapis.com"}
	for _, api := range apis {
		enableCmd := exec.CommandContext(ctx, "gcloud", "services", "enable", api, "--project", projectID)
		if err := enableCmd.Run(); err != nil {
			fmt.Printf("Warning: Could not enable %s: %v\n", api, err)
		} else {
			fmt.Printf("  Enabled: %s\n", api)
		}
	}

	// Create OAuth credentials
	fmt.Println("\nCreating OAuth credentials...")
	fmt.Println("\nUnfortunately, gcloud CLI cannot create OAuth client IDs directly.")
	fmt.Println("You have two options:\n")
	fmt.Println("1. Use Mnemosyne's built-in credentials (recommended, works immediately)")
	fmt.Println("2. Create your own at: https://console.cloud.google.com/apis/credentials")
	fmt.Println("")

	fmt.Print("Use built-in credentials? [Y/n]: ")
	response, _ := w.reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response == "" || response == "y" || response == "yes" {
		fmt.Println("\nUsing built-in credentials.")
		fmt.Println("You can now use /connect gmail and /connect calendar")
		return nil
	}

	fmt.Println("\nTo create your own credentials:")
	fmt.Println("1. Go to: https://console.cloud.google.com/apis/credentials")
	fmt.Printf("2. Select project: %s\n", projectID)
	fmt.Println("3. Click 'Create Credentials' > 'OAuth client ID'")
	fmt.Println("4. Choose 'Desktop app'")
	fmt.Println("5. Copy the Client ID and Secret")
	fmt.Println("6. Set environment variables:")
	fmt.Println("   export GOOGLE_CLIENT_ID=\"your-client-id\"")
	fmt.Println("   export GOOGLE_CLIENT_SECRET=\"your-client-secret\"")

	return nil
}

// getOrCreateProject gets or creates a GCP project.
func (w *SetupWizard) getOrCreateProject(ctx context.Context) (string, error) {
	// List existing projects
	fmt.Println("\nChecking existing projects...")
	listCmd := exec.CommandContext(ctx, "gcloud", "projects", "list", "--format=value(projectId)", "--limit=10")
	output, err := listCmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to list projects: %w", err)
	}

	projects := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(projects) > 0 && projects[0] != "" {
		fmt.Println("Existing projects:")
		for i, p := range projects {
			fmt.Printf("  %d. %s\n", i+1, p)
		}
		fmt.Println("")
		fmt.Print("Enter project number to use, or 'new' to create: ")
		response, _ := w.reader.ReadString('\n')
		response = strings.TrimSpace(response)

		if response != "new" {
			var idx int
			if _, err := fmt.Sscanf(response, "%d", &idx); err == nil && idx > 0 && idx <= len(projects) {
				projectID := projects[idx-1]
				// Set as default
				exec.CommandContext(ctx, "gcloud", "config", "set", "project", projectID).Run()
				return projectID, nil
			}
			// If invalid input, use first project
			if len(projects) > 0 {
				exec.CommandContext(ctx, "gcloud", "config", "set", "project", projects[0]).Run()
				return projects[0], nil
			}
		}
	}

	// Create new project
	projectID := fmt.Sprintf("mnemosyne-%d", time.Now().Unix()%100000)
	fmt.Printf("Creating project: %s\n", projectID)

	createCmd := exec.CommandContext(ctx, "gcloud", "projects", "create", projectID, "--name=Mnemosyne")
	createCmd.Stdout = os.Stdout
	createCmd.Stderr = os.Stderr
	if err := createCmd.Run(); err != nil {
		return "", fmt.Errorf("failed to create project: %w", err)
	}

	// Set as default
	exec.CommandContext(ctx, "gcloud", "config", "set", "project", projectID).Run()
	return projectID, nil
}

// SetupSlack guides through Slack OAuth setup.
func (w *SetupWizard) SetupSlack(ctx context.Context) error {
	fmt.Println("\n=== Slack OAuth Setup ===\n")
	fmt.Println("Slack requires creating an app through their web interface.")
	fmt.Println("\nYou have two options:\n")
	fmt.Println("1. Use Mnemosyne's built-in credentials (works immediately)")
	fmt.Println("2. Create your own Slack app")
	fmt.Println("")

	fmt.Print("Use built-in credentials? [Y/n]: ")
	response, _ := w.reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response == "" || response == "y" || response == "yes" {
		fmt.Println("\nUsing built-in credentials.")
		fmt.Println("You can now use /connect slack")
		return nil
	}

	fmt.Println("\nTo create your own Slack app:")
	fmt.Println("1. Go to: https://api.slack.com/apps")
	fmt.Println("2. Click 'Create New App' > 'From scratch'")
	fmt.Println("3. Add OAuth scopes: channels:history, channels:read, users:read")
	fmt.Println("4. Add redirect URL: http://localhost:8087/callback")
	fmt.Println("5. Install to workspace")
	fmt.Println("6. Set environment variables:")
	fmt.Println("   export SLACK_CLIENT_ID=\"your-client-id\"")
	fmt.Println("   export SLACK_CLIENT_SECRET=\"your-client-secret\"")

	return nil
}

// QuickSetup does automatic setup with built-in credentials.
func (w *SetupWizard) QuickSetup() {
	fmt.Println("\n=== Quick Setup ===\n")
	fmt.Println("Mnemosyne includes built-in OAuth credentials for easy setup.")
	fmt.Println("No configuration needed - just use these commands:\n")
	fmt.Println("  /connect gmail     - Connect your Gmail")
	fmt.Println("  /connect calendar  - Connect Google Calendar")
	fmt.Println("  /connect slack     - Connect Slack")
	fmt.Println("\nThe first time you connect, a browser window will open")
	fmt.Println("for you to authorize access to your account.")
	fmt.Println("\nYour OAuth tokens are encrypted with AES-256-GCM and stored locally.")
}
