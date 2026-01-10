package tui

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/existflow/irontask/internal/database"
	"github.com/existflow/irontask/internal/db"
	"github.com/existflow/irontask/internal/model"
	"github.com/existflow/irontask/internal/sync"
	"github.com/google/uuid"
)

// Pane represents which pane is focused
type Pane int

const (
	PaneSidebar Pane = iota
	PaneTaskList
)

// Mode represents the current UI mode
type Mode int

const (
	ModeNormal Mode = iota
	ModeAddTask
	ModeAddProject
	ModeEditTask
	ModeFilter
	ModeHelp
)

// Model is the main TUI model
type Model struct {
	db       *db.DB
	projects []database.Project
	tasks    []database.Task
	allTasks []database.Task // Original unfiltered list

	// Sync
	syncClient *sync.Client
	autoSync   *sync.AutoSync

	// UI state
	width      int
	height     int
	pane       Pane
	mode       Mode
	projCursor int
	taskCursor int

	// Input
	input textinput.Model

	// Sorting state
	recentlyDone map[string]time.Time

	// Filter (vim-style)
	filterText   string
	matchIndices []int // Indices of matching tasks
	matchCursor  int   // Current match for n/N navigation
	searchAll    bool  // true = all projects, false = current project

	message string
}

// Key bindings
type keyMap struct {
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	Tab      key.Binding
	Enter    key.Binding
	Add      key.Binding
	Done     key.Binding
	Delete   key.Binding
	Project  key.Binding
	Priority key.Binding
	Help     key.Binding
	Quit     key.Binding
	Escape   key.Binding
	Logout   key.Binding
}

var keys = keyMap{
	Up:      key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
	Down:    key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	Left:    key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "left pane")),
	Right:   key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "right pane")),
	Tab:     key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch pane")),
	Enter:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select/toggle")),
	Add:     key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add task")),
	Done:    key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "toggle done")),
	Delete:  key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
	Project: key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "new project")),
	Help:    key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	Escape:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
	Logout:  key.NewBinding(key.WithKeys("L"), key.WithHelp("L", "logout")),
}

// NewModel creates a new TUI model
func NewModel(database *db.DB) Model {
	ti := textinput.New()
	ti.Placeholder = "Enter task..."
	ti.CharLimit = 256
	ti.Width = 50

	m := Model{
		db:           database,
		pane:         PaneSidebar,
		mode:         ModeNormal,
		input:        ti,
		recentlyDone: make(map[string]time.Time),
	}

	// Initialize sync
	sClient, err := sync.NewClient()
	if err == nil && sClient.IsLoggedIn() {
		m.syncClient = sClient
		m.autoSync = sync.NewAutoSync(sClient, database)
		// Trigger initial sync
		m.autoSync.TriggerSync()
	}

	m.loadData()
	return m
}

func (m *Model) loadData() {
	m.projects, _ = m.db.ListProjects(context.Background())
	if m.projCursor >= len(m.projects) {
		m.projCursor = 0
	}
	if len(m.projects) > 0 {
		m.tasks, _ = m.db.ListTasks(context.Background(), database.ListTasksParams{
			ProjectID:   m.projects[m.projCursor].ID,
			IncludeDone: true,
		})

		// Sort tasks: Active first, Done last (with delay)
		sort.SliceStable(m.tasks, func(i, j int) bool {
			t1 := m.tasks[i]
			t2 := m.tasks[j]

			// Determine "effective done" status (delayed)
			isDone1 := t1.Done
			if isDone1 {
				if doneTime, ok := m.recentlyDone[t1.ID]; ok {
					if time.Since(doneTime) < 10*time.Second {
						isDone1 = false // Treat as active for sorting
					}
				}
			}

			isDone2 := t2.Done
			if isDone2 {
				if doneTime, ok := m.recentlyDone[t2.ID]; ok {
					if time.Since(doneTime) < 10*time.Second {
						isDone2 = false // Treat as active for sorting
					}
				}
			}

			// If one is done and other is not, active comes first
			if isDone1 != isDone2 {
				return !isDone1
			}

			// If both same state, sort by priority (1 is high) then creation time
			if t1.Priority != t2.Priority {
				return t1.Priority < t2.Priority
			}
			return t1.CreatedAt > t2.CreatedAt // Newest first (lexicographical string compare works for ISO8601)
		})
	}
}

func (m *Model) currentProject() *database.Project {
	if m.projCursor < len(m.projects) {
		return &m.projects[m.projCursor]
	}
	return nil
}

func (m *Model) currentTask() *database.Task {
	if m.taskCursor < len(m.tasks) {
		return &m.tasks[m.taskCursor]
	}
	return nil
}

// tickMsg is sent every second for time updates
type tickMsg time.Time

// Init initializes the model with a tick command
func (m Model) Init() tea.Cmd {
	return tea.Batch(tickCmd())
}

func tickCmd() tea.Cmd {
	return tea.Every(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tickMsg:
		// Check for delayed sorting
		needsRefresh := false
		for id, doneTime := range m.recentlyDone {
			if time.Since(doneTime) >= 10*time.Second {
				delete(m.recentlyDone, id)
				needsRefresh = true
			}
		}
		if needsRefresh {
			m.loadData()
		}
		// Continue ticking for time updates
		return m, tickCmd()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		// Handle mode-specific input
		switch m.mode {
		case ModeAddTask, ModeAddProject, ModeEditTask:
			return m.updateInput(msg)
		case ModeFilter:
			return m.updateFilter(msg)
		case ModeHelp:
			m.mode = ModeNormal
			return m, nil
		}

		// Normal mode
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, keys.Tab):
			if m.pane == PaneSidebar {
				m.pane = PaneTaskList
			} else {
				m.pane = PaneSidebar
			}

		case key.Matches(msg, keys.Left):
			m.pane = PaneSidebar

		case key.Matches(msg, keys.Right):
			m.pane = PaneTaskList

		case key.Matches(msg, keys.Up):
			if m.pane == PaneSidebar {
				if m.projCursor > 0 {
					m.projCursor--
					m.taskCursor = 0
					m.loadData()
				}
			} else {
				if m.taskCursor > 0 {
					m.taskCursor--
				}
			}

		case key.Matches(msg, keys.Down):
			if m.pane == PaneSidebar {
				if m.projCursor < len(m.projects)-1 {
					m.projCursor++
					m.taskCursor = 0
					m.loadData()
				}
			} else {
				if m.taskCursor < len(m.tasks)-1 {
					m.taskCursor++
				}
			}

		// Vim: gg = go to top
		case msg.String() == "g":
			// Wait for second 'g'
			return m, nil

		// Vim: G = go to bottom
		case msg.String() == "G":
			if m.pane == PaneSidebar {
				m.projCursor = len(m.projects) - 1
				m.taskCursor = 0
				m.loadData()
			} else {
				m.taskCursor = len(m.tasks) - 1
			}

		// Priority keys 1-4
		case msg.String() == "1", msg.String() == "2", msg.String() == "3", msg.String() == "4":
			if m.pane == PaneTaskList && len(m.tasks) > 0 {
				task := m.currentTask()
				if task != nil {
					priority := int(msg.String()[0] - '0')
					task.Priority = priority
					_ = m.db.UpdateTask(context.Background(), database.UpdateTaskParams{
						ID:        task.ID,
						ProjectID: task.ProjectID,
						Content:   task.Content,
						Done:      task.Done,
						Priority:  priority,
						DueDate:   task.DueDate,
						Tags:      task.Tags,
						UpdatedAt: time.Now().Format(time.RFC3339),
					})
					if m.autoSync != nil {
						m.autoSync.TriggerSync()
					}
					m.loadData()
					m.message = fmt.Sprintf("Priority set to P%d", priority)
				}
			}

		case key.Matches(msg, keys.Add):
			m.mode = ModeAddTask
			m.input.SetValue("")
			m.input.Placeholder = "Enter task..."
			m.input.Focus()
			return m, textinput.Blink

		case key.Matches(msg, keys.Project):
			m.mode = ModeAddProject
			m.input.SetValue("")
			m.input.Placeholder = "Enter project name..."
			m.input.Focus()
			return m, textinput.Blink

		case key.Matches(msg, keys.Done), key.Matches(msg, keys.Enter):
			if m.pane == PaneTaskList && len(m.tasks) > 0 {
				task := m.currentTask()
				if task != nil {
					newDone := !task.Done
					_ = m.db.MarkTaskDone(context.Background(), database.MarkTaskDoneParams{
						ID:        task.ID,
						Done:      newDone,
						UpdatedAt: time.Now().Format(time.RFC3339),
					})

					if newDone {
						m.recentlyDone[task.ID] = time.Now()
					} else {
						delete(m.recentlyDone, task.ID)
					}
					if m.autoSync != nil {
						m.autoSync.TriggerSync()
					}
					m.loadData()
				}
			}

		case key.Matches(msg, keys.Delete):
			if m.pane == PaneTaskList && len(m.tasks) > 0 {
				task := m.currentTask()
				if task != nil {
					_ = m.db.DeleteTask(context.Background(), database.DeleteTaskParams{
						ID:        task.ID,
						DeletedAt: sql.NullString{String: time.Now().Format(time.RFC3339), Valid: true},
						UpdatedAt: time.Now().Format(time.RFC3339),
					})
					if m.autoSync != nil {
						m.autoSync.TriggerSync()
					}
					m.loadData()
					if m.taskCursor >= len(m.tasks) && m.taskCursor > 0 {
						m.taskCursor--
					}
				}
			}

		// Edit task
		case msg.String() == "e":
			if m.pane == PaneTaskList && len(m.tasks) > 0 {
				task := m.currentTask()
				if task != nil {
					m.mode = ModeEditTask
					m.input.SetValue(task.Content)
					m.input.Placeholder = "Edit task..."
					m.input.Focus()
					m.input.CursorEnd()
					return m, textinput.Blink
				}
			}

		// Filter/search
		case msg.String() == "/":
			m.mode = ModeFilter
			m.input.SetValue(m.filterText)
			m.input.Placeholder = "/"
			m.input.Focus()
			return m, textinput.Blink

		// Next match (n)
		case msg.String() == "n":
			if len(m.matchIndices) > 0 {
				m.matchCursor = (m.matchCursor + 1) % len(m.matchIndices)
				m.taskCursor = m.matchIndices[m.matchCursor]
				m.message = fmt.Sprintf("[%d/%d] matches", m.matchCursor+1, len(m.matchIndices))
			}

		// Previous match (N)
		case msg.String() == "N":
			if len(m.matchIndices) > 0 {
				m.matchCursor--
				if m.matchCursor < 0 {
					m.matchCursor = len(m.matchIndices) - 1
				}
				m.taskCursor = m.matchIndices[m.matchCursor]
				m.message = fmt.Sprintf("[%d/%d] matches", m.matchCursor+1, len(m.matchIndices))
			}

		// Clear filter with Escape
		case key.Matches(msg, keys.Escape):
			if m.filterText != "" {
				m.filterText = ""
				m.matchIndices = nil
				m.message = "Filter cleared"
			}

		case key.Matches(msg, keys.Help):
			m.mode = ModeHelp

		case key.Matches(msg, keys.Logout):
			if m.syncClient != nil {
				if err := m.syncClient.Logout(); err != nil {
					m.message = fmt.Sprintf("Logout error: %v", err)
				} else {
					m.syncClient = nil
					m.autoSync = nil
					m.message = "Logged out successfully"
				}
			} else {
				m.message = "Not logged in"
			}
		}
	}

	return m, cmd
}

func (m Model) updateInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Escape):
		m.mode = ModeNormal
		return m, nil

	case key.Matches(msg, keys.Enter):
		value := m.input.Value()
		if value == "" {
			m.mode = ModeNormal
			return m, nil
		}

		switch m.mode {
		case ModeAddTask:
			proj := m.currentProject()
			if proj != nil {
				now := time.Now().Format(time.RFC3339)
				err := m.db.CreateTask(context.Background(), database.CreateTaskParams{
					ID:          uuid.New().String(),
					ProjectID:   proj.ID,
					Content:     value,
					Done:        false,
					Priority:    model.PriorityLow,
					CreatedAt:   now,
					UpdatedAt:   now,
					SyncVersion: sql.NullInt64{Int64: 0, Valid: true},
				})
				if err != nil {
					m.message = fmt.Sprintf("Error adding task: %v", err)
				} else {
					if m.autoSync != nil {
						m.autoSync.TriggerSync()
					}
					m.message = fmt.Sprintf("Added: %s", value)
				}
			}
		case ModeAddProject:
			now := time.Now().Format(time.RFC3339)
			err := m.db.CreateProject(context.Background(), database.CreateProjectParams{
				ID:          uuid.New().String()[:8],
				Name:        value,
				Color:       sql.NullString{String: "#4ECDC4", Valid: true},
				CreatedAt:   now,
				UpdatedAt:   now,
				SyncVersion: sql.NullInt64{Int64: 0, Valid: true},
			})
			if err != nil {
				m.message = fmt.Sprintf("Error creating project: %v", err)
			} else {
				if m.autoSync != nil {
					m.autoSync.TriggerSync()
				}
				m.message = fmt.Sprintf("Created project: %s", value)
			}
		case ModeEditTask:
			task := m.currentTask()
			if task != nil {
				task.Content = value
				_ = m.db.UpdateTask(context.Background(), database.UpdateTaskParams{
					ID:        task.ID,
					ProjectID: task.ProjectID,
					Content:   value,
					Done:      task.Done,
					Priority:  task.Priority,
					DueDate:   task.DueDate,
					Tags:      task.Tags,
					UpdatedAt: time.Now().Format(time.RFC3339),
				})
				if m.autoSync != nil {
					m.autoSync.TriggerSync()
				}
				m.message = fmt.Sprintf("Updated: %s", value)
			}
		}

		m.loadData()
		m.mode = ModeNormal
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) updateFilter(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Escape):
		m.mode = ModeNormal
		m.filterText = ""
		m.matchIndices = nil
		m.loadData()
		return m, nil

	case key.Matches(msg, keys.Tab):
		// Toggle between current project and all projects
		m.searchAll = !m.searchAll
		m.applyFilter()
		return m, nil

	case key.Matches(msg, keys.Up):
		// Navigate up in results
		if len(m.matchIndices) > 0 && m.matchCursor > 0 {
			m.matchCursor--
		}
		return m, nil

	case key.Matches(msg, keys.Down):
		// Navigate down in results
		if len(m.matchIndices) > 0 && m.matchCursor < len(m.matchIndices)-1 {
			m.matchCursor++
		}
		return m, nil

	case key.Matches(msg, keys.Enter):
		// Jump to selected match
		if len(m.matchIndices) > 0 && m.matchCursor < len(m.matchIndices) {
			m.taskCursor = m.matchIndices[m.matchCursor]
		}
		m.mode = ModeNormal
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	// Live filter as user types
	m.filterText = m.input.Value()
	m.applyFilter()
	return m, cmd
}

func (m *Model) applyFilter() {
	m.matchIndices = nil
	m.matchCursor = 0

	if m.filterText == "" {
		return
	}

	// Get tasks to search based on scope
	var tasksToSearch []database.Task
	if m.searchAll {
		// Search all projects
		tasksToSearch, _ = m.db.ListTasks(context.Background(), database.ListTasksParams{
			IncludeDone: true,
		})
	} else {
		tasksToSearch = m.tasks
	}

	filter := strings.ToLower(m.filterText)
	for i, t := range tasksToSearch {
		if strings.Contains(strings.ToLower(t.Content), filter) {
			m.matchIndices = append(m.matchIndices, i)
		}
	}

	// Store filtered tasks for display
	if m.searchAll {
		m.allTasks = tasksToSearch
	}
}

// View renders the UI
func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Build the layout
	sidebar := m.renderSidebar()
	taskList := m.renderTaskList()
	statusBar := m.renderStatusBar()

	// Combine sidebar and irontask list
	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, taskList)

	// Add modal if in input mode
	if m.mode == ModeAddTask || m.mode == ModeAddProject || m.mode == ModeEditTask {
		modal := m.renderModal()
		mainContent = lipgloss.Place(
			m.width, m.height-2,
			lipgloss.Center, lipgloss.Center,
			modal,
			lipgloss.WithWhitespaceChars(" "),
		)
	}

	// Filter modal with live results
	if m.mode == ModeFilter {
		modal := m.renderFilterModal()
		mainContent = lipgloss.Place(
			m.width, m.height-2,
			lipgloss.Center, lipgloss.Center,
			modal,
			lipgloss.WithWhitespaceChars(" "),
		)
	}

	if m.mode == ModeHelp {
		mainContent = m.renderHelp()
	}

	// Combine with status bar (filter input shows inline here)
	return lipgloss.JoinVertical(lipgloss.Left, mainContent, statusBar)
}

func (m Model) renderSidebar() string {
	sidebarWidth := 22
	var s string

	// Header with time
	now := time.Now().Format("15:04:05")
	s += lipgloss.NewStyle().Bold(true).Foreground(Primary).Render("IronTask") + "\n"
	s += HelpStyle.Render(now) + "\n"
	s += lipgloss.NewStyle().Foreground(Border).Render("─────────────────") + "\n\n"

	for i, p := range m.projects {
		counts, _ := m.db.CountTasks(context.Background(), p.ID)
		pending := counts.Count
		total := counts.Count_2

		cursor := "  "
		style := ProjectItemStyle
		if i == m.projCursor {
			cursor = "❯ "
			if m.pane == PaneSidebar {
				style = ProjectItemSelectedStyle
			}
		}

		line := fmt.Sprintf("%s %-10s %d/%d", cursor, truncate(p.Name, 10), pending, total)
		s += style.Render(line) + "\n"
	}

	s += "\n" + lipgloss.NewStyle().Foreground(Border).Render("─────────────────") + "\n"
	s += HelpStyle.Render("p new project")

	return SidebarStyle.Width(sidebarWidth).Height(m.height - 2).Render(s)
}

func (m Model) renderTaskList() string {
	width := m.width - 24
	var s string

	proj := m.currentProject()
	if proj == nil {
		return TaskListStyle.Width(width).Height(m.height - 2).Render("No project selected")
	}

	// Header
	pending := 0
	for _, t := range m.tasks {
		if !t.Done {
			pending++
		}
	}
	header := fmt.Sprintf("%s (%d pending)", proj.Name, pending)
	s += lipgloss.NewStyle().Bold(true).Foreground(Primary).Render(header) + "\n"
	s += lipgloss.NewStyle().Foreground(Border).Render(repeat("─", width-4)) + "\n\n"

	if len(m.tasks) == 0 {
		s += HelpStyle.Render("  No tasks. Press 'a' to add one.")
	}

	for i, t := range m.tasks {
		cursor := "  "
		style := TaskItemStyle
		if i == m.taskCursor && m.pane == PaneTaskList {
			cursor = "❯ "
			style = TaskItemSelectedStyle
		}

		// Highlight matching tasks
		isMatch := false
		for _, idx := range m.matchIndices {
			if idx == i {
				isMatch = true
				break
			}
		}
		if isMatch && i != m.taskCursor {
			style = lipgloss.NewStyle().Foreground(Highlight)
		}

		icon := "[ ]"
		if t.Done {
			icon = "[x]"
			style = TaskDoneStyle
		}

		content := truncate(t.Content, width-30)
		priority := FormatPriority(t.Priority)

		// Construct line carefully to avoid style nesting issues
		// Render parts
		check := style.Render(cursor + icon)
		desc := style.Render(fmt.Sprintf(" %-*s ", width-30, content))

		s += check + desc + priority + "\n"
	}

	return TaskListStyle.Width(width).Height(m.height - 2).Render(s)
}

func (m Model) renderStatusBar() string {
	// When in filter mode, show inline search input (like vim)
	if m.mode == ModeFilter {
		matches := ""
		if len(m.matchIndices) > 0 {
			matches = fmt.Sprintf(" [%d/%d]", m.matchCursor+1, len(m.matchIndices))
		} else if m.filterText != "" {
			matches = " [no match]"
		}
		return StatusBarStyle.Width(m.width).Render("/" + m.input.View() + matches)
	}

	help := "/:search  n/N:next/prev  a:add  e:edit  x:done  d:del  ?:help  q:quit  L:logout"
	if m.filterText != "" {
		if len(m.matchIndices) > 0 {
			help = fmt.Sprintf("/%s  [%d/%d matches]  n:next  N:prev  Esc:clear",
				m.filterText, m.matchCursor+1, len(m.matchIndices))
		} else {
			help = fmt.Sprintf("/%s  [no matches]  Esc:clear", m.filterText)
		}
	} else if m.message != "" {
		help = m.message
	}

	// Append sync status (right aligned)
	syncMsg := ""
	if m.autoSync != nil && m.autoSync.IsPending() {
		syncMsg = "Syncing..."
	}

	if syncMsg != "" {
		// Calculate available space
		avail := m.width - len(help) - len(syncMsg) - 2
		if avail > 0 {
			help += strings.Repeat(" ", avail) + syncMsg
		} else {
			help += " " + syncMsg
		}
	}

	return StatusBarStyle.Width(m.width).Render(help)
}

func (m Model) renderModal() string {
	title := "Add Task"
	switch m.mode {
	case ModeAddProject:
		title = "New Project"
	case ModeEditTask:
		title = "Edit Task"
	case ModeFilter:
		title = "🔍 Filter Tasks"
	}

	proj := m.currentProject()
	if proj != nil && m.mode == ModeAddTask {
		title = fmt.Sprintf("Add Task to: %s", proj.Name)
	}

	content := lipgloss.NewStyle().Bold(true).Render(title) + "\n\n"
	content += m.input.View() + "\n\n"
	content += HelpStyle.Render("Enter:save  Esc:cancel")

	return ModalStyle.Render(content)
}

func (m Model) renderFilterModal() string {
	modalWidth := 55
	maxResults := 8

	var content string

	// Scope indicator
	scope := "📁 Current Project"
	if m.searchAll {
		scope = "📂 All Projects"
	}

	// Header with scope toggle
	content += lipgloss.NewStyle().Bold(true).Foreground(Primary).Render("🔍 Search") + "  "
	content += HelpStyle.Render(scope) + "\n"
	content += HelpStyle.Render("Tab: toggle scope") + "\n\n"

	// Search input
	content += "/" + m.input.View() + "\n\n"

	// Divider
	content += lipgloss.NewStyle().Foreground(Border).Render(strings.Repeat("─", modalWidth-6)) + "\n\n"

	// Get tasks source
	var tasksSource []database.Task
	if m.searchAll {
		tasksSource = m.allTasks
	} else {
		tasksSource = m.tasks
	}

	// Show matching results
	if m.filterText == "" {
		content += HelpStyle.Render("Type to search...") + "\n"
	} else if len(m.matchIndices) == 0 {
		content += HelpStyle.Render("No matches found") + "\n"
	} else {
		content += fmt.Sprintf("%d matches\n\n", len(m.matchIndices))

		// Show matched tasks
		count := 0
		for i, idx := range m.matchIndices {
			if count >= maxResults {
				content += HelpStyle.Render(fmt.Sprintf("... +%d more", len(m.matchIndices)-maxResults)) + "\n"
				break
			}

			if idx >= len(tasksSource) {
				continue
			}

			t := tasksSource[idx]
			icon := "○"
			if t.Done {
				icon = "✓"
			}

			// Highlight current selection
			marker := "  "
			style := lipgloss.NewStyle()
			if i == m.matchCursor {
				marker = "❯ "
				style = lipgloss.NewStyle().Bold(true).Foreground(Primary)
			}

			taskLine := fmt.Sprintf("%s%s %s", marker, icon, truncate(t.Content, modalWidth-12))
			content += style.Render(taskLine) + "\n"
			count++
		}
	}

	content += "\n" + HelpStyle.Render("↑↓:nav  Enter:select  Esc:close")

	return ModalStyle.Width(modalWidth).Render(content)
}

func (m Model) renderHelp() string {
	help := `
╭─── Keyboard Shortcuts ───╮
│                          │
│  Navigation              │
│  ──────────              │
│  j/↓    Move down        │
│  k/↑    Move up          │
│  h/l    Switch pane      │
│  Tab    Switch pane      │
│  G      Go to bottom     │
│                          │
│  Actions                 │
│  ───────                 │
│  a       Add task        │
│  x/Enter Toggle done     │
│  d       Delete          │
│  p       New project     │
│  1-4     Set priority    │
│                          │
│  Other                   │
│  ─────                   │
│  ?       Toggle help     │
│  q       Quit            │
│                          │
╰──────────────────────────╯

     Press any key to close
`
	return lipgloss.Place(m.width, m.height-2, lipgloss.Center, lipgloss.Center, help)
}

// Helpers
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func repeat(s string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}
