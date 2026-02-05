package mux

import (
	"fmt"
	"os/exec"
	"strings"
)

const overviewWindowName = "__vibeit_overview"
const overviewSleepCmd = "sleep 1000000"

// ToggleOverview shows or hides the overview grid for managed windows.
func ToggleOverview() error {
	session, err := tmuxOutput("display-message", "-p", "#S")
	if err != nil || session == "" {
		return fmt.Errorf("tmux session not found")
	}

	active := tmuxOption(session, "@vibeit_overview_active")
	if active == "1" {
		return hideOverview(session)
	}
	return showOverview(session)
}

func showOverview(session string) error {
	lastWin, err := tmuxOutput("display-message", "-p", "#{window_id}")
	if err != nil || lastWin == "" {
		return fmt.Errorf("tmux window not found")
	}

	info, err := tmuxOutput(
		"new-window",
		"-P",
		"-F",
		"#{window_id}:#{pane_id}",
		"-n",
		overviewWindowName,
		"-t",
		session,
		"-d",
		overviewSleepCmd,
	)
	if err != nil || info == "" {
		return fmt.Errorf("failed to create overview window")
	}

	parts := strings.SplitN(info, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("failed to parse overview window info")
	}
	overviewWin := parts[0]
	seedPane := parts[1]

	windows, err := tmuxOutput(
		"list-windows",
		"-t",
		session,
		"-F",
		"#{window_id}\t#{window_name}\t#{window_panes}",
	)
	if err != nil {
		_ = tmuxRun("kill-window", "-t", overviewWin)
		return err
	}

	moved := 0
	placeholder := seedPane
	for _, line := range strings.Split(windows, "\n") {
		if line == "" {
			continue
		}
		fields := strings.SplitN(line, "\t", 3)
		if len(fields) != 3 {
			continue
		}
		winID := fields[0]
		winName := fields[1]
		paneCount := fields[2]
		if winID == overviewWin {
			continue
		}
		if paneCount != "1" || !isManagedWindowName(winName) {
			continue
		}

		paneID, err := tmuxOutput("list-panes", "-t", winID, "-F", "#{pane_id}")
		if err != nil || paneID == "" {
			continue
		}

		if moved > 0 {
			placeholder, err = tmuxOutput(
				"split-window",
				"-t",
				overviewWin,
				"-d",
				"-P",
				"-F",
				"#{pane_id}",
				overviewSleepCmd,
			)
			if err != nil || placeholder == "" {
				continue
			}
		}

		_ = tmuxRun("set-option", "-p", "-t", paneID, "@vibeit_overview_placeholder", placeholder)
		_ = tmuxRun("set-option", "-p", "-t", paneID, "@vibeit_overview_orig_window", winID)
		_ = tmuxRun("swap-pane", "-s", paneID, "-t", placeholder)
		moved++
	}

	if moved == 0 {
		_ = tmuxRun("kill-window", "-t", overviewWin)
		return nil
	}

	_ = tmuxRun("select-layout", "-t", overviewWin, "tiled")
	_ = tmuxRun("select-window", "-t", overviewWin)

	_ = tmuxRun("set-option", "-t", session, "@vibeit_overview_active", "1")
	_ = tmuxRun("set-option", "-t", session, "@vibeit_overview_window", overviewWin)
	_ = tmuxRun("set-option", "-t", session, "@vibeit_overview_last_window", lastWin)
	return nil
}

func hideOverview(session string) error {
	overviewWin := tmuxOption(session, "@vibeit_overview_window")
	lastWin := tmuxOption(session, "@vibeit_overview_last_window")

	if overviewWin != "" {
		panes, err := tmuxOutput("list-panes", "-t", overviewWin, "-F", "#{pane_id}")
		if err == nil {
			for _, paneID := range strings.Split(panes, "\n") {
				if paneID == "" {
					continue
				}
				placeholder := tmuxPaneOption(paneID, "@vibeit_overview_placeholder")
				origWin := tmuxPaneOption(paneID, "@vibeit_overview_orig_window")
				if placeholder != "" && tmuxPaneExists(placeholder) {
					_ = tmuxRun("swap-pane", "-s", paneID, "-t", placeholder)
				} else if origWin != "" && tmuxWindowExists(origWin) {
					_ = tmuxRun("move-pane", "-s", paneID, "-t", origWin)
				} else if lastWin != "" {
					_ = tmuxRun("move-pane", "-s", paneID, "-t", lastWin)
				}
				_ = tmuxRun("set-option", "-p", "-t", paneID, "-u", "@vibeit_overview_placeholder")
				_ = tmuxRun("set-option", "-p", "-t", paneID, "-u", "@vibeit_overview_orig_window")
			}
		}
		_ = tmuxRun("kill-window", "-t", overviewWin)
	}

	if lastWin != "" {
		_ = tmuxRun("select-window", "-t", lastWin)
	}

	_ = tmuxRun("set-option", "-t", session, "-u", "@vibeit_overview_active")
	_ = tmuxRun("set-option", "-t", session, "-u", "@vibeit_overview_window")
	_ = tmuxRun("set-option", "-t", session, "-u", "@vibeit_overview_last_window")
	return nil
}

func isManagedWindowName(name string) bool {
	prefixes := []string{
		string(TabLazygit),
		string(TabClaude),
		string(TabCodex),
		string(TabNeovim),
		string(TabTerminal),
	}
	for _, prefix := range prefixes {
		if name == prefix || strings.HasPrefix(name, prefix+"-") {
			return true
		}
	}
	return false
}

func tmuxOption(target, option string) string {
	out, err := tmuxOutput("show", "-t", target, "-v", option)
	if err != nil {
		return ""
	}
	return out
}

func tmuxPaneOption(paneID, option string) string {
	out, err := tmuxOutput("show", "-p", "-t", paneID, "-v", option)
	if err != nil {
		return ""
	}
	return out
}

func tmuxPaneExists(paneID string) bool {
	_, err := exec.Command("tmux", "display-message", "-p", "-t", paneID, "#{pane_id}").Output()
	return err == nil
}

func tmuxWindowExists(winID string) bool {
	_, err := exec.Command("tmux", "display-message", "-p", "-t", winID, "#{window_id}").Output()
	return err == nil
}

func tmuxOutput(args ...string) (string, error) {
	cmd := exec.Command("tmux", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func tmuxRun(args ...string) error {
	cmd := exec.Command("tmux", args...)
	return cmd.Run()
}
