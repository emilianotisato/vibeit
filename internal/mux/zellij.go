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

	for _, line := range strings.Split(string(output), "\n") {
		name := strings.Split(strings.TrimSpace(line), " ")[0]
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

// AttachOrCreateCmd returns a command that attaches to session, creating if needed
func AttachOrCreateCmd(sessionName, workDir string) *exec.Cmd {
	cmd := exec.Command("zellij", "attach", "--create", sessionName)
	cmd.Dir = workDir
	return cmd
}

// OpenWithCommand creates/attaches to session with a specific command running
func OpenWithCommand(sessionName, workDir string, tabType TabType) *exec.Cmd {
	command := TabCommand(tabType)
	tabName := string(tabType)

	if command != "" {
		// Create layout file
		layoutContent := fmt.Sprintf("session_name \"%s\"\nlayout {\n    tab name=\"%s\" {\n        pane command=\"%s\"\n    }\n}\n", sessionName, tabName, command)
		tmpFile, err := os.CreateTemp("", "vibeit-layout-*.kdl")
		if err == nil {
			tmpFile.WriteString(layoutContent)
			tmpFile.Close()

			// Delete any existing session (active or dead), then create fresh
			script := fmt.Sprintf(
				`zellij delete-session --force "%s" 2>/dev/null; zellij --layout "%s"`,
				sessionName, tmpFile.Name(),
			)
			cmd := exec.Command("sh", "-c", script)
			cmd.Dir = workDir
			return cmd
		}
	}

	// No command or temp file failed - just attach/create session
	cmd := exec.Command("zellij", "attach", "--create", sessionName)
	cmd.Dir = workDir
	return cmd
}

// DeleteSession deletes a zellij session (works for both active and dead)
func DeleteSession(sessionName string) error {
	cmd := exec.Command("zellij", "delete-session", "--force", sessionName)
	return cmd.Run()
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

// OpenNotes opens a notes file in neovim
func OpenNotesCmd(notesPath, workDir string) *exec.Cmd {
	// Ensure the notes file exists
	dir := filepath.Dir(notesPath)
	os.MkdirAll(dir, 0755)

	// Create empty file if doesn't exist
	if _, err := os.Stat(notesPath); os.IsNotExist(err) {
		os.WriteFile(notesPath, []byte("# Notes\n\n"), 0644)
	}

	cmd := exec.Command("nvim", notesPath)
	cmd.Dir = workDir
	return cmd
}
