package tui

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/existflow/irontask/internal/database"
)

// View renders the UI
func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Build the layout
	sidebar := m.renderSidebar()
	taskList := m.renderTaskList()
	statusBar := m.renderStatusBar()

	// Combine sidebar and task list
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

	if m.mode == ModeConflict {
		modal := m.renderConflictModal()
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

	// Combine with status bar
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
		if t.Status.String != "done" {
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
		if t.Status.String == "done" {
			icon = "[x]"
			style = TaskDoneStyle
		} else if t.Status.String == "ignore" {
			icon = "[-]"
			style = TaskDoneStyle
		}

		content := truncate(t.Content, width-30)
		priority := FormatPriority(t.Priority)

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
	if m.autoSync != nil {
		if m.autoSync.IsPending() {
			syncMsg = "Syncing..."
		} else if err := m.autoSync.GetLastError(); err != nil {
			syncMsg = "Sync Error!"
			if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "Unauthorized") {
				syncMsg = "Auth Error (press L)"
			}
		}
	}

	if syncMsg != "" {
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
		title = "Filter Tasks"
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
	scope := "Current Project"
	if m.searchAll {
		scope = "All Projects"
	}

	// Header with scope toggle
	content += lipgloss.NewStyle().Bold(true).Foreground(Primary).Render("Search") + "  "
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
			icon := "[ ]"
			if t.Status.String == "done" {
				icon = "[x]"
			} else if t.Status.String == "ignore" {
				icon = "[-]"
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

func (m Model) renderConflictModal() string {
	if len(m.conflicts) == 0 {
		return ""
	}
	conflict := m.conflicts[0]
	modalWidth := 60

	content := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF5555")).Render("⚠ Sync Conflict Detected") + "\n\n"
	content += fmt.Sprintf("Conflict %d of %d\n\n", 1, len(m.conflicts))

	content += fmt.Sprintf("Item: %s (%s)\n", conflict.ClientID, conflict.Type)

	// Local version info
	localContent := "Unknown"
	if conflict.Type == "task" {
		// Try to find content in encoded data
		if conflict.ClientData.EncryptedContent != "" {
			data, _ := base64.StdEncoding.DecodeString(conflict.ClientData.EncryptedContent)
			var c struct {
				Content string `json:"content"`
			}
			if err := json.Unmarshal(data, &c); err == nil {
				localContent = c.Content
			}
		}
	} else {
		localContent = conflict.ClientData.Name
	}
	content += lipgloss.NewStyle().Bold(true).Render("Local Version:") + "\n"
	content += fmt.Sprintf("Last Modified: %s\n", conflict.ClientData.ClientUpdatedAt)
	content += fmt.Sprintf("Content: %s\n\n", truncate(localContent, modalWidth-10))

	// Server version info
	serverContent := "Unknown"
	if conflict.Type == "task" {
		if conflict.ServerData.EncryptedContent != "" {
			data, _ := base64.StdEncoding.DecodeString(conflict.ServerData.EncryptedContent)
			var c struct {
				Content string `json:"content"`
			}
			if err := json.Unmarshal(data, &c); err == nil {
				serverContent = c.Content
			}
		}
	} else {
		serverContent = conflict.ServerData.Name
	}
	content += lipgloss.NewStyle().Bold(true).Render("Server Version:") + "\n"
	// Server doesn't send updated_at explicitly in SyncItem, but SyncVersion roughly correlates
	content += fmt.Sprintf("Sync Version: %d\n", conflict.ServerData.SyncVersion)
	content += fmt.Sprintf("Content: %s\n\n", truncate(serverContent, modalWidth-10))

	content += lipgloss.NewStyle().Foreground(Border).Render(strings.Repeat("─", modalWidth-6)) + "\n\n"
	content += HelpStyle.Render("[L] Keep Local (Overwrite Server)") + "\n"
	content += HelpStyle.Render("[S] Keep Server (Overwrite Local)") + "\n"
	content += HelpStyle.Render("[Q] Ignore for now")

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
