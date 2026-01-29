package mux

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// SessionName generates a zellij session name for a workspace
func SessionName(projectName, workspaceName string) string {
	return fmt.Sprintf("vibeit-%s-%s", sanitize(projectName), sanitize(workspaceName))
}

// sanitize removes characters that might cause issues in session names
func sanitize(s string) string {
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, ".", "-")
	return s
}

// SessionExists checks if a zellij session exists
func SessionExists(sessionName string) bool {
	cmd := exec.Command("zellij", "list-sessions")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	sessions := strings.Split(string(output), "\n")
	for _, s := range sessions {
		// Session list format: "session-name (current)" or just "session-name"
		name := strings.Split(strings.TrimSpace(s), " ")[0]
		if name == sessionName {
			return true
		}
	}
	return false
}

// TabType represents the type of tab to create
type TabType string

const (
	TabTerminal TabType = "terminal"
	TabLazygit  TabType = "lazygit"
	TabClaude   TabType = "claude"
	TabCodex    TabType = "codex"
	TabNotes    TabType = "notes"
)

// TabCommand returns the command to run for a tab type
func TabCommand(tabType TabType) string {
	switch tabType {
	case TabLazygit:
		return "lazygit"
	case TabClaude:
		return "claude"
	case TabCodex:
		return "codex"
	default:
		return ""
	}
}

// CreateSessionCmd returns an exec.Cmd to create and attach to a new zellij session
func CreateSessionCmd(sessionName, workDir string, tabType TabType) *exec.Cmd {
	// Build the command to run in the session
	var runCmd string
	tabName := string(tabType)

	switch tabType {
	case TabLazygit:
		runCmd = "lazygit"
	case TabClaude:
		runCmd = "claude"
		tabName = "claude"
	case TabCodex:
		runCmd = "codex"
		tabName = "codex"
	case TabNotes:
		// Notes will be handled separately (open in nvim)
		runCmd = "$SHELL"
	default:
		runCmd = "$SHELL"
	}

	// Create session with initial command
	cmd := exec.Command("zellij",
		"--session", sessionName,
		"--new-session-with-layout", "compact",
	)
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("ZELLIJ_SESSION_NAME=%s", sessionName),
	)

	// If we have a specific command, we'll run it after attaching
	if runCmd != "$SHELL" {
		// Use zellij action to run the command after session starts
		cmd = exec.Command("zellij",
			"--session", sessionName,
		)
		cmd.Dir = workDir
	}

	_ = tabName // Will use for tab naming later

	return cmd
}

// AttachSessionCmd returns an exec.Cmd to attach to an existing session
func AttachSessionCmd(sessionName string) *exec.Cmd {
	return exec.Command("zellij", "attach", sessionName)
}

// AttachOrCreateCmd returns a command that attaches if session exists, creates otherwise
func AttachOrCreateCmd(sessionName, workDir string) *exec.Cmd {
	// zellij attach --create will attach if exists, create if not
	// This is cleaner than checking separately
	cmd := exec.Command("zellij", "attach", "--create", sessionName)
	cmd.Dir = workDir
	return cmd
}

// RunInSession runs a command in an existing zellij session (creates new pane)
func RunInSession(sessionName, workDir, command string) *exec.Cmd {
	// Use zellij action to run command in a new pane
	cmd := exec.Command("zellij",
		"--session", sessionName,
		"action", "new-pane", "--",
		"sh", "-c", fmt.Sprintf("cd %s && %s", workDir, command),
	)
	return cmd
}

// NewTabInSession creates a new tab in an existing session and runs a command
func NewTabInSession(sessionName, workDir, tabName, command string) error {
	// First, write the command to a temp script to ensure proper execution
	if !SessionExists(sessionName) {
		return fmt.Errorf("session %s does not exist", sessionName)
	}

	// Use zellij action to create a new tab
	cmd := exec.Command("zellij",
		"--session", sessionName,
		"action", "new-tab", "--name", tabName,
	)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create tab: %w", err)
	}

	// Run the command in the new tab
	if command != "" && command != "$SHELL" {
		cmd = exec.Command("zellij",
			"--session", sessionName,
			"action", "write-chars", command+"\n",
		)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to run command: %w", err)
		}
	}

	return nil
}

// KillSession kills a zellij session
func KillSession(sessionName string) error {
	cmd := exec.Command("zellij", "kill-session", sessionName)
	return cmd.Run()
}

// IsZellijInstalled checks if zellij is available
func IsZellijInstalled() bool {
	_, err := exec.LookPath("zellij")
	return err == nil
}

// OpenNotes opens a notes file in neovim within a zellij session
func OpenNotesCmd(sessionName, notesPath string) *exec.Cmd {
	// Ensure the notes file exists
	dir := filepath.Dir(notesPath)
	os.MkdirAll(dir, 0755)

	// Create empty file if doesn't exist
	if _, err := os.Stat(notesPath); os.IsNotExist(err) {
		os.WriteFile(notesPath, []byte("# Notes\n\n"), 0644)
	}

	// Open nvim with the notes file
	return exec.Command("zellij",
		"--session", sessionName,
		"action", "new-pane", "--",
		"nvim", notesPath,
	)
}
