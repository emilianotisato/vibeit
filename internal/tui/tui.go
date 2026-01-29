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
	"github.com/emilianotisato/vibeit/internal/mux"
	"github.com/emilianotisato/vibeit/internal/workspace"
	"github.com/emilianotisato/vibeit/internal/worktree"
)

// Modal types
type modalType int

const (
	modalNone modalType = iota
	modalNewWorktree
	modalDeleteWorktree
	modalTabPicker
	modalTabTypePicker
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

	modalItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	modalItemSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("230")).
				Background(lipgloss.Color("62")).
				Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("46"))
)

type keyMap struct {
	Quit        key.Binding
	NextTab     key.Binding
	PrevTab     key.Binding
	Terminal    key.Binding
	Git         key.Binding
	Claude      key.Binding
	Codex       key.Binding
	Neovim      key.Binding
	Notes       key.Binding
	Worktree    key.Binding
	Delete      key.Binding
	KillSession key.Binding
	Enter       key.Binding
	CommandKey  key.Binding
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
	Terminal: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "terminal"),
	),
	Git: key.NewBinding(
		key.WithKeys("g"),
		key.WithHelp("g", "lazygit"),
	),
	Claude: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "claude"),
	),
	Codex: key.NewBinding(
		key.WithKeys("x"),
		key.WithHelp("x", "codex"),
	),
	Neovim: key.NewBinding(
		key.WithKeys("v"),
		key.WithHelp("v", "nvim"),
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
	KillSession: key.NewBinding(
		key.WithKeys("k"),
		key.WithHelp("k", "kill session"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "tabs"),
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

	// Tab picker
	tabPickerTabs    []string
	tabPickerIdx     int
	tabPickerFilter  mux.TabType
	tabPickerSession string
	tabTypePickerIdx int
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

type externalCmdFinishedMsg struct {
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

		if initErr := worktree.Init(repoPath, wtPath); initErr != nil {
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

func runExternalCmd(cmd *exec.Cmd) tea.Cmd {
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return externalCmdFinishedMsg{err}
	})
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

	case externalCmdFinishedMsg:
		if msg.err != nil {
			m.statusMessage = errorStyle.Render(fmt.Sprintf("Error: %v", msg.err))
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
			if m.activeIdx >= len(m.workspaces)-1 && m.activeIdx > 0 {
				m.activeIdx--
			}
		}
		return m, loadWorkspaces

	case tea.KeyMsg:
		if m.modal != modalNone {
			return m.handleModalInput(msg)
		}

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

		case key.Matches(msg, keys.Enter):
			// Show all managed tabs
			if len(m.workspaces) > 0 {
				return m.showTabPicker(mux.TabType(""))
			}

		case key.Matches(msg, keys.Terminal):
			// Multi-instance terminal tabs
			if len(m.workspaces) > 0 {
				return m.showTabPicker(mux.TabTerminal)
			}

		case key.Matches(msg, keys.Git):
			// Single-instance lazygit
			if len(m.workspaces) > 0 {
				return m.openSingleTab(mux.TabLazygit)
			}

		case key.Matches(msg, keys.Claude):
			// Multi-instance claude tabs
			if len(m.workspaces) > 0 {
				return m.showTabPicker(mux.TabClaude)
			}

		case key.Matches(msg, keys.Codex):
			// Multi-instance codex tabs
			if len(m.workspaces) > 0 {
				return m.showTabPicker(mux.TabCodex)
			}

		case key.Matches(msg, keys.Neovim):
			// Multi-instance nvim tabs
			if len(m.workspaces) > 0 {
				return m.showTabPicker(mux.TabNeovim)
			}

		case key.Matches(msg, keys.Notes):
			if len(m.workspaces) > 0 {
				return m.openNotes()
			}

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

		case key.Matches(msg, keys.KillSession):
			if len(m.workspaces) > 0 {
				ws := m.workspaces[m.activeIdx]
				sessionName := mux.SessionName(m.projectName, ws.Name, ws.Branch)
				if err := mux.DeleteSession(sessionName); err != nil {
					m.statusMessage = errorStyle.Render(fmt.Sprintf("Failed to kill session: %v", err))
				} else {
					m.statusMessage = successStyle.Render(fmt.Sprintf("Killed session: %s", sessionName))
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

func (m Model) attachSession() (tea.Model, tea.Cmd) {
	if !mux.IsTmuxInstalled() {
		m.statusMessage = errorStyle.Render("tmux not installed. Run 'vibeit doctor' for help.")
		return m, nil
	}

	ws := m.workspaces[m.activeIdx]
	sessionName := mux.SessionName(m.projectName, ws.Name, ws.Branch)

	cmd := mux.AttachOrCreateCmd(sessionName, ws.Path)
	return m, runExternalCmd(cmd)
}

func (m Model) openSession(tabType mux.TabType) (tea.Model, tea.Cmd) {
	if !mux.IsTmuxInstalled() {
		m.statusMessage = errorStyle.Render("tmux not installed. Run 'vibeit doctor' for help.")
		return m, nil
	}

	ws := m.workspaces[m.activeIdx]
	sessionName := mux.SessionName(m.projectName, ws.Name, ws.Branch)

	cmd := mux.OpenWithCommand(sessionName, ws.Path, tabType)
	return m, runExternalCmd(cmd)
}

func (m Model) openSingleTab(tabType mux.TabType) (tea.Model, tea.Cmd) {
	if !mux.IsTmuxInstalled() {
		m.statusMessage = errorStyle.Render("tmux not installed. Run 'vibeit doctor' for help.")
		return m, nil
	}

	ws := m.workspaces[m.activeIdx]
	sessionName := mux.SessionName(m.projectName, ws.Name, ws.Branch)
	cmd := mux.GoToOrCreateSingleTabCmd(sessionName, ws.Path, tabType)
	return m, runExternalCmd(cmd)
}

func (m Model) showTabPicker(filter mux.TabType) (tea.Model, tea.Cmd) {
	if !mux.IsTmuxInstalled() {
		m.statusMessage = errorStyle.Render("tmux not installed. Run 'vibeit doctor' for help.")
		return m, nil
	}

	ws := m.workspaces[m.activeIdx]
	sessionName := mux.SessionName(m.projectName, ws.Name, ws.Branch)

	var tabs []string
	if mux.SessionExists(sessionName) {
		var err error
		tabs, err = mux.QueryTabNames(sessionName)
		if err != nil {
			m.statusMessage = errorStyle.Render(fmt.Sprintf("Failed to query tabs: %v", err))
			return m, nil
		}
	}

	if filter != "" {
		tabs = mux.FilterTabsByPrefix(tabs, string(filter))
	} else {
		tabs = filterManagedTabs(tabs)
	}

	m.tabPickerTabs = tabs
	m.tabPickerIdx = 0
	m.tabPickerFilter = filter
	m.tabPickerSession = sessionName
	m.modal = modalTabPicker

	return m, nil
}

func (m Model) openNotes() (tea.Model, tea.Cmd) {
	ws := m.workspaces[m.activeIdx]
	parentDir := filepath.Dir(m.projectPath)
	notesFile := fmt.Sprintf("%s-%s.md", m.projectName, ws.Branch)
	notesPath := filepath.Join(parentDir, notesFile)

	// Open notes in nvim directly (without tmux for simplicity)
	cmd := exec.Command("nvim", notesPath)
	cmd.Dir = ws.Path
	return m, runExternalCmd(cmd)
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
				m.deleteNotes = !m.deleteNotes
			} else {
				m.modal = modalNone
			}
			return m, nil
		}

	case modalTabPicker:
		return m.handleTabPickerInput(msg)

	case modalTabTypePicker:
		return m.handleTabTypePickerInput(msg)
	}

	return m, nil
}

func (m Model) tabPickerNewLabel() string {
	if m.tabPickerFilter == "" {
		return "New..."
	}
	return fmt.Sprintf("New %s", m.tabPickerFilter)
}

func (m Model) handleTabPickerInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	ws := m.workspaces[m.activeIdx]

	switch msg.String() {
	case "esc":
		m.modal = modalNone
		return m, nil

	case "up", "k":
		if m.tabPickerIdx > 0 {
			m.tabPickerIdx--
		}
		return m, nil

	case "down", "j":
		// +1 for "New" option
		if m.tabPickerIdx < len(m.tabPickerTabs) {
			m.tabPickerIdx++
		}
		return m, nil

	case "enter":
		m.modal = modalNone

		// Check if "New" option is selected (last item)
		if m.tabPickerIdx == len(m.tabPickerTabs) {
			if m.tabPickerFilter == "" {
				m.modal = modalTabTypePicker
				m.tabTypePickerIdx = 0
				return m, nil
			}

			tabName := mux.NextTabName(m.tabPickerTabs, m.tabPickerFilter)
			cmd := mux.NewTabCmd(m.tabPickerSession, ws.Path, tabName, m.tabPickerFilter)
			return m, runExternalCmd(cmd)
		}

		// Go to selected existing tab
		tabName := m.tabPickerTabs[m.tabPickerIdx]
		cmd := mux.GoToTabCmd(m.tabPickerSession, ws.Path, tabName)
		return m, runExternalCmd(cmd)

	// Quick select by number
	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		idx := int(msg.String()[0] - '1')
		// +1 because last item is "New"
		if idx <= len(m.tabPickerTabs) {
			m.tabPickerIdx = idx
			// Trigger enter
			return m.handleTabPickerInput(tea.KeyMsg{Type: tea.KeyEnter})
		}
		return m, nil
	}

	return m, nil
}

func (m Model) handleTabTypePickerInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	ws := m.workspaces[m.activeIdx]
	options := tabTypeOptions()

	switch msg.String() {
	case "esc":
		m.modal = modalTabPicker
		return m, nil

	case "up", "k":
		if m.tabTypePickerIdx > 0 {
			m.tabTypePickerIdx--
		}
		return m, nil

	case "down", "j":
		if m.tabTypePickerIdx < len(options)-1 {
			m.tabTypePickerIdx++
		}
		return m, nil

	case "enter":
		m.modal = modalNone
		tabType := options[m.tabTypePickerIdx]
		tabName := mux.NextTabName(m.tabPickerTabs, tabType)
		cmd := mux.NewTabCmd(m.tabPickerSession, ws.Path, tabName, tabType)
		return m, runExternalCmd(cmd)

	case "1", "2", "3", "4":
		idx := int(msg.String()[0] - '1')
		if idx < len(options) {
			m.tabTypePickerIdx = idx
			return m.handleTabTypePickerInput(tea.KeyMsg{Type: tea.KeyEnter})
		}
		return m, nil
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

	b.WriteString(m.renderTopBar())
	b.WriteString("\n")

	contentHeight := m.height - 4
	b.WriteString(m.renderMainContent(contentHeight))

	b.WriteString(m.renderFooter())

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
	case modalTabPicker:
		modal = m.renderTabPickerModal()
	case modalTabTypePicker:
		modal = m.renderTabTypePickerModal()
	}

	lines := strings.Split(background, "\n")
	modalLines := strings.Split(modal, "\n")

	startLine := (len(lines) - len(modalLines)) / 2
	if startLine < 2 {
		startLine = 2
	}

	modalWidth := lipgloss.Width(modal)
	leftPad := (m.width - modalWidth) / 2
	if leftPad < 0 {
		leftPad = 0
	}

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
		notesFile := fmt.Sprintf("%s-%s.md", m.projectName, ws.Branch)
		notesPath := filepath.Join(filepath.Dir(m.projectPath), notesFile)
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

func (m Model) renderTabPickerModal() string {
	var content strings.Builder

	title := "Tabs"
	if m.tabPickerFilter != "" {
		title = fmt.Sprintf("%s Tabs", tabTypeLabel(m.tabPickerFilter))
	}

	content.WriteString(modalTitleStyle.Render(title))
	content.WriteString("\n\n")

	items := append([]string{}, m.tabPickerTabs...)
	items = append(items, m.tabPickerNewLabel())

	for i, item := range items {
		prefix := "  "
		if i == m.tabPickerIdx {
			prefix = "> "
			content.WriteString(modalItemSelectedStyle.Render(prefix + item))
		} else {
			content.WriteString(modalItemStyle.Render(prefix + item))
		}
		content.WriteString("\n")
	}

	content.WriteString(modalHintStyle.Render("Enter to select • Esc to cancel"))
	return modalStyle.Render(content.String())
}

func (m Model) renderTabTypePickerModal() string {
	var content strings.Builder

	content.WriteString(modalTitleStyle.Render("New Tab"))
	content.WriteString("\n\n")

	options := tabTypeOptions()
	for i, option := range options {
		label := tabTypeLabel(option)
		prefix := "  "
		if i == m.tabTypePickerIdx {
			prefix = "> "
			content.WriteString(modalItemSelectedStyle.Render(prefix + label))
		} else {
			content.WriteString(modalItemStyle.Render(prefix + label))
		}
		content.WriteString("\n")
	}

	content.WriteString(modalHintStyle.Render("Enter to create • Esc to go back"))
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

		// Show tmux session indicator
		sessionName := mux.SessionName(m.projectName, ws.Name, ws.Branch)
		if mux.SessionExists(sessionName) {
			name += " ●"
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

	sessionName := mux.SessionName(m.projectName, ws.Name, ws.Branch)
	sessionStatus := "no session"
	if mux.SessionExists(sessionName) {
		sessionStatus = "session active"
	}

	sessionHint := ""
	if sessionStatus == "session active" {
		sessionHint = helpTextStyle.Render("\n\nTip: In tmux, press Ctrl+b then d to detach (keeps session alive)")
	}

	content := fmt.Sprintf(
		"Workspace: %s (%s)\n"+
			"Path: %s\n"+
			"Branch: %s\n"+
			"Git: %s\n"+
			"Session: %s%s%s\n\n"+
			"%s",
		ws.Name,
		wtType,
		ws.Path,
		ws.Branch,
		statusText(ws),
		sessionStatus,
		statusLine,
		sessionHint,
		helpTextStyle.Render("g:lazygit  c:claude  x:codex  v:nvim  t:term  n:notes  w:worktree  Enter:tabs"),
	)

	lines := strings.Split(content, "\n")
	for len(lines) < height {
		lines = append(lines, "")
	}

	return mainContentStyle.Render(strings.Join(lines[:height], "\n"))
}

func tabTypeOptions() []mux.TabType {
	return []mux.TabType{
		mux.TabClaude,
		mux.TabCodex,
		mux.TabNeovim,
		mux.TabTerminal,
	}
}

func tabTypeLabel(tabType mux.TabType) string {
	switch tabType {
	case mux.TabClaude:
		return "Claude"
	case mux.TabCodex:
		return "Codex"
	case mux.TabNeovim:
		return "Nvim"
	case mux.TabTerminal:
		return "Term"
	case mux.TabLazygit:
		return "Lazygit"
	default:
		return string(tabType)
	}
}

func filterManagedTabs(tabs []string) []string {
	var filtered []string
	for _, tab := range tabs {
		if isManagedTabName(tab) {
			filtered = append(filtered, tab)
		}
	}
	return filtered
}

func isManagedTabName(name string) bool {
	prefixes := []string{
		string(mux.TabLazygit),
		string(mux.TabClaude),
		string(mux.TabCodex),
		string(mux.TabNeovim),
		string(mux.TabTerminal),
	}
	for _, prefix := range prefixes {
		if name == prefix || strings.HasPrefix(name, prefix+"-") {
			return true
		}
	}
	return false
}

func statusText(ws workspace.Workspace) string {
	if ws.IsDirty {
		return "dirty"
	}
	return "clean"
}

func (m Model) renderFooter() string {
	bindings := []struct {
		key  string
		desc string
	}{
		{"g", "lazygit"},
		{"c", "claude"},
		{"x", "codex"},
		{"v", "nvim"},
		{"t", "term"},
		{"n", "notes"},
		{"w", "new wt"},
		{"d", "del wt"},
		{"k", "kill ses"},
		{"enter", "tabs"},
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

func Run() error {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
