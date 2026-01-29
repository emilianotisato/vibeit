package mux

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// SessionName generates a tmux session name for a workspace
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

// SessionExists checks if a tmux session exists
func SessionExists(sessionName string) bool {
	cmd := exec.Command("tmux", "has-session", "-t", sessionName)
	return cmd.Run() == nil
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
	script := fmt.Sprintf(
		`tmux has-session -t %q 2>/dev/null && tmux attach -t %q || tmux new-session -s %q -c %q`,
		sessionName, sessionName, sessionName, workDir,
	)
	cmd := exec.Command("sh", "-c", script)
	cmd.Dir = workDir
	return cmd
}

// OpenWithCommand creates/attaches to session with a specific command running
func OpenWithCommand(sessionName, workDir string, tabType TabType) *exec.Cmd {
	command := TabCommand(tabType)
	if command == "" {
		return AttachOrCreateCmd(sessionName, workDir)
	}

	tabName := string(tabType)
	windowTarget := fmt.Sprintf("%s:%s", sessionName, tabName)
	script := fmt.Sprintf(
		`if tmux has-session -t %q 2>/dev/null; then `+
			`tmux new-window -t %q -n %q -c %q %q 2>/dev/null; `+
			`tmux select-window -t %q 2>/dev/null; `+
			`tmux attach -t %q; `+
			`else `+
			`tmux new-session -d -s %q -n %q -c %q %q; `+
			`tmux attach -t %q; `+
			`fi`,
		sessionName,
		sessionName, tabName, workDir, command,
		windowTarget,
		sessionName,
		sessionName, tabName, workDir, command,
		sessionName,
	)
	cmd := exec.Command("sh", "-c", script)
	cmd.Dir = workDir
	return cmd
}

// DeleteSession deletes a tmux session
func DeleteSession(sessionName string) error {
	cmd := exec.Command("tmux", "kill-session", "-t", sessionName)
	return cmd.Run()
}

// KillSession kills a tmux session
func KillSession(sessionName string) error {
	cmd := exec.Command("tmux", "kill-session", "-t", sessionName)
	return cmd.Run()
}

// IsTmuxInstalled checks if tmux is available
func IsTmuxInstalled() bool {
	_, err := exec.LookPath("tmux")
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
