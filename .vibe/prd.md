# vibeit

**Type**: Feature (New Application)
**Status**: Draft
**Created**: 2026-01-28
**Author**: Emiliano

## Overview

### Problem Statement

Modern AI-assisted development involves juggling multiple terminal sessions, worktrees, and AI agents (Claude Code, Codex) simultaneously. Developers currently manage this through manual window tiling, separate terminal tabs, and ad-hoc scripts. There's no unified interface to:

- See all active worktrees for a project at a glance
- Quickly switch context between parallel feature work
- Monitor what AI agents are doing across worktrees
- Review diffs and jump directly into neovim at the right location
- Keep mental notes per feature branch without cluttering the repo

### Goals

- Provide a single TUI "command center" for project-centric worktree management
- Enable rapid context switching between worktrees with keyboard shortcuts
- Offer on-demand terminal spawning for AI agents, lazygit, neovim, and general terminals
- Display git status (dirty/clean, ahead/behind) per worktree without requiring lazygit
- Support diff viewing with direct neovim integration (open file at specific line)
- Maintain per-worktree notes outside the git repo
- Support working directly on main branch (not just worktrees)

### Non-Goals (Out of Scope)

- Agent configuration management (CLAUDE.md, .codex/, etc.) - users set this up per project
- Window manager integration (Hyprland, i3, etc.) - vibeit is a standalone TUI
- Git operations (commit, push, rebase) - that's what lazygit is for
- Being a general-purpose terminal multiplexer - focused specifically on vibe coding workflow
- Remote repository management - local workflow only

## Background & Context

### Current State

The developer currently uses:
- **Omarchy/Hyprland** workspaces with tiled terminals
- **A custom `wt` bash script** for worktree creation with `.vibe/wt.json` config
- **lazygit** for all git operations
- **neovim** for editing
- **Claude Code / Codex** for AI assistance
- **Multiple terminal windows** manually arranged per feature

Pain points:
- Context switching requires finding the right workspace/windows
- No unified view of what's happening across worktrees
- Manual setup of terminal layouts for each new feature
- No easy way to jump from a diff to the exact line in neovim
- Mental notes get lost or scattered

### Why Now?

AI-assisted development has matured to where developers commonly run multiple agents in parallel. The tooling hasn't caught up - there's no purpose-built interface for this workflow on Linux.

### Related Work

- **[Ledger](https://github.com/peterjthomson/ledger)**: macOS/Electron git GUI for agent-powered development. Inspiration for the concept, but vibeit is a Linux TUI built in Go.
- **tmux/zellij**: Terminal multiplexers that vibeit will leverage under the hood
- **lazygit**: Git TUI that vibeit complements (not replaces)

## Users & Stakeholders

### Target Users

- **Solo developers**: Using AI assistants for personal projects, need to manage parallel feature work
- **Vibe coders**: Developers who embrace AI-assisted development workflow with multiple agents

### Stakeholders

- **Engineering**: Emiliano (sole developer)
- **Users**: Linux developers using terminal-based workflows

## Requirements

### Functional Requirements

#### FR-1: Project-Centric Launch
**Priority**: Must Have
**Description**: vibeit launches from a project directory and is aware only of that project. Running `vibeit` in `/home/user/projects/myapp` shows only myapp's worktrees and main branch.

#### FR-2: Main Branch as Default Workspace
**Priority**: Must Have
**Description**: The main repository directory is always available as a workspace, regardless of worktrees. Users can work directly on main without creating worktrees. This is the default view when no worktrees exist.

#### FR-3: Worktree Creation
**Priority**: Must Have
**Description**: Create new worktrees via modal prompt. User provides branch name, vibeit:
- Creates worktree at `../{project}-{branch}`
- Runs initialization from `.vibe/wt.json` (copy files, run commands)
- Adds worktree to the UI as a new workspace tab

#### FR-4: Worktree Deletion
**Priority**: Must Have
**Description**: Delete worktrees with confirmation modal showing:
- Branch name to be deleted
- Folder to be removed
- Option to also delete notes file (`../{branch}.md`)
Performs: `git worktree remove`, `git branch -D`, folder cleanup, optional notes deletion.

#### FR-5: Workspace Switching
**Priority**: Must Have
**Description**: Quick keyboard navigation between workspaces (main + worktrees). Suggested: `Ctrl+Alt+{number}` or similar prefix-based approach to switch between active workspaces.

#### FR-6: On-Demand Tab Creation
**Priority**: Must Have
**Description**: Within each workspace, create tabs on demand via modal:
- `(l)` lazygit
- `(c)` Claude Code agent
- `(x)` Codex agent
- `(n)` Notes (opens/creates `../{branch}.md`)
- `(t)` Plain terminal

Tabs are not pre-created; user spawns what they need.

#### FR-7: Tab Management
**Priority**: Must Have
**Description**: Close tabs, rename tabs, reorder tabs within a workspace. Footer shows context-sensitive keybindings based on current tab type.

#### FR-8: Git Status Polling
**Priority**: Must Have
**Description**: vibeit periodically polls git status for each workspace showing:
- Dirty/clean state (uncommitted changes indicator)
- Ahead/behind remote
- Current branch name
Displayed in top bar per workspace tab.

#### FR-9: Diff Viewer
**Priority**: Should Have
**Description**: View git diff for the current workspace without opening lazygit. Navigate changed files, see diff hunks, and press a key to open that file in neovim at the specific line.

#### FR-10: Neovim Integration
**Priority**: Must Have
**Description**: Open files in neovim from:
- Diff viewer (at specific line)
- Notes file
- General file picker
Uses nvim or neovim-remote to open in existing instance or spawn new one.

#### FR-11: Notes System
**Priority**: Should Have
**Description**: Per-worktree notes stored at `../{branch}.md` (outside repo, not committed). Created on first access. Editable via embedded neovim or simple text editor within vibeit.

#### FR-12: View Modes
**Priority**: Should Have
**Description**: Toggle between:
- **Tabs mode**: One workspace full-width, tabs for switching
- **Tile mode**: All workspaces visible in grid layout for overview

#### FR-13: Session Persistence
**Priority**: Nice to Have
**Description**: Remember open tabs per workspace between vibeit sessions. Store state in `.vibe/vibeit-session.json` or similar.

### Non-Functional Requirements

#### Performance
- Startup time: < 500ms
- Git status polling: Every 5 seconds (configurable)
- Tab switching: Instant (< 50ms)
- No noticeable lag when typing in embedded terminals

#### Security
- No remote connections except standard git operations
- Notes files stored locally, user-accessible
- No telemetry or data collection

#### Reliability
- Graceful handling of git errors (not a repo, detached HEAD, etc.)
- Clean shutdown of spawned processes on exit
- Recovery from crashed child processes (respawn or show error)

#### Scalability
- Support up to 10 concurrent workspaces
- Support up to 20 tabs total across workspaces
- Handle large repos without blocking UI

## User Stories / Use Cases

### US-1: Start New Feature Work
**As a** developer
**I want to** create a new worktree from vibeit with just a branch name
**So that** I can start working on a feature without manual setup

**Acceptance Criteria**:
- [ ] Press keybinding to create new worktree
- [ ] Modal prompts for branch name only
- [ ] Worktree created with proper naming convention
- [ ] `.vibe/wt.json` init runs automatically
- [ ] New workspace appears in top bar

### US-2: Quick Context Switch
**As a** developer with multiple features in progress
**I want to** instantly switch between worktrees
**So that** I can check on AI agent progress or review changes

**Acceptance Criteria**:
- [ ] Keyboard shortcut switches workspaces
- [ ] See git status indicator before switching
- [ ] Terminal state preserved when switching away

### US-3: Review AI Changes
**As a** developer supervising AI agents
**I want to** see the diff and jump to specific files in neovim
**So that** I can review and fix AI mistakes before committing

**Acceptance Criteria**:
- [ ] View diff without leaving vibeit
- [ ] Navigate between changed files
- [ ] Press key to open file in neovim at changed line
- [ ] Then commit manually via lazygit tab

### US-4: Spawn Tools On Demand
**As a** developer
**I want to** open lazygit, claude, or a terminal only when needed
**So that** I don't waste resources on unused tabs

**Acceptance Criteria**:
- [ ] Press key to open "new tab" modal
- [ ] Select tool type from list
- [ ] Tool spawns in new tab
- [ ] Tab can be closed when done

### US-5: Work on Main Branch
**As a** developer on a simple project
**I want to** use vibeit without creating worktrees
**So that** I can still benefit from the TUI for direct main branch work

**Acceptance Criteria**:
- [ ] Main branch workspace always available
- [ ] Can spawn tabs (lazygit, agents, etc.) on main
- [ ] No requirement to create worktrees

### US-6: Clean Up Completed Feature
**As a** developer who merged a feature
**I want to** delete the worktree, branch, and optionally notes
**So that** I keep my workspace clean

**Acceptance Criteria**:
- [ ] Select worktree to delete
- [ ] Confirmation shows what will be removed
- [ ] Option to keep or delete notes file
- [ ] Git worktree and branch properly cleaned up

### US-7: Take Notes While Reviewing
**As a** developer reviewing AI output
**I want to** jot down notes, ideas, and TODOs per feature
**So that** I don't forget insights while context switching

**Acceptance Criteria**:
- [ ] Open notes from any workspace
- [ ] Notes file created if doesn't exist
- [ ] Notes stored outside repo (not committed)
- [ ] Editable in neovim or embedded editor

## Technical Design

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                        vibeit TUI                           │
│  ┌───────────────────────────────────────────────────────┐  │
│  │ Top Bar: [main] [feature-auth*] [bugfix-123]    +new  │  │
│  └───────────────────────────────────────────────────────┘  │
│  ┌───────────────────────────────────────────────────────┐  │
│  │                                                       │  │
│  │              Terminal Multiplexer Embed               │  │
│  │                  (tmux or zellij)                     │  │
│  │                                                       │  │
│  │   Managed session: vibeit-{project}-{workspace}       │  │
│  │                                                       │  │
│  └───────────────────────────────────────────────────────┘  │
│  ┌───────────────────────────────────────────────────────┐  │
│  │ Footer: (n)otes (g)it (d)iff (t)erminal │ Ctrl+1..9  │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
        ┌─────────────────────────────────────┐
        │         Terminal Multiplexer        │
        │           (tmux / zellij)           │
        │                                     │
        │  Session: vibeit-myapp-main         │
        │  Session: vibeit-myapp-feature-auth │
        │  Session: vibeit-myapp-bugfix-123   │
        │                                     │
        │  Each session has windows/panes:    │
        │  - claude, lazygit, nvim, shell     │
        └─────────────────────────────────────┘
```

### Technology Stack

| Component | Technology | Rationale |
|-----------|------------|-----------|
| Language | Go | Fast startup, single binary, good TUI libraries |
| TUI Framework | [Bubble Tea](https://github.com/charmbracelet/bubbletea) | Mature, well-documented, Elm-inspired architecture |
| Styling | [Lip Gloss](https://github.com/charmbracelet/lipgloss) | Pairs with Bubble Tea |
| Terminal Mux | **zellij** (recommended) | Native pane management, locked mode for keybinding control, better UX than tmux for this use case |
| Git Operations | [go-git](https://github.com/go-git/go-git) | Pure Go git implementation for status/diff |
| Config | JSON (`.vibe/`) | Simple, already used for wt.json |

### Why Zellij over Tmux

1. **Locked mode**: Zellij can lock keyboard input so only zellij keybindings work - perfect for vibeit's meta-navigation
2. **Better pane management API**: Cleaner programmatic control
3. **Session management**: Named sessions with better isolation
4. **Modern defaults**: No legacy baggage, better out-of-box experience

### Components

| Component | Type | Description |
|-----------|------|-------------|
| `cmd/vibeit/main.go` | New | CLI entrypoint |
| `internal/tui/` | New | Bubble Tea TUI components |
| `internal/worktree/` | New | Worktree management logic |
| `internal/git/` | New | Git status, diff operations |
| `internal/mux/` | New | Zellij session management |
| `internal/config/` | New | Config loading (.vibe/*.json) |
| `internal/notes/` | New | Notes file management |

### Key Abstractions

```go
// Workspace represents main branch or a worktree
type Workspace struct {
    Name       string     // "main" or branch name
    Path       string     // Filesystem path
    IsWorktree bool       // false for main repo
    GitStatus  GitStatus  // Polled status
    Tabs       []Tab      // Open tabs
    MuxSession string     // zellij session name
}

// Tab represents an open terminal tab
type Tab struct {
    ID       string
    Type     TabType  // lazygit, claude, codex, terminal, notes
    Title    string
    PaneID   string   // zellij pane reference
}

// TabType enum
type TabType int
const (
    TabTerminal TabType = iota
    TabLazygit
    TabClaude
    TabCodex
    TabNotes
)
```

### Keybinding Strategy

**Problem**: When focused inside a terminal (e.g., neovim), how does the user trigger vibeit commands?

**Solution**: Use zellij's "locked" mode with a prefix key:

1. `Ctrl+Space` (or configurable) enters vibeit command mode
2. In command mode:
   - `1-9` switches workspaces
   - `n` opens notes
   - `g` opens lazygit
   - `t` new terminal
   - `d` diff view
   - `w` new worktree
   - `q` close tab
   - `Esc` returns to terminal

Alternatively, use zellij's native prefix (`Ctrl+g` by default) and custom keybindings.

### File Conventions

```
project/                     # Main repo (workspace: "main")
├── .vibe/
│   ├── wt.json             # Worktree init config (existing)
│   └── vibeit.json         # vibeit settings (new)
├── .env                     # Copied to worktrees
└── src/

../project-feature-auth/     # Worktree (workspace: "feature-auth")
├── .git                     # Worktree link
├── .env                     # Copied from main
└── src/

../feature-auth.md           # Notes file (outside repo)
```

### Configuration: `.vibe/vibeit.json`

```json
{
  "prefix_key": "ctrl+space",
  "git_poll_interval": 5,
  "default_tabs": [],
  "zellij_layout": "default"
}
```

### Technical Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Zellij API changes | Low | Medium | Pin zellij version, abstract mux layer |
| Terminal escape sequence issues | Medium | Medium | Use proven pty libraries, test on multiple terminals |
| Performance with many worktrees | Low | Low | Lazy load, limit polling frequency |
| Keybinding conflicts | Medium | Medium | Make prefix configurable, document conflicts |

## Implementation Plan

### Phase 1: Foundation
**Scope**: Core TUI shell, project detection, basic worktree listing

**Deliverables**:
- [ ] Project scaffolding with Go modules
- [ ] Bubble Tea app skeleton
- [ ] Top bar with project name
- [ ] Footer with static keybindings
- [ ] Detect if in git repo, find worktrees
- [ ] List workspaces (main + worktrees) in top bar

### Phase 2: Worktree Management
**Scope**: Create and delete worktrees

**Deliverables**:
- [ ] Modal for new worktree (branch name input)
- [ ] Worktree creation with naming convention
- [ ] Integration with `.vibe/wt.json` for init
- [ ] Modal for worktree deletion with confirmation
- [ ] Cleanup: git worktree remove, branch delete, folder delete
- [ ] Optional notes file deletion

### Phase 3: Terminal Integration
**Scope**: Zellij session management, embedded terminals

**Deliverables**:
- [ ] Create/attach zellij sessions per workspace
- [ ] Spawn tabs: terminal, lazygit, claude, codex
- [ ] Tab switching within workspace
- [ ] Tab closing
- [ ] Keybinding prefix for meta-navigation

### Phase 4: Git Integration
**Scope**: Status polling, diff viewer

**Deliverables**:
- [ ] Git status polling per workspace
- [ ] Status indicators in top bar (dirty, ahead/behind)
- [ ] Diff viewer panel
- [ ] Navigate changed files in diff
- [ ] Open file in neovim at line from diff

### Phase 5: Notes & Polish
**Scope**: Notes system, view modes, persistence

**Deliverables**:
- [ ] Notes file creation/opening
- [ ] Embedded editing (spawn neovim)
- [ ] Tile view mode (all workspaces visible)
- [ ] Session persistence (remember open tabs)
- [ ] Configuration file support

## Testing Strategy

### Unit Tests
- Worktree path generation
- Git status parsing
- Config file loading
- Keybinding parsing

### Integration Tests
- Worktree creation/deletion with real git
- Zellij session spawning
- File operations (notes, config)

### End-to-End Tests
- Full workflow: launch → create worktree → spawn tabs → switch → delete
- Test on: Alacritty, Kitty, foot (common Linux terminals)

### Manual Testing
- Keybinding conflicts with common tools (neovim, lazygit)
- Large repo performance
- Multiple vibeit instances (different projects)

## Success Metrics

### Key Metrics

| Metric | Target | How to Measure |
|--------|--------|----------------|
| Startup time | < 500ms | Benchmark |
| Context switch time | < 100ms | Benchmark |
| Daily usage | Actually use it | Self-assessment |
| Worktree setup time | < 5 seconds | From branch name to ready workspace |

### Definition of Done

- [ ] All Phase 1-5 deliverables complete
- [ ] Works on Arch Linux with common terminals
- [ ] Replaces manual worktree workflow entirely
- [ ] Documentation: README with install/usage
- [ ] Single binary distribution

## Open Questions

- [ ] Should vibeit manage its own zellij config or use user's existing?
- [ ] How to handle the case where user already has zellij running?
- [ ] Should notes support markdown preview in TUI or just raw editing?
- [ ] Should there be a "quick open" file picker beyond diff integration?

## Appendix

### Glossary

- **Worktree**: Git feature allowing multiple working directories from one repo
- **Workspace**: vibeit's term for main repo or a worktree
- **Tab**: A terminal pane within a workspace (lazygit, agent, etc.)
- **Vibe coding**: AI-assisted development workflow with agents like Claude/Codex

### References

- [Ledger - Git GUI for Agent-Powered Development](https://github.com/peterjthomson/ledger)
- [Bubble Tea - TUI Framework](https://github.com/charmbracelet/bubbletea)
- [Zellij - Terminal Multiplexer](https://zellij.dev/)
- [go-git - Pure Go Git Implementation](https://github.com/go-git/go-git)

### Existing Tooling

The user's current `wt` bash script handles:
- `wt <branch>` - Create worktree
- `wt delete` - Interactive deletion
- `wt init` - Copy files from `.vibe/wt.json`

This functionality will be absorbed into vibeit.
