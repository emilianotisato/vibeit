# Zellij Integration Lessons Learned

**ALWAYS check the official zellij documentation before implementing anything.**

## Key Commands Reference

### Session Management

```bash
# List sessions (may contain ANSI color codes!)
zellij list-sessions

# Attach to existing session
zellij attach <session-name>

# Attach, create if doesn't exist
zellij attach --create <session-name>

# Create session in background (detached)
zellij attach --create-background <session-name>

# Delete session (use --force for active sessions)
zellij delete-session --force <session-name>

# Kill session
zellij kill-session <session-name>
```

### Creating Sessions with Layouts

```bash
# Create NEW session with layout (ALWAYS creates new)
zellij --new-session-with-layout /path/to/layout.kdl

# Layout file must contain session_name directive:
# session_name "my-session"
# layout { ... }
```

### Actions on Specific Sessions

```bash
# Run action on a specific session (from OUTSIDE zellij)
zellij --session <session-name> action <action>

# Examples:
zellij --session mysession action query-tab-names
zellij --session mysession action go-to-tab-name "lazygit"
zellij --session mysession action new-pane -- lazygit
zellij --session mysession action new-tab --name "lazygit"
```

### Key Actions

```bash
# Go to tab by name (--create creates if doesn't exist)
zellij action go-to-tab-name "tabname"
zellij action go-to-tab-name --create "tabname"

# Create new tab
zellij action new-tab --name "tabname" --cwd /path

# Create new pane with command
zellij action new-pane --name "panename" -- command args

# Query all tab names
zellij action query-tab-names

# Dump current layout
zellij action dump-layout
```

### The Run Command

```bash
# Run command in new pane
zellij run -- lazygit
zellij run --name "lazygit" --close-on-exit -- lazygit
zellij run --floating -- htop
```

## Environment Variables

- `ZELLIJ` - Set to "0" when inside a zellij session
- `ZELLIJ_SESSION_NAME` - Current session name

## Common Pitfalls We Encountered

### 1. ANSI Color Codes in Output
`zellij list-sessions` output contains ANSI escape codes that break parsing.
**Solution**: Strip them with `sed 's/\x1b\[[0-9;]*m//g'` or check `ZELLIJ_SESSION_NAME` env var.

### 2. Confusing `-s/--session` Flag
The `-s/--session` flag is for specifying which session to run actions on, NOT for creating sessions.
- WRONG: `zellij -s mysession` (this doesn't create a session)
- RIGHT: `zellij attach --create mysession`

### 3. Layout Behavior Depends on Context
- `zellij --layout file.kdl` - Creates new session
- `zellij --session name --layout file.kdl` - Adds to existing session
- `zellij --new-session-with-layout file.kdl` - ALWAYS creates new session

### 4. Session Already Exists Error
When creating with layout and session exists:
**Solution**: Use `zellij attach --create` instead, or delete first with `--force`.

### 5. Actions Require Session Context
Actions like `query-tab-names`, `new-pane`, `go-to-tab-name` need either:
- To be run from inside zellij, OR
- Use `--session <name>` flag to specify target session

### 6. new-tab vs new-pane - CRITICAL!
- `new-tab` creates a new tab but **CANNOT run a command** - no `-- command` support!
- `new-pane -- command` can run a command in current tab
- **To create a tab with a command**: Use `--layout` with a layout file:
  ```bash
  # This adds a tab to existing session with a command
  zellij --session myses --layout /tmp/tab.kdl
  ```
  Where tab.kdl contains:
  ```kdl
  layout {
      tab name="lazygit" {
          pane command="lazygit"
      }
  }
  ```

## Recommended Approach for vibeit

1. **Check if session exists**:
   - Check `ZELLIJ_SESSION_NAME` env var first (if inside zellij)
   - Or parse `zellij list-sessions` carefully (strip ANSI codes)

2. **Create session**: Use `zellij attach --create-background <name>` to create detached

3. **Add tabs/panes**: Use `zellij --session <name> action new-pane -- command`

4. **Switch tabs**: Use `zellij --session <name> action go-to-tab-name "name"`

5. **Attach**: Use `zellij attach <name>`

## Useful Layout Template

```kdl
session_name "vibeit-project-branch"

layout {
    tab name="lazygit" {
        pane command="lazygit"
    }
    tab name="claude" {
        pane command="claude"
    }
    tab name="term" {
        pane
    }
}
```
