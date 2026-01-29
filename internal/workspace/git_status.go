package workspace

import (
	"os/exec"
	"strings"
)

// UpdateGitStatus refreshes git-related fields for a workspace.
func UpdateGitStatus(ws Workspace) Workspace {
	if branch := gitBranch(ws.Path); branch != "" {
		ws.Branch = branch
	}
	if dirty, ok := gitDirty(ws.Path); ok {
		ws.IsDirty = dirty
	}
	if ahead, behind, ok := gitAheadBehind(ws.Path); ok {
		ws.Ahead = ahead
		ws.Behind = behind
	}
	if stashCount, ok := gitStashCount(ws.Path); ok {
		ws.StashCount = stashCount
	}
	if commits, ok := gitRecentCommits(ws.Path, 5); ok {
		ws.RecentCommits = commits
	}
	return ws
}

func gitBranch(path string) string {
	out, err := runGitCommand(path, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

func gitDirty(path string) (bool, bool) {
	out, err := runGitCommand(path, "status", "--porcelain")
	if err != nil {
		return false, false
	}
	return strings.TrimSpace(out) != "", true
}

func gitAheadBehind(path string) (int, int, bool) {
	upstream, err := runGitCommand(path, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}")
	if err != nil || strings.TrimSpace(upstream) == "" {
		return 0, 0, false
	}

	out, err := runGitCommand(path, "rev-list", "--left-right", "--count", strings.TrimSpace(upstream)+"...HEAD")
	if err != nil {
		return 0, 0, false
	}

	parts := strings.Fields(strings.TrimSpace(out))
	if len(parts) != 2 {
		return 0, 0, false
	}

	behind, ahead := parseInt(parts[0]), parseInt(parts[1])
	return ahead, behind, true
}

func gitStashCount(path string) (int, bool) {
	out, err := runGitCommand(path, "stash", "list")
	if err != nil {
		return 0, false
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) == 1 && strings.TrimSpace(lines[0]) == "" {
		return 0, true
	}
	return len(lines), true
}

func gitRecentCommits(path string, limit int) ([]string, bool) {
	out, err := runGitCommand(path, "log", "-n", intToString(limit), "--pretty=format:%h %s")
	if err != nil {
		return nil, false
	}
	var commits []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			commits = append(commits, line)
		}
	}
	return commits, true
}

func runGitCommand(path string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", path}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func parseInt(value string) int {
	n := 0
	for _, r := range value {
		if r < '0' || r > '9' {
			break
		}
		n = n*10 + int(r-'0')
	}
	return n
}

func intToString(value int) string {
	if value == 0 {
		return "0"
	}
	sign := ""
	if value < 0 {
		sign = "-"
		value = -value
	}
	var digits []byte
	for value > 0 {
		digits = append(digits, byte('0'+value%10))
		value /= 10
	}
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}
	return sign + string(digits)
}
