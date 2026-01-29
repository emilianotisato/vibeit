package tui

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/emilianotisato/vibeit/internal/workspace"
	"github.com/emilianotisato/vibeit/internal/worktree"
)

// Modal types
type modalType int

const (
	modalNone modalType = iota
	modalNewWorktree
	modalDeleteWorktree
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

	statusMsgStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true)

	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2).
			Width(50)

	modalTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212")).
			Bold(true).
			MarginBottom(1)

	modalHintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("46"))
)

type keyMap struct {
	Quit       key.Binding
	NextTab    key.Binding
	PrevTab    key.Binding
	NewTerm    key.Binding
	Git        key.Binding
	Notes      key.Binding
	Worktree   key.Binding
	Delete     key.Binding
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
	Delete: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "delete worktree"),
	),
	CommandKey: key.NewBinding(
		key.WithKeys("ctrl+\\"),
		key.WithHelp("C-\\", "command mode"),
	),
}

type Model struct {
	projectName   string
	projectPath   string
	workspaces    []workspace.Workspace
	activeIdx     int
	width         int
	height        int
	err           error
	ready         bool
	statusMessage string

	// Modal state
	modal      modalType
	textInput  textinput.Model
	modalError string

	// Delete confirmation
	deleteConfirm bool
	deleteNotes   bool
}

func initialModel() Model {
	ti := textinput.New()
	ti.Placeholder = "feature-name"
	ti.CharLimit = 50
	ti.Width = 40

	return Model{
		projectName: "loading...",
		workspaces:  []workspace.Workspace{},
		activeIdx:   0,
		textInput:   ti,
	}
}

// Messages
type workspacesLoadedMsg struct {
	projectName string
	projectPath string
	workspaces  []workspace.Workspace
	err         error
}

type lazygitFinishedMsg struct {
	err error
}

type worktreeCreatedMsg struct {
	path string
	err  error
}

type worktreeDeletedMsg struct {
	err error
}

func loadWorkspaces() tea.Msg {
	projectName, _ := workspace.GetProjectName()
	projectPath, _ := workspace.GetProjectPath()
	workspaces, err := workspace.Detect()
	return workspacesLoadedMsg{
		projectName: projectName,
		projectPath: projectPath,
		workspaces:  workspaces,
		err:         err,
	}
}

func createWorktree(repoPath, branchName string) tea.Cmd {
	return func() tea.Msg {
		wtPath, err := worktree.Create(repoPath, branchName)
		if err != nil {
			return worktreeCreatedMsg{err: err}
		}

		// Run init
		if initErr := worktree.Init(repoPath, wtPath); initErr != nil {
			// Log but don't fail
			return worktreeCreatedMsg{path: wtPath, err: nil}
		}

		return worktreeCreatedMsg{path: wtPath, err: nil}
	}
}

func deleteWorktree(repoPath, wtPath, branchName string, deleteNotes bool) tea.Cmd {
	return func() tea.Msg {
		err := worktree.Delete(repoPath, wtPath, branchName, deleteNotes)
		return worktreeDeletedMsg{err: err}
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
		m.projectPath = msg.projectPath
		m.workspaces = msg.workspaces
		m.err = msg.err
		if len(m.workspaces) == 0 && m.err == nil {
			m.err = fmt.Errorf("not a git repository")
		}

	case lazygitFinishedMsg:
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("lazygit error: %v", msg.err)
		}
		return m, loadWorkspaces

	case worktreeCreatedMsg:
		m.modal = modalNone
		m.textInput.SetValue("")
		if msg.err != nil {
			m.statusMessage = errorStyle.Render(fmt.Sprintf("Error: %v", msg.err))
		} else {
			m.statusMessage = successStyle.Render(fmt.Sprintf("Created worktree: %s", filepath.Base(msg.path)))
		}
		return m, loadWorkspaces

	case worktreeDeletedMsg:
		m.modal = modalNone
		m.deleteConfirm = false
		if msg.err != nil {
			m.statusMessage = errorStyle.Render(fmt.Sprintf("Error: %v", msg.err))
		} else {
			m.statusMessage = successStyle.Render("Worktree deleted")
			// Adjust active index if needed
			if m.activeIdx >= len(m.workspaces)-1 && m.activeIdx > 0 {
				m.activeIdx--
			}
		}
		return m, loadWorkspaces

	case tea.KeyMsg:
		// Handle modal input first
		if m.modal != modalNone {
			return m.handleModalInput(msg)
		}

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
			m.modal = modalNewWorktree
			m.textInput.SetValue("")
			m.textInput.Focus()
			m.modalError = ""
			return m, textinput.Blink

		case key.Matches(msg, keys.Delete):
			if len(m.workspaces) > 0 {
				ws := m.workspaces[m.activeIdx]
				if ws.IsWorktree {
					m.modal = modalDeleteWorktree
					m.deleteConfirm = false
					m.deleteNotes = false
					m.modalError = ""
				} else {
					m.statusMessage = "Cannot delete main workspace"
				}
			}

		case msg.String() >= "1" && msg.String() <= "9":
			idx := int(msg.String()[0] - '1')
			if idx < len(m.workspaces) {
				m.activeIdx = idx
			}
		}
	}

	return m, nil
}

func (m Model) handleModalInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.modal {
	case modalNewWorktree:
		switch msg.String() {
		case "esc":
			m.modal = modalNone
			return m, nil
		case "enter":
			branchName := strings.TrimSpace(m.textInput.Value())
			if branchName == "" {
				m.modalError = "Branch name cannot be empty"
				return m, nil
			}
			// Validate branch name (simple check)
			if strings.ContainsAny(branchName, " \t\n\\:*?\"<>|") {
				m.modalError = "Invalid branch name"
				return m, nil
			}
			m.modalError = ""
			return m, createWorktree(m.projectPath, branchName)
		default:
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}

	case modalDeleteWorktree:
		switch msg.String() {
		case "esc":
			m.modal = modalNone
			return m, nil
		case "y", "Y":
			if !m.deleteConfirm {
				m.deleteConfirm = true
				return m, nil
			}
			ws := m.workspaces[m.activeIdx]
			return m, deleteWorktree(m.projectPath, ws.Path, ws.Branch, m.deleteNotes)
		case "n", "N":
			if m.deleteConfirm {
				// Toggle notes deletion
				m.deleteNotes = !m.deleteNotes
			} else {
				m.modal = modalNone
			}
			return m, nil
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
	contentHeight := m.height - 4
	b.WriteString(m.renderMainContent(contentHeight))

	// Footer
	b.WriteString(m.renderFooter())

	// Modal overlay
	if m.modal != modalNone {
		return m.renderWithModal(b.String())
	}

	return b.String()
}

func (m Model) renderWithModal(background string) string {
	var modal string

	switch m.modal {
	case modalNewWorktree:
		modal = m.renderNewWorktreeModal()
	case modalDeleteWorktree:
		modal = m.renderDeleteWorktreeModal()
	}

	// Center the modal
	lines := strings.Split(background, "\n")
	modalLines := strings.Split(modal, "\n")

	// Calculate vertical position (roughly center)
	startLine := (len(lines) - len(modalLines)) / 2
	if startLine < 2 {
		startLine = 2
	}

	// Calculate horizontal padding for centering
	modalWidth := lipgloss.Width(modal)
	leftPad := (m.width - modalWidth) / 2
	if leftPad < 0 {
		leftPad = 0
	}

	// Overlay modal on background
	for i, modalLine := range modalLines {
		lineIdx := startLine + i
		if lineIdx < len(lines) {
			paddedModal := strings.Repeat(" ", leftPad) + modalLine
			lines[lineIdx] = paddedModal
		}
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderNewWorktreeModal() string {
	var content strings.Builder

	content.WriteString(modalTitleStyle.Render("New Worktree"))
	content.WriteString("\n\n")
	content.WriteString("Branch name:\n")
	content.WriteString(m.textInput.View())

	if m.modalError != "" {
		content.WriteString("\n")
		content.WriteString(errorStyle.Render(m.modalError))
	}

	content.WriteString("\n")
	content.WriteString(modalHintStyle.Render("Enter to create • Esc to cancel"))

	return modalStyle.Render(content.String())
}

func (m Model) renderDeleteWorktreeModal() string {
	var content strings.Builder

	ws := m.workspaces[m.activeIdx]

	content.WriteString(modalTitleStyle.Render("Delete Worktree"))
	content.WriteString("\n\n")

	if !m.deleteConfirm {
		content.WriteString(fmt.Sprintf("Delete worktree '%s'?\n\n", ws.Name))
		content.WriteString(fmt.Sprintf("  Path: %s\n", ws.Path))
		content.WriteString(fmt.Sprintf("  Branch: %s\n", ws.Branch))
		content.WriteString("\n")
		content.WriteString(modalHintStyle.Render("Y to confirm • N to cancel"))
	} else {
		content.WriteString("Also delete notes file?\n\n")
		notesPath := filepath.Join(filepath.Dir(m.projectPath), ws.Branch+".md")
		content.WriteString(fmt.Sprintf("  %s\n", notesPath))
		content.WriteString("\n")
		deleteNotesStr := "No"
		if m.deleteNotes {
			deleteNotesStr = "Yes"
		}
		content.WriteString(fmt.Sprintf("Delete notes: %s\n", deleteNotesStr))
		content.WriteString("\n")
		content.WriteString(modalHintStyle.Render("N to toggle • Y to delete"))
	}

	return modalStyle.Render(content.String())
}

func (m Model) renderTopBar() string {
	projectPart := projectNameStyle.Render(m.projectName)

	var tabs []string
	for i, ws := range m.workspaces {
		name := ws.Name
		if ws.IsDirty {
			name += dirtyIndicator.String()
		}

		numPrefix := fmt.Sprintf("%d:", i+1)

		if i == m.activeIdx {
			tabs = append(tabs, activeTabStyle.Render(numPrefix+name))
		} else {
			tabs = append(tabs, inactiveTabStyle.Render(numPrefix+name))
		}
	}

	tabsPart := strings.Join(tabs, " ")
	content := projectPart + "  " + tabsPart

	padding := m.width - lipgloss.Width(content) - 2
	if padding < 0 {
		padding = 0
	}

	return topBarStyle.Width(m.width).Render(content + strings.Repeat(" ", padding))
}

func (m Model) renderMainContent(height int) string {
	if len(m.workspaces) == 0 {
		return mainContentStyle.Height(height).Render("No workspaces found")
	}

	ws := m.workspaces[m.activeIdx]

	statusLine := ""
	if m.statusMessage != "" {
		statusLine = "\n" + m.statusMessage
	}

	wtType := "main repo"
	if ws.IsWorktree {
		wtType = "worktree"
	}

	content := fmt.Sprintf(
		"Workspace: %s (%s)\n"+
			"Path: %s\n"+
			"Branch: %s\n"+
			"Status: %s%s\n\n"+
			"%s",
		ws.Name,
		wtType,
		ws.Path,
		ws.Branch,
		statusText(ws),
		statusLine,
		helpTextStyle.Render("g:lazygit  w:new worktree  d:delete worktree  t:terminal  n:notes"),
	)

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
	bindings := []struct {
		key  string
		desc string
	}{
		{"g", "lazygit"},
		{"w", "worktree"},
		{"d", "delete"},
		{"t", "terminal"},
		{"n", "notes"},
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

func Run() error {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
