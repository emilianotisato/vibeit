# md-luncher

**Type**: Feature
**Status**: Draft
**Created**: 2026-02-02
**Parent PRD**: vibeit

## Overview

### Problem Statement

When working in vibeit, developers often need to reference markdown documentation (PRDs, specs, notes) stored in the project. Currently this requires leaving the TUI, navigating to the file manually, and opening it in a browser. There's no quick way to browse and open markdown files from within vibeit.

### Goals

- Provide quick access to markdown files from any workspace via a single keypress
- Allow browsing any folder recursively for `.md` files
- Open selected file in the system browser using `omarchy-launch-browser`

### Non-Goals

- Markdown preview within the TUI
- Multiple file selection
- Recent folder history
- Editing markdown files (that's what the notes feature is for)

## Requirements

### Functional Requirements

#### FR-1: Trigger Modal
**Priority**: Must Have
**Description**: Press `o` from any workspace to open the md-luncher modal.

#### FR-2: Folder Input
**Priority**: Must Have
**Description**: Modal prompts for folder path to scan. Default value: `docs`. User can change to any relative path (e.g., `.vibe`, `documentation/specs`).

#### FR-3: Recursive Scan
**Priority**: Must Have
**Description**: Scan the specified folder recursively for all `.md` files. Display results as a selectable list showing relative paths from the scanned folder.

#### FR-4: File Selection
**Priority**: Must Have
**Description**: Navigate the file list with arrow keys (or j/k). Press Enter to select and open.

#### FR-5: Launch Browser
**Priority**: Must Have
**Description**: On selection, run:
```
omarchy-launch-browser "<absolute-path-to-file>"
```
The path must be absolute (e.g., `/home/emiliano/Projects/vibeit-wt-1/.vibe/prd.md`).

#### FR-6: Error Handling
**Priority**: Must Have
**Description**: Show friendly error messages for:
- Folder does not exist: "Folder 'xxx' not found"
- No markdown files found: "No .md files found in 'xxx'"

#### FR-7: Cancel Modal
**Priority**: Must Have
**Description**: Press `Esc` at any point to close the modal and return to the workspace.

## User Story

### US-1: Quick Doc Access
**As a** developer working in vibeit
**I want to** quickly open a markdown file in my browser
**So that** I can reference documentation without leaving my workflow

**Acceptance Criteria**:
- [ ] Press `o` from any workspace
- [ ] Type folder name (or accept default `docs`)
- [ ] See list of all `.md` files in that folder (recursive)
- [ ] Select file and press Enter
- [ ] File opens in browser via `omarchy-launch-browser`
- [ ] Modal closes after launch

## Technical Design

### UI Flow

```
┌─────────────────────────────────────┐
│  Open Markdown                      │
│                                     │
│  Folder: [docs____________]         │
│                                     │
│  (Enter to scan, Esc to cancel)     │
└─────────────────────────────────────┘
           │
           ▼ Enter
┌─────────────────────────────────────┐
│  Select file (3 found)              │
│                                     │
│  > prd.md                           │
│    api-spec.md                      │
│    guides/setup.md                  │
│                                     │
│  (Enter to open, Esc to cancel)     │
└─────────────────────────────────────┘
           │
           ▼ Enter
    omarchy-launch-browser "/abs/path/to/prd.md"
```

### Implementation

| Component | Changes |
|-----------|---------|
| `internal/tui/` | New modal component for md-luncher |
| `internal/tui/keybindings.go` | Add `o` keybinding |
| `internal/mdluncher/` | New package for folder scanning logic |

### Key Functions

```go
// ScanMarkdownFiles returns all .md files in dir recursively
func ScanMarkdownFiles(dir string) ([]string, error)

// LaunchInBrowser opens file with omarchy-launch-browser
func LaunchInBrowser(absolutePath string) error
```

## Testing

### Unit Tests
- [ ] `ScanMarkdownFiles` finds nested `.md` files
- [ ] `ScanMarkdownFiles` returns error for non-existent folder
- [ ] `ScanMarkdownFiles` returns empty slice (not error) for folder with no `.md` files
- [ ] Absolute path construction is correct

### Manual Tests
- [ ] Modal opens on `o` keypress
- [ ] Default folder is `docs`
- [ ] File list shows recursive results
- [ ] Selected file opens in browser
- [ ] Error messages display correctly
- [ ] Esc closes modal at both stages

## Implementation Plan

### Single Phase
**Scope**: Complete feature implementation

**Deliverables**:
- [ ] Add `o` keybinding to workspace view
- [ ] Create folder input modal
- [ ] Implement recursive `.md` file scanning
- [ ] Create file selection list modal
- [ ] Implement `omarchy-launch-browser` execution
- [ ] Add error handling with friendly messages
- [ ] Update footer to show `o` keybinding hint
