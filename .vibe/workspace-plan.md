# Plan: Convert Git Worktrees to Directory-Based Workspaces

## Summary
Replace git worktrees with simple `git clone` based workspaces. Each workspace is an independent git repo located as a sibling directory: `{main_project}-wt-1` through `{main_project}-wt-9` (max 9). This eliminates agent authorization issues and database conflicts.

**Simplifications:**
- No workspace deletion functionality (users can just `rm -rf {project}-wt-N` manually)
- Single project-wide notes file (accessible from any workspace)

---

## Phase 1: Rename Package and Terminology

### 1.1 Rename `internal/worktree/` to `internal/workspace_init/`
- Rename directory and file
- Change package declaration to `package workspace_init`
- Update import in `internal/tui/tui.go`

### 1.2 Rename Constants, Types, and Functions
**In `tui.go`:**
- `modalNewWorktree` → `modalNewWorkspace`
- Remove `modalDeleteWorktree` entirely
- `worktreeCreatedMsg` → `workspaceCreatedMsg`
- Remove `worktreeDeletedMsg`
- `createWorktree()` → `createWorkspace()`
- Remove `deleteWorktree()`
- `keys.Worktree` → `keys.Workspace`
- Remove `keys.Delete` (d key)
- All user-facing strings: "worktree" → "workspace"

**In `workspace.go`:**
- `IsWorktree` field → `IsSiblingWorkspace`

---

## Phase 2: New Workspace Creation Logic

**File: `internal/workspace_init/workspace_init.go`**

Replace `Create()` function:
```go
func Create(mainRepoPath, branchName, baseBranch string) (string, error) {
    // 1. Find next available {projectName}-wt-N (1-9) as sibling directory
    //    e.g., /home/user/myproject → /home/user/myproject-wt-1
    // 2. git clone mainRepoPath → {projectName}-wt-N
    // 3. Update origin remote to point to original remote (not local clone)
    // 4. git checkout -b branchName [baseBranch]
    // 5. Return workspace path
}
```

Remove functions: `Delete()`, `List()`, `parseWorktreeList()` (no longer needed)

---

## Phase 3: Update Workspace Detection

**File: `internal/workspace/workspace.go`**

Replace `listWorktrees()` with new `listSiblingWorkspaces()`:
```go
func listSiblingWorkspaces(mainRepoPath string) []workspaceLocation {
    // 1. Add main workspace first
    // 2. Scan parent directory for {projectName}-wt-1 through {projectName}-wt-9
    //    e.g., if mainRepoPath is /home/user/myproject, look for:
    //    /home/user/myproject-wt-1, /home/user/myproject-wt-2, etc.
    // 3. Verify each is a git repo (.git exists)
    // 4. Return list of locations
}
```

Update `GetProjectPath()`:
```go
func GetProjectPath() (string, error) {
    // 1. Check if cwd matches {something}-wt-N pattern
    //    - If so, derive main repo path by removing "-wt-N" suffix
    // 2. Otherwise use existing logic
}
```

Update `Detect()` to use new `listSubWorkspaces()`

---

## Phase 4: Auto-Select Workspace on TUI Start

**File: `internal/tui/tui.go`**

Add function:
```go
func determineInitialWorkspaceIndex(workspaces []workspace.Workspace) int {
    // Match cwd to workspace path, return index (0 = main)
}
```

In `workspacesLoadedMsg` handler, set `m.activeIdx` based on cwd.

---

## Phase 5: Simplify Notes to Project-Wide File

**Current:** Each workspace has its own notes file (`{project}-{branch}.md`)
**New:** Single notes file for entire project (`{project}.md` in main repo parent dir)

**File: `internal/tui/tui.go`**
- Update `openNotes()` to always use `{projectPath}/../{projectName}.md`
- Remove notes display from workspace panel (or show same file for all)
- Remove notes-related deletion logic

---

## Phase 6: Update UI Labels and Remove Delete

**File: `internal/tui/tui.go`**

- Footer: Remove `"d:del wt"`, update `"w:new wt"` → `"w:new ws"`
- Remove `keys.Delete` keybinding
- Remove `modalDeleteWorktree` constant and all delete modal code
- Remove `renderDeleteWorktreeModal()` function
- Key binding help: `"new worktree"` → `"new workspace"`
- Modal titles: `"New Worktree"` → `"New Workspace"`

**File: `cmd/vibeit/main.go`**
- Update help text terminology

---

## Files to Modify

| File | Action |
|------|--------|
| `internal/worktree/worktree.go` | Rename to `internal/workspace_init/workspace_init.go`, rewrite Create, remove Delete/List |
| `internal/workspace/workspace.go` | New detection logic (`listSiblingWorkspaces`), rename `IsWorktree` → `IsSiblingWorkspace` |
| `internal/tui/tui.go` | Rename references, add initial selection, update labels, remove delete functionality, simplify notes |
| `cmd/vibeit/main.go` | Update help text |

---

## Verification

1. **Create workspace**: Press `w`, enter branch name → creates `myproject-wt-1` as sibling cloned repo
2. **Auto-select**: `cd myproject-wt-1 && vibeit` → TUI starts with that workspace tab selected
3. **Switch workspaces**: Number keys 1-9 switch between main and workspaces
4. **Notes**: Press `n` from any workspace → opens same project notes file
5. **wt.json**: Copy files and run commands work on new workspace creation
6. **Max limit**: After 9 workspaces, show error when trying to create 10th
7. **No delete key**: `d` key should do nothing (removed)

**Example directory structure:**
```
/home/user/
├── myproject/          # main repo
├── myproject-wt-1/     # workspace 1
├── myproject-wt-2/     # workspace 2
└── myproject.md        # shared notes file
```
