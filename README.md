# vibeit

A worktree-centric TUI for managing git worktrees with integrated tmux sessions, AI coding assistants, and more.

## Features

- **Workspace Management**: Navigate between your main repo and git worktrees
- **Tmux Integration**: Each workspace gets its own tmux session with multiple tabs
- **AI Assistant Support**: Quick access to Claude and Codex coding assistants
- **Git Status**: Real-time git status, commits, ahead/behind tracking
- **Notes**: Per-branch markdown notes for tracking work
- **Lazygit Integration**: One-key access to lazygit

## Installation

### Requirements

- Go 1.24+
- git
- tmux (3.0+)
- neovim (0.9+)
- lazygit (optional)

### Arch Linux

Install dependencies:

```bash
sudo pacman -S git neovim tmux go
yay -S lazygit  # optional, or paru -S lazygit
```

Install vibeit:

```bash
git clone https://github.com/emilianotisato/vibeit.git
cd vibeit
make install
```

This installs to `~/.local/bin/vibeit`. Make sure `~/.local/bin` is in your PATH:

```bash
# Add to ~/.bashrc or ~/.zshrc
export PATH="$HOME/.local/bin:$PATH"
```

For system-wide installation:

```bash
sudo make install PREFIX=/usr
```

### Verify Installation

```bash
vibeit doctor
```

This checks that all dependencies are installed correctly.

## Usage

Run `vibeit` in any git repository:

```bash
cd your-project
vibeit
```

### Commands

| Command | Description |
|---------|-------------|
| `vibeit` | Launch the TUI |
| `vibeit doctor` | Check dependencies |
| `vibeit version` | Show version |
| `vibeit help` | Show help |

### Keybindings

| Key | Action |
|-----|--------|
| `1-9` | Switch workspace |
| `h/l` or `Tab/S-Tab` | Previous/Next workspace |
| `Enter` | Show all tabs |
| `g` | Open lazygit |
| `c` | Open Claude |
| `x` | Open Codex |
| `v` | Open neovim |
| `t` | New terminal |
| `n` | Open notes |
| `w` | Create new worktree |
| `d` | Delete worktree |
| `k` | Kill tmux session |
| `Ctrl+\` | Command mode (detach from tmux) |
| `F9` | Toggle tmux overview grid (managed windows) |
| `q` | Quit |

### Workflow

1. Run `vibeit` in your project root
2. Use `w` to create new worktrees for features/fixes
3. Switch between workspaces with `1-9` or `Tab`
4. Each workspace has its own tmux session with tabs for terminals, editors, and AI assistants
5. Use `n` to keep notes per branch
6. Delete worktrees with `d` when done

## Uninstall

```bash
make uninstall
```

Or for system-wide:

```bash
sudo make uninstall PREFIX=/usr
```

## License

MIT
