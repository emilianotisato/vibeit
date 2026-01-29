package workspace

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
)

// Workspace represents the main repo or a worktree
type Workspace struct {
	Name       string
	Path       string
	Branch     string
	IsWorktree bool
	IsDirty    bool
	Ahead      int
	Behind     int
}

// Detect finds the main repo and all worktrees from the current directory
func Detect() ([]Workspace, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	// Open the git repository
	repo, err := git.PlainOpenWithOptions(cwd, &git.PlainOpenOptions{
		DetectDotGit: true,
	})
	if err != nil {
		return nil, err
	}

	var workspaces []Workspace

	// Get worktree list using git command (go-git doesn't have great worktree support)
	worktrees, err := listWorktrees(cwd)
	if err != nil {
		// Fallback: just use current directory
		ws, err := getWorkspaceInfo(cwd, repo, false)
		if err != nil {
			return nil, err
		}
		return []Workspace{ws}, nil
	}

	for i, wt := range worktrees {
		isMain := i == 0 // First worktree is typically the main one
		ws, err := getWorkspaceInfo(wt.Path, repo, !isMain)
		if err != nil {
			continue
		}
		workspaces = append(workspaces, ws)
	}

	return workspaces, nil
}

type worktreeInfo struct {
	Path   string
	Branch string
}

func listWorktrees(repoPath string) ([]worktreeInfo, error) {
	// Find git directory
	repo, err := git.PlainOpenWithOptions(repoPath, &git.PlainOpenOptions{
		DetectDotGit: true,
	})
	if err != nil {
		return nil, err
	}

	wt, err := repo.Worktree()
	if err != nil {
		return nil, err
	}

	// Get the main worktree path
	mainPath := wt.Filesystem.Root()

	// Check for worktrees directory
	gitDir := filepath.Join(mainPath, ".git")
	info, err := os.Stat(gitDir)
	if err != nil {
		return nil, err
	}

	var worktreesDir string
	if info.IsDir() {
		worktreesDir = filepath.Join(gitDir, "worktrees")
	} else {
		// .git is a file (we're in a worktree), read it to find the real git dir
		content, err := os.ReadFile(gitDir)
		if err != nil {
			return nil, err
		}
		line := strings.TrimSpace(string(content))
		if strings.HasPrefix(line, "gitdir: ") {
			realGitDir := strings.TrimPrefix(line, "gitdir: ")
			// Go up from worktrees/<name> to the main .git
			worktreesDir = filepath.Dir(realGitDir)
			mainPath = filepath.Dir(filepath.Dir(worktreesDir))
		}
	}

	// Start with main worktree
	worktrees := []worktreeInfo{{Path: mainPath, Branch: "main"}}

	// Find additional worktrees
	if worktreesDir != "" {
		entries, err := os.ReadDir(worktreesDir)
		if err == nil {
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				wtPath := filepath.Join(worktreesDir, entry.Name(), "gitdir")
				content, err := os.ReadFile(wtPath)
				if err != nil {
					continue
				}
				path := strings.TrimSpace(string(content))
				// The gitdir file contains the path to the worktree's .git file
				// We need the parent directory
				path = filepath.Dir(path)
				worktrees = append(worktrees, worktreeInfo{
					Path:   path,
					Branch: entry.Name(),
				})
			}
		}
	}

	return worktrees, nil
}

func getWorkspaceInfo(path string, repo *git.Repository, isWorktree bool) (Workspace, error) {
	ws := Workspace{
		Path:       path,
		IsWorktree: isWorktree,
	}

	// Get name from directory
	ws.Name = filepath.Base(path)

	// Try to get branch name
	head, err := repo.Head()
	if err == nil {
		ws.Branch = head.Name().Short()
	}

	// Check if dirty
	wt, err := repo.Worktree()
	if err == nil {
		status, err := wt.Status()
		if err == nil {
			ws.IsDirty = !status.IsClean()
		}
	}

	return ws, nil
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
func GetProjectPath() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

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

	// Get the root path
	root := wt.Filesystem.Root()

	// If we're in a worktree, find the main repo
	gitPath := filepath.Join(root, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return root, nil
	}

	if !info.IsDir() {
		// .git is a file, we're in a worktree
		// Read it to find the main repo
		content, err := os.ReadFile(gitPath)
		if err != nil {
			return root, nil
		}
		line := strings.TrimSpace(string(content))
		if strings.HasPrefix(line, "gitdir: ") {
			gitDir := strings.TrimPrefix(line, "gitdir: ")
			// Go up from .git/worktrees/<name> to the main repo
			// gitDir is like /path/to/main/.git/worktrees/<name>
			mainGit := filepath.Dir(filepath.Dir(gitDir))
			return filepath.Dir(mainGit), nil
		}
	}

	return root, nil
}
