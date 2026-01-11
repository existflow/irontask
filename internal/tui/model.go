package tui

import (
	"context"
	"sort"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/existflow/irontask/internal/database"
	"github.com/existflow/irontask/internal/db"
	"github.com/existflow/irontask/internal/logger"
	"github.com/existflow/irontask/internal/sync"
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
	syncClient      *sync.Client
	autoSync        *sync.AutoSync
	syncRefreshChan chan struct{} // Channel to trigger UI refresh on remote sync

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

// NewModel creates a new TUI model
func NewModel(database *db.DB) Model {
	logger.Info("Initializing TUI model")

	ti := textinput.New()
	ti.Placeholder = "Enter task..."
	ti.CharLimit = 256
	ti.Width = 50

	m := Model{
		db:              database,
		pane:            PaneSidebar,
		mode:            ModeNormal,
		input:           ti,
		recentlyDone:    make(map[string]time.Time),
		syncRefreshChan: make(chan struct{}, 1), // Buffered to avoid blocking
	}

	// Initialize sync
	sClient, err := sync.NewClient()
	if err == nil && sClient.IsLoggedIn() {
		logger.Info("Sync client initialized and logged in")
		m.syncClient = sClient
		m.autoSync = sync.NewAutoSync(sClient, database)

		// Set callback to signal UI refresh when remote changes are pulled
		m.autoSync.SetOnPull(func() {
			logger.Debug("Auto-sync pull callback triggered")
			// Non-blocking send to trigger UI refresh
			select {
			case m.syncRefreshChan <- struct{}{}:
			default:
			}
		})

		// Trigger initial sync
		m.autoSync.TriggerSync()
	} else if err != nil {
		logger.Debug("Sync client not initialized", logger.F("error", err))
	} else {
		logger.Debug("Sync client not logged in")
	}

	m.loadData()
	logger.Debug("TUI model initialized",
		logger.F("projects", len(m.projects)),
		logger.F("tasks", len(m.tasks)))
	return m
}

func (m *Model) loadData() {
	m.projects, _ = m.db.ListProjects(context.Background())
	if m.projCursor >= len(m.projects) {
		m.projCursor = 0
	}
	if len(m.projects) > 0 {
		m.tasks, _ = m.db.ListTasks(context.Background(), database.ListTasksParams{
			ProjectID: m.projects[m.projCursor].ID,
			ShowAll:   true, // Show all including done
		})

		// Sort tasks: Active first, Done last (with delay)
		sort.SliceStable(m.tasks, func(i, j int) bool {
			t1 := m.tasks[i]
			t2 := m.tasks[j]

			// Determine "effective done" status (delayed)
			isDone1 := t1.Status.String == "done"
			if isDone1 {
				if doneTime, ok := m.recentlyDone[t1.ID]; ok {
					if time.Since(doneTime) < 10*time.Second {
						isDone1 = false // Treat as active for sorting
					}
				}
			}

			isDone2 := t2.Status.String == "done"
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
			return t1.CreatedAt > t2.CreatedAt // Newest first
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
