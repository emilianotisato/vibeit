package tui

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/emilianotisato/vibeit/internal/mux"
	"github.com/emilianotisato/vibeit/internal/workspace"
	workspace_init "github.com/emilianotisato/vibeit/internal/workspace_init"
)

// Modal types
type modalType int

const (
	modalNone modalType = iota
	modalNewWorkspace
	modalTabPicker
	modalTabTypePicker
)

const gitPollInterval = 5 * time.Second

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

	sectionTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("244")).
				Bold(true)

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))

	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	mutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	pillStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("238")).
			Bold(true).
			Padding(0, 1)

	pillGoodStyle = pillStyle.Copy().
			Background(lipgloss.Color("22"))

	pillWarnStyle = pillStyle.Copy().
			Background(lipgloss.Color("160"))

	pillInfoStyle = pillStyle.Copy().
			Background(lipgloss.Color("62"))

	commitHashStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("81")).
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
	Config      key.Binding
	Workspace   key.Binding
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
	Config: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "wt.json"),
	),
	Workspace: key.NewBinding(
		key.WithKeys("w"),
		key.WithHelp("w", "new workspace"),
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
	projectName    string
	projectPath    string
	workspaces     []workspace.Workspace
	activeIdx      int
	width          int
	height         int
	err            error
	ready          bool
	statusMessage  string
	gitPollActive  bool
	wtConfigExists bool

	// Modal state
	modal           modalType
	branchInput     textinput.Model
	baseBranchInput textinput.Model
	activeInput     int
	modalError      string

	baseBranchOptions  []string
	baseBranchFiltered []string
	baseBranchIdx      int

	// Tab picker
	tabPickerTabs    []string
	tabPickerIdx     int
	tabPickerFilter  mux.TabType
	tabPickerSession string
	tabTypePickerIdx int

	showTabPickerOnReturn bool
}

func initialModel() Model {
	branchInput := textinput.New()
	branchInput.Placeholder = "feature-name"
	branchInput.CharLimit = 50
	branchInput.Width = 40

	baseBranchInput := textinput.New()
	baseBranchInput.Placeholder = "base-branch"
	baseBranchInput.CharLimit = 50
	baseBranchInput.Width = 40

	return Model{
		projectName:     "loading...",
		workspaces:      []workspace.Workspace{},
		activeIdx:       0,
		branchInput:     branchInput,
		baseBranchInput: baseBranchInput,
		activeInput:     0,
	}
}

// Messages
type workspacesLoadedMsg struct {
	projectName  string
	projectPath  string
	workspaces   []workspace.Workspace
	configExists bool
	err          error
}

type externalCmdFinishedMsg struct {
	err error
}

type gitStatusTickMsg struct{}

type gitStatusMsg struct {
	workspaces []workspace.Workspace
	err        error
}

type workspaceCreatedMsg struct {
	path string
	err  error
}

func loadWorkspaces() tea.Msg {
	projectName, _ := workspace.GetProjectName()
	projectPath, _ := workspace.GetProjectPath()
	workspaces, err := workspace.Detect()
	configExists := false
	if projectPath != "" {
		configExists = workspace_init.ConfigExists(projectPath)
	}
	return workspacesLoadedMsg{
		projectName:  projectName,
		projectPath:  projectPath,
		workspaces:   workspaces,
		configExists: configExists,
		err:          err,
	}
}

func createWorkspace(repoPath, branchName, baseBranch string) tea.Cmd {
	return func() tea.Msg {
		wsPath, err := workspace_init.Create(repoPath, branchName, baseBranch)
		if err != nil {
			return workspaceCreatedMsg{err: err}
		}

		if initErr := workspace_init.Init(repoPath, wsPath); initErr != nil {
			return workspaceCreatedMsg{path: wsPath, err: nil}
		}

		return workspaceCreatedMsg{path: wsPath, err: nil}
	}
}

func runExternalCmd(cmd *exec.Cmd) tea.Cmd {
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return externalCmdFinishedMsg{err}
	})
}

// determineInitialWorkspaceIndex returns the index of the workspace that
// matches the current working directory, or 0 (main) if no match found
func determineInitialWorkspaceIndex(workspaces []workspace.Workspace) int {
	cwd, err := os.Getwd()
	if err != nil {
		return 0
	}

	for i, ws := range workspaces {
		if ws.Path == cwd {
			return i
		}
		// Also check if cwd is a subdirectory of the workspace
		if strings.HasPrefix(cwd, ws.Path+string(os.PathSeparator)) {
			return i
		}
	}
	return 0
}

func scheduleGitStatusTick() tea.Cmd {
	return tea.Tick(gitPollInterval, func(time.Time) tea.Msg {
		return gitStatusTickMsg{}
	})
}

func refreshGitStatus(workspaces []workspace.Workspace, projectPath, projectName string) tea.Cmd {
	wsSnapshot := make([]workspace.Workspace, len(workspaces))
	copy(wsSnapshot, workspaces)

	return func() tea.Msg {
		for i, ws := range wsSnapshot {
			updated := workspace.UpdateGitStatus(ws)
			exists, preview := notesPreview(projectPath, projectName, updated.Branch)
			updated.NotesExists = exists
			updated.NotesPreview = preview
			wsSnapshot[i] = updated
		}
		return gitStatusMsg{workspaces: wsSnapshot}
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
		m.wtConfigExists = msg.configExists
		m.err = msg.err
		if len(m.workspaces) == 0 && m.err == nil {
			m.err = fmt.Errorf("not a git repository")
		}
		// Auto-select workspace based on cwd on initial load
		if !m.gitPollActive {
			m.activeIdx = determineInitialWorkspaceIndex(m.workspaces)
		}
		cmds := []tea.Cmd{
			refreshGitStatus(m.workspaces, m.projectPath, m.projectName),
		}
		if !m.gitPollActive {
			m.gitPollActive = true
			cmds = append(cmds, scheduleGitStatusTick())
		}
		if m.showTabPickerOnReturn && msg.err == nil {
			m.showTabPickerOnReturn = false
			model, pickerCmd := m.showTabPicker(mux.TabType(""))
			if updated, ok := model.(Model); ok {
				m = updated
			}
			if pickerCmd != nil {
				cmds = append(cmds, pickerCmd)
			}
		} else {
			m.showTabPickerOnReturn = false
		}
		return m, tea.Batch(cmds...)

	case externalCmdFinishedMsg:
		if msg.err != nil {
			m.statusMessage = errorStyle.Render(fmt.Sprintf("Error: %v", msg.err))
		}
		return m, loadWorkspaces

	case gitStatusTickMsg:
		return m, refreshGitStatus(m.workspaces, m.projectPath, m.projectName)

	case gitStatusMsg:
		if msg.err == nil {
			m.workspaces = msg.workspaces
		}
		return m, scheduleGitStatusTick()

	case workspaceCreatedMsg:
		m.modal = modalNone
		m.branchInput.SetValue("")
		m.baseBranchInput.SetValue("")
		if msg.err != nil {
			m.statusMessage = errorStyle.Render(fmt.Sprintf("Error: %v", msg.err))
		} else {
			m.statusMessage = successStyle.Render(fmt.Sprintf("Created workspace: %s", filepath.Base(msg.path)))
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

		case key.Matches(msg, keys.Config):
			return m.openWorkspaceConfig()

		case key.Matches(msg, keys.Workspace):
			m.modal = modalNewWorkspace
			m.branchInput.SetValue("")
			branches, err := workspace.ListBranches(m.workspaces[m.activeIdx].Path)
			if err != nil {
				branches = nil
			}
			m.baseBranchOptions = branches
			m.baseBranchFiltered = nil
			m.baseBranchIdx = 0
			currentBranch := m.workspaces[m.activeIdx].Branch
			m.baseBranchInput.SetValue("")
			m.updateBaseBranchFilter()
			m.baseBranchIdx = indexOfBranch(m.baseBranchFiltered, currentBranch)
			m.branchInput.Focus()
			m.baseBranchInput.Blur()
			m.activeInput = 0
			m.modalError = ""
			return m, textinput.Blink

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
	m.showTabPickerOnReturn = true
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
	m.showTabPickerOnReturn = true
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
	m.showTabPickerOnReturn = true
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
	notesPath := notesPath(m.projectPath, m.projectName, ws.Branch)

	// Open notes in nvim directly (without tmux for simplicity)
	cmd := exec.Command("nvim", notesPath)
	cmd.Dir = ws.Path
	return m, runExternalCmd(cmd)
}

func (m Model) openWorkspaceConfig() (tea.Model, tea.Cmd) {
	if m.projectPath == "" {
		m.statusMessage = errorStyle.Render("Cannot locate repository path")
		return m, nil
	}

	configPath, created, err := workspace_init.EnsureConfig(m.projectPath)
	if err != nil {
		m.statusMessage = errorStyle.Render(fmt.Sprintf("Failed to open wt.json: %v", err))
		return m, nil
	}
	if created {
		m.statusMessage = successStyle.Render("Created .vibe/wt.json")
	}

	cmd := exec.Command("nvim", configPath)
	cmd.Dir = m.projectPath
	return m, runExternalCmd(cmd)
}

func (m Model) handleModalInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.modal {
	case modalNewWorkspace:
		switch msg.String() {
		case "esc":
			m.modal = modalNone
			return m, nil
		case "enter":
			branchName := strings.TrimSpace(m.branchInput.Value())
			if branchName == "" {
				m.modalError = "Branch name cannot be empty"
				return m, nil
			}
			if strings.ContainsAny(branchName, " \t\n\\:*?\"<>|") {
				m.modalError = "Invalid branch name"
				return m, nil
			}
			m.modalError = ""
			baseBranch := strings.TrimSpace(m.baseBranchInput.Value())
			if len(m.baseBranchFiltered) > 0 && m.baseBranchIdx < len(m.baseBranchFiltered) {
				baseBranch = m.baseBranchFiltered[m.baseBranchIdx]
			}
			if baseBranch == "" {
				baseBranch = m.workspaces[m.activeIdx].Branch
			}
			return m, createWorkspace(m.projectPath, branchName, baseBranch)
		case "up", "k":
			if m.activeInput == 1 && m.baseBranchIdx > 0 {
				m.baseBranchIdx--
				return m, nil
			}
		case "down", "j":
			if m.activeInput == 1 && m.baseBranchIdx < len(m.baseBranchFiltered)-1 {
				m.baseBranchIdx++
				return m, nil
			}
		case "tab", "shift+tab":
			if m.activeInput == 0 {
				m.activeInput = 1
				m.branchInput.Blur()
				m.baseBranchInput.Focus()
			} else {
				m.activeInput = 0
				m.baseBranchInput.Blur()
				m.branchInput.Focus()
			}
			return m, nil
		default:
			var cmd tea.Cmd
			if m.activeInput == 0 {
				m.branchInput, cmd = m.branchInput.Update(msg)
			} else {
				m.baseBranchInput, cmd = m.baseBranchInput.Update(msg)
				m.updateBaseBranchFilter()
			}
			return m, cmd
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
			m.showTabPickerOnReturn = true
			return m, runExternalCmd(cmd)
		}

		// Go to selected existing tab
		tabName := m.tabPickerTabs[m.tabPickerIdx]
		cmd := mux.GoToTabCmd(m.tabPickerSession, ws.Path, tabName)
		m.showTabPickerOnReturn = true
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
		m.showTabPickerOnReturn = true
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

	b.WriteString("\n")
	b.WriteString(m.renderFooter())

	if m.modal != modalNone {
		return m.renderWithModal(b.String())
	}

	return b.String()
}

func (m Model) renderWithModal(background string) string {
	var modal string

	switch m.modal {
	case modalNewWorkspace:
		modal = m.renderNewWorkspaceModal()
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

func (m Model) renderNewWorkspaceModal() string {
	var content strings.Builder

	content.WriteString(modalTitleStyle.Render("New Workspace"))
	content.WriteString("\n\n")
	content.WriteString("Branch name:\n")
	content.WriteString(m.branchInput.View())
	content.WriteString("\n\n")
	content.WriteString("Base branch:\n")
	content.WriteString(m.baseBranchInput.View())
	content.WriteString("\n")
	content.WriteString(m.renderBaseBranchOptions())

	if m.modalError != "" {
		content.WriteString("\n")
		content.WriteString(errorStyle.Render(m.modalError))
	}

	content.WriteString("\n")
	content.WriteString(modalHintStyle.Render("Tab to switch • ↑↓ to pick base • Enter to create • Esc to cancel"))

	return modalStyle.Render(content.String())
}

func (m Model) renderBaseBranchOptions() string {
	if len(m.baseBranchOptions) == 0 {
		return helpTextStyle.Render("  (no branches found)")
	}
	if len(m.baseBranchFiltered) == 0 {
		return helpTextStyle.Render("  (no matches)")
	}

	total := len(m.baseBranchFiltered)
	maxItems := 6
	if total < maxItems {
		maxItems = total
	}

	start := 0
	if m.baseBranchIdx >= maxItems {
		start = m.baseBranchIdx - maxItems + 1
	}
	end := start + maxItems
	if end > total {
		end = total
		start = end - maxItems
		if start < 0 {
			start = 0
		}
	}

	var content strings.Builder
	if start > 0 {
		content.WriteString(helpTextStyle.Render(fmt.Sprintf("  ...%d above", start)))
		content.WriteString("\n")
	}
	for i := start; i < end; i++ {
		prefix := "  "
		if i == m.baseBranchIdx {
			prefix = "> "
			content.WriteString(modalItemSelectedStyle.Render(prefix + m.baseBranchFiltered[i]))
		} else {
			content.WriteString(modalItemStyle.Render(prefix + m.baseBranchFiltered[i]))
		}
		if i < end-1 {
			content.WriteString("\n")
		}
	}

	if end < total {
		remaining := total - end
		content.WriteString("\n")
		content.WriteString(helpTextStyle.Render(fmt.Sprintf("  ...%d more", remaining)))
	}

	return content.String()
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
		if ws.Ahead > 0 || ws.Behind > 0 {
			name += fmt.Sprintf(" ↑%d↓%d", ws.Ahead, ws.Behind)
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

	innerWidth := m.width - 4
	contentWidth := innerWidth
	if contentWidth < 20 {
		contentWidth = 20
	}

	var content string
	if innerWidth < 70 {
		left := m.renderWorkspacePanel(ws, contentWidth)
		right := m.renderGitPanel(ws, contentWidth)
		content = left + "\n\n" + right
	} else {
		gap := mutedStyle.Render(" | ")
		colWidth := (contentWidth - lipgloss.Width(gap)) / 2
		if colWidth < 20 {
			colWidth = 20
		}
		left := m.renderWorkspacePanel(ws, colWidth)
		right := m.renderGitPanel(ws, colWidth)
		leftCol := lipgloss.NewStyle().Width(colWidth).Render(left)
		rightCol := lipgloss.NewStyle().Width(colWidth).Render(right)
		content = lipgloss.JoinHorizontal(lipgloss.Top, leftCol, gap, rightCol)
	}

	lines := strings.Split(content, "\n")
	for len(lines) < height {
		lines = append(lines, "")
	}

	return mainContentStyle.Render(strings.Join(lines[:height], "\n"))
}

func (m Model) renderWorkspacePanel(ws workspace.Workspace, width int) string {
	labelWidth := 8
	wtType := "main repo"
	if ws.IsSubWorkspace {
		wtType = "sub-workspace"
	}

	sessionName := mux.SessionName(m.projectName, ws.Name, ws.Branch)
	sessionActive := mux.SessionExists(sessionName)
	sessionValue := mutedStyle.Render("NONE")
	if sessionActive {
		sessionValue = pillInfoStyle.Render("ACTIVE")
	}

	configValue := pillGoodStyle.Render("OK")
	if !m.wtConfigExists {
		configValue = pillWarnStyle.Render("MISSING")
	}

	var content strings.Builder
	content.WriteString(sectionTitleStyle.Render("WORKSPACE"))
	content.WriteString("\n")
	content.WriteString(formatLabelValue("Name", ws.Name, labelWidth, width))
	content.WriteString("\n")
	content.WriteString(formatLabelValue("Type", wtType, labelWidth, width))
	content.WriteString("\n")
	content.WriteString(formatLabelValue("Branch", ws.Branch, labelWidth, width))
	content.WriteString("\n")
	content.WriteString(formatLabelValue("Path", truncateMiddle(ws.Path, width-labelWidth-1), labelWidth, width))
	content.WriteString("\n")
	content.WriteString(formatLabelLine("Session", sessionValue, labelWidth))
	content.WriteString("\n")
	content.WriteString(formatLabelLine("WT cfg", configValue, labelWidth))

	if sessionActive {
		content.WriteString("\n")
		content.WriteString(helpTextStyle.Render("Detach: Ctrl+\\  Last tab: Ctrl+]"))
	}

	if !m.wtConfigExists {
		content.WriteString("\n")
		content.WriteString(helpTextStyle.Render("Press e to edit .vibe/wt.json"))
	}

	if msg := styleStatusMessage(m.statusMessage); msg != "" {
		content.WriteString("\n")
		content.WriteString(msg)
	}

	content.WriteString("\n\n")
	content.WriteString(sectionTitleStyle.Render("NOTES"))
	content.WriteString("\n")
	content.WriteString(renderNotes(ws.NotesPreview, ws.NotesExists, width))

	return content.String()
}

func (m Model) renderGitPanel(ws workspace.Workspace, width int) string {
	labelWidth := 8
	statusValue := pillGoodStyle.Render("CLEAN")
	if ws.IsDirty {
		statusValue = pillWarnStyle.Render("DIRTY")
	}

	syncValue := pillInfoStyle.Render("SYNC")
	if ws.Ahead > 0 || ws.Behind > 0 {
		parts := []string{}
		if ws.Ahead > 0 {
			parts = append(parts, pillInfoStyle.Render(fmt.Sprintf("AHEAD %d", ws.Ahead)))
		}
		if ws.Behind > 0 {
			parts = append(parts, pillInfoStyle.Render(fmt.Sprintf("BEHIND %d", ws.Behind)))
		}
		syncValue = strings.Join(parts, " ")
	}

	stashValue := mutedStyle.Render("0")
	if ws.StashCount > 0 {
		stashValue = pillInfoStyle.Render(fmt.Sprintf("STASH %d", ws.StashCount))
	}

	var content strings.Builder
	content.WriteString(sectionTitleStyle.Render("GIT"))
	content.WriteString("\n")
	content.WriteString(formatLabelLine("Status", statusValue, labelWidth))
	content.WriteString("\n")
	content.WriteString(formatLabelLine("Sync", syncValue, labelWidth))
	content.WriteString("\n")
	content.WriteString(formatLabelLine("Stash", stashValue, labelWidth))
	content.WriteString("\n\n")
	content.WriteString(sectionTitleStyle.Render("COMMITS"))
	content.WriteString("\n")
	content.WriteString(renderCommits(ws.RecentCommits, width))

	return content.String()
}

func formatLabelValue(label, value string, labelWidth, width int) string {
	labelText := fmt.Sprintf("%-*s", labelWidth, label+":")
	valueMax := width - labelWidth - 1
	if valueMax < 0 {
		valueMax = 0
	}
	return labelStyle.Render(labelText) + " " + valueStyle.Render(truncateText(value, valueMax))
}

func formatLabelLine(label, value string, labelWidth int) string {
	labelText := fmt.Sprintf("%-*s", labelWidth, label+":")
	return labelStyle.Render(labelText) + " " + value
}

func renderNotes(lines []string, exists bool, width int) string {
	if !exists {
		return mutedStyle.Render("  (no notes yet)") + "\n" + helpTextStyle.Render("  Press n to create")
	}
	if len(lines) == 0 {
		return mutedStyle.Render("  (empty)")
	}

	var content strings.Builder
	for i, line := range lines {
		prefix := fmt.Sprintf("%2d ", i+1)
		available := width - len(prefix)
		if available < 0 {
			available = 0
		}
		content.WriteString(mutedStyle.Render(prefix))
		content.WriteString(truncateText(line, available))
		if i < len(lines)-1 {
			content.WriteString("\n")
		}
	}
	return content.String()
}

func renderCommits(commits []string, width int) string {
	if len(commits) == 0 {
		return mutedStyle.Render("  (none)")
	}

	var content strings.Builder
	for i, line := range commits {
		hash, msg := splitCommitLine(line)
		prefix := mutedStyle.Render("- ")
		available := width - 2 - len(hash) - 1
		if available < 0 {
			available = 0
		}
		content.WriteString(prefix)
		content.WriteString(commitHashStyle.Render(hash))
		if msg != "" {
			content.WriteString(" ")
			content.WriteString(mutedStyle.Render(truncateText(msg, available)))
		}
		if i < len(commits)-1 {
			content.WriteString("\n")
		}
	}
	return content.String()
}

func splitCommitLine(line string) (string, string) {
	parts := strings.SplitN(strings.TrimSpace(line), " ", 2)
	if len(parts) == 0 || parts[0] == "" {
		return line, ""
	}
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}

func truncateText(value string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}

func truncateMiddle(value string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	if max <= 3 {
		return string(runes[:max])
	}
	head := (max - 3) / 2
	tail := max - 3 - head
	return string(runes[:head]) + "..." + string(runes[len(runes)-tail:])
}

func styleStatusMessage(message string) string {
	if message == "" {
		return ""
	}
	if strings.Contains(message, "\x1b[") {
		return message
	}
	return statusMsgStyle.Render(message)
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

func (m *Model) updateBaseBranchFilter() {
	query := strings.TrimSpace(m.baseBranchInput.Value())
	m.baseBranchFiltered = filterBranches(m.baseBranchOptions, query)
	m.baseBranchIdx = 0
	if query == "" {
		return
	}
	for i, branch := range m.baseBranchFiltered {
		if branch == query {
			m.baseBranchIdx = i
			return
		}
	}
}

func filterBranches(branches []string, query string) []string {
	if len(branches) == 0 {
		return nil
	}
	if query == "" {
		filtered := make([]string, len(branches))
		copy(filtered, branches)
		return filtered
	}
	query = strings.ToLower(query)
	var filtered []string
	for _, branch := range branches {
		if strings.Contains(strings.ToLower(branch), query) {
			filtered = append(filtered, branch)
		}
	}
	return filtered
}

func indexOfBranch(branches []string, target string) int {
	for i, branch := range branches {
		if branch == target {
			return i
		}
	}
	return 0
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

func notesPath(projectPath, projectName, _ string) string {
	parentDir := filepath.Dir(projectPath)
	notesFile := fmt.Sprintf("%s.md", projectName)
	return filepath.Join(parentDir, notesFile)
}

func notesPreview(projectPath, projectName, branch string) (bool, []string) {
	path := notesPath(projectPath, projectName, branch)
	file, err := os.Open(path)
	if err != nil {
		return false, nil
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) >= 10 {
			break
		}
	}
	return true, lines
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
		{"e", "wt.json"},
		{"w", "new ws"},
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
