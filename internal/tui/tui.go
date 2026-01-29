package tui

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/emilianotisato/vibeit/internal/workspace"
)

// Styles
var (
	topBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("252")).
			Padding(0, 1)

	activeTabStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("230")).
			Bold(true).
			Padding(0, 2)

	inactiveTabStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("238")).
				Foreground(lipgloss.Color("250")).
				Padding(0, 2)

	dirtyIndicator = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			SetString("*")

	footerStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("245")).
			Padding(0, 1)

	footerKeyStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("230")).
			Padding(0, 1)

	footerDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("250")).
			Padding(0, 1, 0, 0)

	mainContentStyle = lipgloss.NewStyle().
				Padding(1, 2)

	projectNameStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("212")).
				Bold(true)

	helpTextStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
)

type keyMap struct {
	Quit       key.Binding
	NextTab    key.Binding
	PrevTab    key.Binding
	NewTerm    key.Binding
	Git        key.Binding
	Notes      key.Binding
	Worktree   key.Binding
	CommandKey key.Binding
}

var keys = keyMap{
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	NextTab: key.NewBinding(
		key.WithKeys("tab", "l"),
		key.WithHelp("tab", "next workspace"),
	),
	PrevTab: key.NewBinding(
		key.WithKeys("shift+tab", "h"),
		key.WithHelp("S-tab", "prev workspace"),
	),
	NewTerm: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "terminal"),
	),
	Git: key.NewBinding(
		key.WithKeys("g"),
		key.WithHelp("g", "lazygit"),
	),
	Notes: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "notes"),
	),
	Worktree: key.NewBinding(
		key.WithKeys("w"),
		key.WithHelp("w", "new worktree"),
	),
	CommandKey: key.NewBinding(
		key.WithKeys("ctrl+\\"),
		key.WithHelp("C-\\", "command mode"),
	),
}

type Model struct {
	projectName    string
	workspaces     []workspace.Workspace
	activeIdx      int
	width          int
	height         int
	commandMode    bool
	err            error
	ready          bool
	statusMessage  string
}

func initialModel() Model {
	return Model{
		projectName: "loading...",
		workspaces:  []workspace.Workspace{},
		activeIdx:   0,
	}
}

type workspacesLoadedMsg struct {
	projectName string
	workspaces  []workspace.Workspace
	err         error
}

func loadWorkspaces() tea.Msg {
	projectName, _ := workspace.GetProjectName()
	workspaces, err := workspace.Detect()
	return workspacesLoadedMsg{
		projectName: projectName,
		workspaces:  workspaces,
		err:         err,
	}
}

func (m Model) Init() tea.Cmd {
	return loadWorkspaces
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

	case workspacesLoadedMsg:
		m.projectName = msg.projectName
		m.workspaces = msg.workspaces
		m.err = msg.err
		if len(m.workspaces) == 0 && m.err == nil {
			m.err = fmt.Errorf("not a git repository")
		}

	case lazygitFinishedMsg:
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("lazygit error: %v", msg.err)
		}
		// Reload workspace status after lazygit (might have committed)
		return m, loadWorkspaces

	case tea.KeyMsg:
		// Clear status message on any key
		m.statusMessage = ""

		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, keys.NextTab):
			if len(m.workspaces) > 0 {
				m.activeIdx = (m.activeIdx + 1) % len(m.workspaces)
			}

		case key.Matches(msg, keys.PrevTab):
			if len(m.workspaces) > 0 {
				m.activeIdx = (m.activeIdx - 1 + len(m.workspaces)) % len(m.workspaces)
			}

		case key.Matches(msg, keys.Git):
			if len(m.workspaces) > 0 {
				ws := m.workspaces[m.activeIdx]
				return m, runLazygit(ws.Path)
			}

		case key.Matches(msg, keys.NewTerm):
			m.statusMessage = "Terminal: requires zellij (coming in Phase 3)"

		case key.Matches(msg, keys.Notes):
			m.statusMessage = "Notes: coming in Phase 5"

		case key.Matches(msg, keys.Worktree):
			m.statusMessage = "New worktree: coming in Phase 2"

		case msg.String() >= "1" && msg.String() <= "9":
			idx := int(msg.String()[0] - '1')
			if idx < len(m.workspaces) {
				m.activeIdx = idx
			}
		}
	}

	return m, nil
}

func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	if m.err != nil {
		return fmt.Sprintf("\n  Error: %v\n\n  Run 'vibeit' in a git repository.\n", m.err)
	}

	var b strings.Builder

	// Top bar
	b.WriteString(m.renderTopBar())
	b.WriteString("\n")

	// Main content area
	contentHeight := m.height - 4 // top bar + footer + margins
	b.WriteString(m.renderMainContent(contentHeight))

	// Footer
	b.WriteString(m.renderFooter())

	return b.String()
}

func (m Model) renderTopBar() string {
	// Project name
	projectPart := projectNameStyle.Render(m.projectName)

	// Workspace tabs
	var tabs []string
	for i, ws := range m.workspaces {
		name := ws.Name
		if ws.IsDirty {
			name += dirtyIndicator.String()
		}

		// Add number prefix for quick switching
		numPrefix := fmt.Sprintf("%d:", i+1)

		if i == m.activeIdx {
			tabs = append(tabs, activeTabStyle.Render(numPrefix+name))
		} else {
			tabs = append(tabs, inactiveTabStyle.Render(numPrefix+name))
		}
	}

	tabsPart := strings.Join(tabs, " ")

	// Combine project name and tabs
	content := projectPart + "  " + tabsPart

	// Fill remaining width
	padding := m.width - lipgloss.Width(content) - 2
	if padding < 0 {
		padding = 0
	}

	return topBarStyle.Width(m.width).Render(content + strings.Repeat(" ", padding))
}

var statusMsgStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("214")).
	Bold(true)

func (m Model) renderMainContent(height int) string {
	if len(m.workspaces) == 0 {
		return mainContentStyle.Height(height).Render("No workspaces found")
	}

	ws := m.workspaces[m.activeIdx]

	statusLine := ""
	if m.statusMessage != "" {
		statusLine = "\n" + statusMsgStyle.Render(m.statusMessage)
	}

	content := fmt.Sprintf(
		"Workspace: %s\n"+
			"Path: %s\n"+
			"Branch: %s\n"+
			"Status: %s%s\n\n"+
			"%s",
		ws.Name,
		ws.Path,
		ws.Branch,
		statusText(ws),
		statusLine,
		helpTextStyle.Render("Press 't' for terminal, 'g' for lazygit, 'n' for notes, 'w' for new worktree"),
	)

	// Fill height
	lines := strings.Split(content, "\n")
	for len(lines) < height {
		lines = append(lines, "")
	}

	return mainContentStyle.Render(strings.Join(lines[:height], "\n"))
}

func statusText(ws workspace.Workspace) string {
	if ws.IsDirty {
		return "dirty (uncommitted changes)"
	}
	return "clean"
}

func (m Model) renderFooter() string {
	// Keybindings
	bindings := []struct {
		key  string
		desc string
	}{
		{"t", "terminal"},
		{"g", "lazygit"},
		{"n", "notes"},
		{"w", "worktree"},
		{"1-9", "switch"},
		{"q", "quit"},
	}

	var parts []string
	for _, b := range bindings {
		parts = append(parts,
			footerKeyStyle.Render(b.key)+footerDescStyle.Render(b.desc),
		)
	}

	content := strings.Join(parts, " ")
	padding := m.width - lipgloss.Width(content) - 2
	if padding < 0 {
		padding = 0
	}

	return footerStyle.Width(m.width).Render(content + strings.Repeat(" ", padding))
}

func runLazygit(path string) tea.Cmd {
	c := exec.Command("lazygit")
	c.Dir = path
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return lazygitFinishedMsg{err}
	})
}

type lazygitFinishedMsg struct {
	err error
}

func Run() error {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
