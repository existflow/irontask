package tui

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/existflow/irontask/internal/database"
	"github.com/existflow/irontask/internal/model"
	"github.com/existflow/irontask/internal/sync"
	"github.com/google/uuid"
)

// tickMsg is sent every second for time updates
type tickMsg time.Time

// syncRefreshMsg is sent when remote changes are pulled
type syncRefreshMsg struct{}

// conflictMsg is sent when conflicts are detected
type conflictMsg struct {
	conflicts []sync.ConflictItem
}

// Init initializes the model with a tick command
func (m Model) Init() tea.Cmd {
	return tea.Batch(tickCmd(), m.waitForSyncRefresh(), m.waitForSyncConflict())
}

func tickCmd() tea.Cmd {
	return tea.Every(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// waitForSyncRefresh listens for sync refresh signals
func (m Model) waitForSyncRefresh() tea.Cmd {
	if m.syncRefreshChan == nil {
		return nil
	}
	return func() tea.Msg {
		<-m.syncRefreshChan
		return syncRefreshMsg{}
	}
}

// waitForSyncConflict listens for sync conflict signals
func (m Model) waitForSyncConflict() tea.Cmd {
	if m.syncConflictChan == nil {
		return nil
	}
	return func() tea.Msg {
		conflicts := <-m.syncConflictChan
		return conflictMsg{conflicts: conflicts}
	}
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

	case syncRefreshMsg:
		// Remote changes were pulled - reload data
		m.loadData()
		m.message = "Synced from cloud"
		return m, m.waitForSyncRefresh()

	case conflictMsg:
		m.conflicts = msg.conflicts
		if len(m.conflicts) > 0 {
			m.mode = ModeConflict
			m.message = fmt.Sprintf("Conflict detected! (%d items)", len(m.conflicts))
		}
		return m, m.waitForSyncConflict()

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
		case ModeConflict:
			return m.handleConflictKeys(msg)
		case ModeHelp:
			m.mode = ModeNormal
			return m, nil
		}

		// Normal mode key handling
		return m.handleNormalKeys(msg)
	}

	return m, cmd
}

// handleNormalKeys handles key presses in normal mode
func (m Model) handleNormalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
		m.handleUp()

	case key.Matches(msg, keys.Down):
		m.handleDown()

	case msg.String() == "G":
		m.handleGoBottom()

	case msg.String() == "1", msg.String() == "2", msg.String() == "3", msg.String() == "4":
		m.handlePriority(msg.String())

	case key.Matches(msg, keys.Add):
		return m.startAddTask()

	case key.Matches(msg, keys.Project):
		return m.startAddProject()

	case key.Matches(msg, keys.Done), key.Matches(msg, keys.Enter):
		m.handleToggleDone()

	case key.Matches(msg, keys.Delete):
		m.handleDelete()

	case msg.String() == "e":
		return m.startEditTask()

	case msg.String() == "/":
		return m.startFilter()

	case msg.String() == "n":
		m.handleNextMatch()

	case msg.String() == "N":
		m.handlePrevMatch()

	case key.Matches(msg, keys.Escape):
		if m.filterText != "" {
			m.filterText = ""
			m.matchIndices = nil
			m.message = "Filter cleared"
		}

	case key.Matches(msg, keys.Help):
		m.mode = ModeHelp

	case key.Matches(msg, keys.Logout):
		m.handleLogout()

	case key.Matches(msg, keys.Refresh):
		m.handleRefresh()
	}

	return m, nil
}

func (m *Model) handleUp() {
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
}

func (m *Model) handleDown() {
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
}

func (m *Model) handleGoBottom() {
	if m.pane == PaneSidebar {
		m.projCursor = len(m.projects) - 1
		m.taskCursor = 0
		m.loadData()
	} else {
		m.taskCursor = len(m.tasks) - 1
	}
}

func (m *Model) handlePriority(key string) {
	if m.pane == PaneTaskList && len(m.tasks) > 0 {
		task := m.currentTask()
		if task != nil {
			priority := int(key[0] - '0')
			task.Priority = priority
			_ = m.db.UpdateTask(context.Background(), database.UpdateTaskParams{
				ID:        task.ID,
				ProjectID: task.ProjectID,
				Content:   task.Content,
				Status:    task.Status,
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
}

func (m Model) startAddTask() (tea.Model, tea.Cmd) {
	m.mode = ModeAddTask
	m.input.SetValue("")
	m.input.Placeholder = "Enter task..."
	m.input.Focus()
	return m, textinput.Blink
}

func (m Model) startAddProject() (tea.Model, tea.Cmd) {
	m.mode = ModeAddProject
	m.input.SetValue("")
	m.input.Placeholder = "Enter project name..."
	m.input.Focus()
	return m, textinput.Blink
}

func (m *Model) handleToggleDone() {
	if m.pane == PaneTaskList && len(m.tasks) > 0 {
		task := m.currentTask()
		if task != nil {
			newStatus := "done"
			if task.Status.String == "done" {
				newStatus = "process"
			}
			_ = m.db.UpdateTaskStatus(context.Background(), database.UpdateTaskStatusParams{
				ID:        task.ID,
				Status:    sql.NullString{String: newStatus, Valid: true},
				UpdatedAt: time.Now().Format(time.RFC3339),
			})

			if newStatus == "done" {
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
}

func (m *Model) handleDelete() {
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
}

func (m Model) startEditTask() (tea.Model, tea.Cmd) {
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
	return m, nil
}

func (m Model) startFilter() (tea.Model, tea.Cmd) {
	m.mode = ModeFilter
	m.input.SetValue(m.filterText)
	m.input.Placeholder = "/"
	m.input.Focus()
	return m, textinput.Blink
}

func (m *Model) handleNextMatch() {
	if len(m.matchIndices) > 0 {
		m.matchCursor = (m.matchCursor + 1) % len(m.matchIndices)
		m.taskCursor = m.matchIndices[m.matchCursor]
		m.message = fmt.Sprintf("[%d/%d] matches", m.matchCursor+1, len(m.matchIndices))
	}
}

func (m *Model) handlePrevMatch() {
	if len(m.matchIndices) > 0 {
		m.matchCursor--
		if m.matchCursor < 0 {
			m.matchCursor = len(m.matchIndices) - 1
		}
		m.taskCursor = m.matchIndices[m.matchCursor]
		m.message = fmt.Sprintf("[%d/%d] matches", m.matchCursor+1, len(m.matchIndices))
	}
}

func (m *Model) handleLogout() {
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

func (m *Model) handleRefresh() {
	if m.autoSync != nil {
		m.autoSync.TriggerSync()
	} else if m.syncClient == nil {
		m.message = "Not logged in - use 'irontask auth login' first"
	}
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
					ID:        uuid.New().String(),
					ProjectID: proj.ID,
					Content:   value,
					Status:    sql.NullString{String: "process", Valid: true},
					Priority:  model.PriorityLow,
					CreatedAt: now,
					UpdatedAt: now,
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
			// Generate slug from name (lowercase, replace spaces with dashes)
			slug := strings.ToLower(strings.ReplaceAll(value, " ", "-"))
			err := m.db.CreateProject(context.Background(), database.CreateProjectParams{
				ID:        uuid.New().String()[:8],
				Slug:      slug,
				Name:      value,
				Color:     sql.NullString{String: "#4ECDC4", Valid: true},
				CreatedAt: now,
				UpdatedAt: now,
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
					Status:    task.Status,
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
			ShowAll: true,
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

func (m *Model) handleConflictKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if len(m.conflicts) == 0 {
		m.mode = ModeNormal
		return m, nil
	}

	switch msg.String() {
	case "l", "L": // Keep Local
		m.resolveConflict(true)
	case "s", "S": // Keep Server
		m.resolveConflict(false)
	case "q", "esc":
		m.mode = ModeNormal
		// Clear conflicts if ignored
		m.conflicts = nil
		return m, nil
	}
	return m, nil
}

func (m *Model) resolveConflict(keepLocal bool) {
	if len(m.conflicts) == 0 {
		return
	}
	conflict := m.conflicts[0]

	if keepLocal {
		// Update local timestamp to force push next time
		ctx := context.Background()
		now := time.Now().Format(time.RFC3339)

		if conflict.Type == "task" {
			t, err := m.db.GetTask(ctx, conflict.ClientID)
			if err == nil {
				_ = m.db.UpdateTask(ctx, database.UpdateTaskParams{
					ID:        t.ID,
					ProjectID: t.ProjectID,
					Content:   t.Content,
					Status:    t.Status,
					Priority:  t.Priority,
					DueDate:   t.DueDate,
					Tags:      t.Tags,
					UpdatedAt: now,
				})
			}
		} else if conflict.Type == "project" {
			p, err := m.db.GetProject(ctx, conflict.ClientID)
			if err == nil {
				_ = m.db.UpdateProject(ctx, database.UpdateProjectParams{
					ID:        p.ID,
					Slug:      p.Slug,
					Name:      p.Name,
					Color:     p.Color,
					UpdatedAt: now,
				})
			}
		}
		m.message = "Keeping local version (will resync)"
	} else {
		// Apply server version
		m.applyServerItem(conflict.ServerData)
		m.message = "Applied server version"
	}

	// Remove resolved conflict
	m.conflicts = m.conflicts[1:]
	if len(m.conflicts) == 0 {
		m.mode = ModeNormal
		// Trigger sync if we kept local (to push the forced update)
		if keepLocal && m.autoSync != nil {
			m.autoSync.TriggerSync()
		}
		m.loadData()
	}
}

func (m *Model) applyServerItem(item sync.SyncItem) {
	ctx := context.Background()

	switch item.Type {
	case "project":
		name := item.Name
		color := "#4ECDC4"
		slug := item.Slug
		if slug == "" {
			slug = item.ClientID
		}
		if name == "" {
			data, err := base64.StdEncoding.DecodeString(item.EncryptedData)
			if err == nil {
				var p struct {
					Name  string `json:"name"`
					Color string `json:"color"`
				}
				_ = json.Unmarshal(data, &p)
				name = p.Name
				if p.Color != "" {
					color = p.Color
				}
			}
		}

		// Upsert project with server sync_version
		_, err := m.db.GetProject(ctx, item.ClientID)
		if err != nil {
			_ = m.db.CreateProject(ctx, database.CreateProjectParams{
				ID:        item.ClientID,
				Slug:      slug,
				Name:      name,
				Color:     sql.NullString{String: color, Valid: true},
				CreatedAt: time.Now().Format(time.RFC3339),
				UpdatedAt: time.Now().Format(time.RFC3339),
			})
			// Set sync_version from server
			_ = m.db.UpdateProjectSyncVersion(ctx, database.UpdateProjectSyncVersionParams{
				ID:          item.ClientID,
				SyncVersion: sql.NullInt64{Int64: item.SyncVersion, Valid: true},
			})
		} else {
			_ = m.db.OverwriteProject(ctx, database.OverwriteProjectParams{
				ID:          item.ClientID,
				Slug:        slug,
				Name:        name,
				Color:       sql.NullString{String: color, Valid: true},
				UpdatedAt:   time.Now().Format(time.RFC3339),
				SyncVersion: sql.NullInt64{Int64: item.SyncVersion, Valid: true},
			})
		}

	case "task":
		content := ""
		if item.EncryptedContent != "" {
			data, err := base64.StdEncoding.DecodeString(item.EncryptedContent)
			if err == nil {
				var c struct {
					Content string `json:"content"`
				}
				_ = json.Unmarshal(data, &c)
				content = c.Content
			}
		}

		status := item.Status
		if status == "" {
			status = "process"
		}

		// Upsert task with server sync_version
		_, err := m.db.GetTask(ctx, item.ClientID)
		if err != nil {
			_ = m.db.CreateTask(ctx, database.CreateTaskParams{
				ID:        item.ClientID,
				ProjectID: item.ProjectID,
				Content:   content,
				Status:    sql.NullString{String: status, Valid: true},
				Priority:  item.Priority,
				DueDate:   sql.NullString{String: item.DueDate, Valid: item.DueDate != ""},
				CreatedAt: time.Now().Format(time.RFC3339),
				UpdatedAt: time.Now().Format(time.RFC3339),
			})
			// Set sync_version from server
			_ = m.db.UpdateTaskSyncVersion(ctx, database.UpdateTaskSyncVersionParams{
				ID:          item.ClientID,
				SyncVersion: sql.NullInt64{Int64: item.SyncVersion, Valid: true},
			})
		} else {
			_ = m.db.OverwriteTask(ctx, database.OverwriteTaskParams{
				ID:          item.ClientID,
				ProjectID:   item.ProjectID,
				Content:     content,
				Status:      sql.NullString{String: status, Valid: true},
				Priority:    item.Priority,
				DueDate:     sql.NullString{String: item.DueDate, Valid: item.DueDate != ""},
				Tags:        sql.NullString{Valid: false}, // Empty tags for now
				UpdatedAt:   time.Now().Format(time.RFC3339),
				SyncVersion: sql.NullInt64{Int64: item.SyncVersion, Valid: true},
			})
		}
	}
}
