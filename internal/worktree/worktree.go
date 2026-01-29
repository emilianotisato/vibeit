package worktree

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Config represents .vibe/wt.json
type Config struct {
	Before []string `json:"before"`
	Copy   []string `json:"copy"`
	After  []string `json:"after"`
}

const defaultConfig = `{
    "before": [],
    "copy": [
        ".env",
        "node_modules",
        "vendor"
    ],
    "after": []
}
`

// Create creates a new worktree with the given branch name
func Create(repoPath, branchName, baseBranch string) (string, error) {
	projectName := filepath.Base(repoPath)
	parentDir := filepath.Dir(repoPath)
	worktreePath := filepath.Join(parentDir, projectName+"-"+branchName)

	// Check if directory already exists
	if _, err := os.Stat(worktreePath); err == nil {
		return "", fmt.Errorf("directory already exists: %s", worktreePath)
	}

	// Create the worktree
	args := []string{"worktree", "add", "-b", branchName, worktreePath}
	if baseBranch != "" {
		args = append(args, baseBranch)
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git worktree add failed: %s: %w", string(output), err)
	}

	return worktreePath, nil
}

// ConfigPath returns the path to .vibe/wt.json in the main repo.
func ConfigPath(repoPath string) string {
	return filepath.Join(repoPath, ".vibe", "wt.json")
}

// ConfigExists reports whether .vibe/wt.json exists.
func ConfigExists(repoPath string) bool {
	_, err := os.Stat(ConfigPath(repoPath))
	return err == nil
}

// EnsureConfig creates .vibe/wt.json with a default template when missing.
func EnsureConfig(repoPath string) (string, bool, error) {
	path := ConfigPath(repoPath)
	if _, err := os.Stat(path); err == nil {
		return path, false, nil
	} else if !os.IsNotExist(err) {
		return "", false, err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", false, err
	}
	if err := os.WriteFile(path, []byte(defaultConfig), 0644); err != nil {
		return "", false, err
	}

	return path, true, nil
}

// Init initializes a worktree by running .vibe/wt.json config
func Init(mainRepoPath, worktreePath string) error {
	configPath := ConfigPath(mainRepoPath)

	// Check if config exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// No config file, nothing to do
		return nil
	}

	// Read config
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read wt.json: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse wt.json: %w", err)
	}

	// Run before commands
	for _, cmdStr := range config.Before {
		if err := runCommand(worktreePath, cmdStr); err != nil {
			return fmt.Errorf("before command failed '%s': %w", cmdStr, err)
		}
	}

	// Copy files
	for _, item := range config.Copy {
		src := filepath.Join(mainRepoPath, item)
		dst := filepath.Join(worktreePath, item)

		if err := copyPath(src, dst); err != nil {
			// Log warning but don't fail
			fmt.Fprintf(os.Stderr, "Warning: failed to copy %s: %v\n", item, err)
		}
	}

	// Run after commands
	for _, cmdStr := range config.After {
		if err := runCommand(worktreePath, cmdStr); err != nil {
			return fmt.Errorf("after command failed '%s': %w", cmdStr, err)
		}
	}

	return nil
}

// Delete removes a worktree, its branch, and optionally the notes file
func Delete(repoPath, worktreePath, branchName string, deleteNotes bool) error {
	// Remove the worktree
	cmd := exec.Command("git", "worktree", "remove", worktreePath, "--force")
	cmd.Dir = repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git worktree remove failed: %s: %w", string(output), err)
	}

	// Delete the branch
	cmd = exec.Command("git", "branch", "-D", branchName)
	cmd.Dir = repoPath
	// Ignore error if branch doesn't exist
	cmd.Run()

	// Delete notes file if requested
	if deleteNotes {
		parentDir := filepath.Dir(repoPath)
		projectName := filepath.Base(repoPath)
		notesPath := filepath.Join(parentDir, projectName+"-"+branchName+".md")
		os.Remove(notesPath) // Ignore error if doesn't exist
	}

	return nil
}

// List returns all worktrees for a repo
func List(repoPath string) ([]WorktreeInfo, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return parseWorktreeList(string(output), repoPath)
}

type WorktreeInfo struct {
	Path   string
	Branch string
	IsMain bool
}

func parseWorktreeList(output, repoPath string) ([]WorktreeInfo, error) {
	var worktrees []WorktreeInfo
	var current WorktreeInfo

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if current.Path != "" {
				// Determine if this is the main worktree
				current.IsMain = current.Path == repoPath
				worktrees = append(worktrees, current)
				current = WorktreeInfo{}
			}
			continue
		}

		if strings.HasPrefix(line, "worktree ") {
			current.Path = strings.TrimPrefix(line, "worktree ")
		} else if strings.HasPrefix(line, "branch refs/heads/") {
			current.Branch = strings.TrimPrefix(line, "branch refs/heads/")
		}
	}

	// Don't forget the last one
	if current.Path != "" {
		current.IsMain = current.Path == repoPath
		worktrees = append(worktrees, current)
	}

	return worktrees, nil
}

func runCommand(dir, cmdStr string) error {
	cmd := exec.Command("sh", "-c", cmdStr)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func copyPath(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Create parent directory
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	if info.IsDir() {
		return copyDir(src, dst)
	}
	return copyFile(src, dst)
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, data, info.Mode())
}

func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}
