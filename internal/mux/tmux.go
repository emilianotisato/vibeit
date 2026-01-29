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
	TabTerminal TabType = "term"
	TabLazygit  TabType = "lazygit"
	TabClaude   TabType = "claude"
	TabCodex    TabType = "codex"
	TabNeovim   TabType = "nvim"
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
	case TabNeovim:
		return "nvim"
	default:
		return ""
	}
}

// QueryTabNames returns all tmux window names for a session
func QueryTabNames(sessionName string) ([]string, error) {
	cmd := exec.Command("tmux", "list-windows", "-t", sessionName, "-F", "#W")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var tabs []string
	for _, line := range strings.Split(string(output), "\n") {
		name := strings.TrimSpace(line)
		if name != "" {
			tabs = append(tabs, name)
		}
	}
	return tabs, nil
}

// FilterTabsByPrefix returns tabs that match a prefix or prefix-N
func FilterTabsByPrefix(tabs []string, prefix string) []string {
	var filtered []string
	for _, tab := range tabs {
		if tab == prefix || strings.HasPrefix(tab, prefix+"-") {
			filtered = append(filtered, tab)
		}
	}
	return filtered
}

// NextTabName generates the next tab name for a multi-instance type
// e.g., if tabs has "claude-1", "claude-2", returns "claude-3"
func NextTabName(tabs []string, tabType TabType) string {
	prefix := string(tabType)
	existing := FilterTabsByPrefix(tabs, prefix)

	if len(existing) == 0 {
		return fmt.Sprintf("%s-1", prefix)
	}

	maxNum := 0
	for _, tab := range existing {
		if tab == prefix {
			if maxNum < 1 {
				maxNum = 1
			}
			continue
		}
		parts := strings.Split(tab, "-")
		if len(parts) >= 2 {
			var num int
			fmt.Sscanf(parts[len(parts)-1], "%d", &num)
			if num > maxNum {
				maxNum = num
			}
		}
	}
	return fmt.Sprintf("%s-%d", prefix, maxNum+1)
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

// GoToTabCmd returns a command that goes to a specific tab and attaches
func GoToTabCmd(sessionName, workDir, tabName string) *exec.Cmd {
	script := fmt.Sprintf(
		`if tmux has-session -t %q 2>/dev/null; then `+
			`tmux select-window -t %q 2>/dev/null; `+
			`tmux attach -t %q; `+
			`else `+
			`tmux new-session -s %q -n %q -c %q; `+
			`fi`,
		sessionName,
		fmt.Sprintf("%s:%s", sessionName, tabName),
		sessionName,
		sessionName, tabName, workDir,
	)
	cmd := exec.Command("sh", "-c", script)
	cmd.Dir = workDir
	return cmd
}

// NewTabCmd creates a new tab with a command and attaches
func NewTabCmd(sessionName, workDir, tabName string, tabType TabType) *exec.Cmd {
	command := TabCommand(tabType)
	var cmdPart string
	if command != "" {
		cmdPart = fmt.Sprintf(" %q", command)
	}

	script := fmt.Sprintf(
		`if tmux has-session -t %q 2>/dev/null; then `+
			`tmux new-window -t %q -n %q -c %q%s 2>/dev/null; `+
			`tmux select-window -t %q 2>/dev/null; `+
			`tmux attach -t %q; `+
			`else `+
			`tmux new-session -d -s %q -n %q -c %q%s; `+
			`tmux attach -t %q; `+
			`fi`,
		sessionName,
		sessionName, tabName, workDir, cmdPart,
		fmt.Sprintf("%s:%s", sessionName, tabName),
		sessionName,
		sessionName, tabName, workDir, cmdPart,
		sessionName,
	)
	cmd := exec.Command("sh", "-c", script)
	cmd.Dir = workDir
	return cmd
}

// GoToOrCreateSingleTabCmd goes to a single-instance tab, creating if it doesn't exist
func GoToOrCreateSingleTabCmd(sessionName, workDir string, tabType TabType) *exec.Cmd {
	tabName := string(tabType)
	command := TabCommand(tabType)
	var cmdPart string
	if command != "" {
		cmdPart = fmt.Sprintf(" %q", command)
	}

	script := fmt.Sprintf(
		`if tmux has-session -t %q 2>/dev/null; then `+
			`tmux select-window -t %q 2>/dev/null || tmux new-window -t %q -n %q -c %q%s; `+
			`tmux select-window -t %q 2>/dev/null; `+
			`tmux attach -t %q; `+
			`else `+
			`tmux new-session -s %q -n %q -c %q%s; `+
			`fi`,
		sessionName,
		fmt.Sprintf("%s:%s", sessionName, tabName),
		sessionName, tabName, workDir, cmdPart,
		fmt.Sprintf("%s:%s", sessionName, tabName),
		sessionName,
		sessionName, tabName, workDir, cmdPart,
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
