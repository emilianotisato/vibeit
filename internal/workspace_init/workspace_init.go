package workspace_init

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
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

const maxWorkspaces = 9

// Create creates a new workspace by cloning the main repo into a sibling {projectName}-wt-N directory
func Create(mainRepoPath, branchName, baseBranch string) (string, error) {
	parentDir := filepath.Dir(mainRepoPath)
	projectName := filepath.Base(mainRepoPath)

	// Find next available {projectName}-wt-N slot (1-9)
	slot, err := findNextSlot(parentDir, projectName)
	if err != nil {
		return "", err
	}

	workspacePath := filepath.Join(parentDir, fmt.Sprintf("%s-wt-%d", projectName, slot))

	// Get the origin remote URL from main repo
	originURL, err := getOriginURL(mainRepoPath)
	if err != nil {
		return "", fmt.Errorf("failed to get origin URL: %w", err)
	}

	// Clone the main repo
	cmd := exec.Command("git", "clone", mainRepoPath, workspacePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git clone failed: %s: %w", string(output), err)
	}

	// Update origin to point to the real remote (not the local clone)
	cmd = exec.Command("git", "remote", "set-url", "origin", originURL)
	cmd.Dir = workspacePath
	if output, err := cmd.CombinedOutput(); err != nil {
		// Clean up on failure
		os.RemoveAll(workspacePath)
		return "", fmt.Errorf("failed to set origin URL: %s: %w", string(output), err)
	}

	// Create and checkout the new branch from baseBranch
	var checkoutArgs []string
	if baseBranch != "" {
		checkoutArgs = []string{"checkout", "-b", branchName, baseBranch}
	} else {
		checkoutArgs = []string{"checkout", "-b", branchName}
	}
	cmd = exec.Command("git", checkoutArgs...)
	cmd.Dir = workspacePath
	if output, err := cmd.CombinedOutput(); err != nil {
		// Clean up on failure
		os.RemoveAll(workspacePath)
		return "", fmt.Errorf("failed to create branch: %s: %w", string(output), err)
	}

	return workspacePath, nil
}

// findNextSlot finds the next available {projectName}-wt-N slot (1-9) in the parent directory
func findNextSlot(parentDir, projectName string) (int, error) {
	used := make(map[int]bool)

	entries, err := os.ReadDir(parentDir)
	if err != nil {
		return 0, fmt.Errorf("failed to read directory: %w", err)
	}

	// Pattern: {projectName}-wt-{N}
	wtPattern := regexp.MustCompile(`^` + regexp.QuoteMeta(projectName) + `-wt-(\d+)$`)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		matches := wtPattern.FindStringSubmatch(entry.Name())
		if matches != nil {
			num, _ := strconv.Atoi(matches[1])
			if num >= 1 && num <= maxWorkspaces {
				used[num] = true
			}
		}
	}

	for i := 1; i <= maxWorkspaces; i++ {
		if !used[i] {
			return i, nil
		}
	}

	return 0, fmt.Errorf("maximum number of workspaces (%d) reached", maxWorkspaces)
}

// getOriginURL gets the origin remote URL from a git repo
func getOriginURL(repoPath string) (string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
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

// Init initializes a workspace by running .vibe/wt.json config
func Init(mainRepoPath, workspacePath string) error {
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
		if err := runCommand(workspacePath, cmdStr); err != nil {
			return fmt.Errorf("before command failed '%s': %w", cmdStr, err)
		}
	}

	// Copy files
	for _, item := range config.Copy {
		src := filepath.Join(mainRepoPath, item)
		dst := filepath.Join(workspacePath, item)

		if err := copyPath(src, dst); err != nil {
			// Log warning but don't fail
			fmt.Fprintf(os.Stderr, "Warning: failed to copy %s: %v\n", item, err)
		}
	}

	// Run after commands
	for _, cmdStr := range config.After {
		if err := runCommand(workspacePath, cmdStr); err != nil {
			return fmt.Errorf("after command failed '%s': %w", cmdStr, err)
		}
	}

	return nil
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
