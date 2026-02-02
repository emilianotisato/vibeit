package workspace

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/go-git/go-git/v5"
)

// Workspace represents the main repo or a sibling workspace ({projectName}-wt-N)
type Workspace struct {
	Name           string
	Path           string
	Branch         string
	IsSubWorkspace bool
	IsDirty        bool
	Ahead          int
	Behind         int
	StashCount     int
	RecentCommits  []string
	NotesExists    bool
	NotesPreview   []string
}

// Detect finds the main repo and all sibling workspaces from the current directory
func Detect() ([]Workspace, error) {
	projectPath, err := GetProjectPath()
	if err != nil {
		return nil, err
	}

	workspaces := listSiblingWorkspaces(projectPath)

	// Get git status for each workspace
	for i := range workspaces {
		workspaces[i] = UpdateGitStatus(workspaces[i])
	}

	return workspaces, nil
}

// listSiblingWorkspaces scans parent directory for main workspace and {projectName}-wt-N siblings
func listSiblingWorkspaces(mainRepoPath string) []Workspace {
	var workspaces []Workspace

	projectName := filepath.Base(mainRepoPath)
	parentDir := filepath.Dir(mainRepoPath)

	// Add main workspace first
	mainWs := Workspace{
		Path:           mainRepoPath,
		Name:           projectName,
		IsSubWorkspace: false,
	}
	// Get branch for main
	if repo, err := git.PlainOpen(mainRepoPath); err == nil {
		if head, err := repo.Head(); err == nil {
			mainWs.Branch = head.Name().Short()
		}
	}
	workspaces = append(workspaces, mainWs)

	// Scan parent directory for {projectName}-wt-1 through {projectName}-wt-9
	// Pattern: {projectName}-wt-{N}
	wtPattern := regexp.MustCompile(`^` + regexp.QuoteMeta(projectName) + `-wt-(\d+)$`)
	entries, err := os.ReadDir(parentDir)
	if err != nil {
		return workspaces
	}

	// Collect valid workspace directories with their numbers for sorting
	type wtDir struct {
		num  int
		path string
		name string
	}
	var wtDirs []wtDir

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		matches := wtPattern.FindStringSubmatch(entry.Name())
		if matches == nil {
			continue
		}
		num, _ := strconv.Atoi(matches[1])
		if num < 1 || num > 9 {
			continue
		}

		wtPath := filepath.Join(parentDir, entry.Name())
		// Verify it's a git repo
		if _, err := os.Stat(filepath.Join(wtPath, ".git")); err != nil {
			continue
		}

		wtDirs = append(wtDirs, wtDir{num: num, path: wtPath, name: entry.Name()})
	}

	// Sort by number
	for i := 0; i < len(wtDirs); i++ {
		for j := i + 1; j < len(wtDirs); j++ {
			if wtDirs[j].num < wtDirs[i].num {
				wtDirs[i], wtDirs[j] = wtDirs[j], wtDirs[i]
			}
		}
	}

	// Add to workspaces
	for _, wt := range wtDirs {
		ws := Workspace{
			Path:           wt.path,
			Name:           wt.name,
			IsSubWorkspace: true,
		}
		// Get branch
		if repo, err := git.PlainOpen(wt.path); err == nil {
			if head, err := repo.Head(); err == nil {
				ws.Branch = head.Name().Short()
			}
		}
		workspaces = append(workspaces, ws)
	}

	return workspaces
}

// GetProjectName returns the name of the project based on the repo root
func GetProjectName() (string, error) {
	path, err := GetProjectPath()
	if err != nil {
		cwd, _ := os.Getwd()
		return filepath.Base(cwd), err
	}
	return filepath.Base(path), nil
}

// GetProjectPath returns the root path of the main repository
// If inside a {projectName}-wt-N sibling directory, derives the main repo path
func GetProjectPath() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// First, find the git root of the current directory
	repo, err := git.PlainOpenWithOptions(cwd, &git.PlainOpenOptions{
		DetectDotGit: true,
	})
	if err != nil {
		return cwd, err
	}

	wt, err := repo.Worktree()
	if err != nil {
		return cwd, err
	}

	repoRoot := wt.Filesystem.Root()

	// Check if we're in a {something}-wt-N directory (sibling workspace)
	dirName := filepath.Base(repoRoot)
	wtPattern := regexp.MustCompile(`^(.+)-wt-\d+$`)
	matches := wtPattern.FindStringSubmatch(dirName)
	if matches != nil {
		// We're in a {projectName}-wt-N workspace
		// The main repo should be {projectName} in the same parent directory
		mainProjectName := matches[1]
		parentDir := filepath.Dir(repoRoot)
		mainRepoPath := filepath.Join(parentDir, mainProjectName)

		// Verify the main repo exists and is a git repo
		if _, err := os.Stat(filepath.Join(mainRepoPath, ".git")); err == nil {
			return mainRepoPath, nil
		}
	}

	return repoRoot, nil
}
