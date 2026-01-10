package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/existflow/irontask/internal/database"
	"github.com/existflow/irontask/internal/db"
	"github.com/existflow/irontask/internal/model"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List tasks",
	Long: `List tasks, optionally filtered by project.

Examples:
  irontask list
  irontask list --project work
  irontask list --all`,
	RunE: runList,
}

var (
	listProject     string
	listAll         bool
	listIncludeDone bool
	listSync        bool
)

func init() {
	listCmd.Flags().StringVarP(&listProject, "project", "P", "", "Filter by project")
	listCmd.Flags().BoolVarP(&listAll, "all", "a", false, "Show all projects")
	listCmd.Flags().BoolVar(&listIncludeDone, "done", false, "Include completed tasks")
	listCmd.Flags().BoolVarP(&listSync, "sync", "s", false, "Sync with server before listing")
}

func runList(cmd *cobra.Command, args []string) error {
	dbConn, err := db.OpenDefault()
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer func() {
		_ = dbConn.Close()
	}()

	// Sync before listing if flag is set or auto-sync is due
	MaybeSyncCLI(dbConn, listSync)

	var projectID interface{}
	if listProject != "" {
		projectID = listProject
	}

	tasks, err := dbConn.ListTasks(context.Background(), database.ListTasksParams{
		ProjectID: projectID,
		ShowAll:   listIncludeDone,
	})
	if err != nil {
		return fmt.Errorf("failed to list tasks: %w", err)
	}

	if len(tasks) == 0 {
		fmt.Println("No tasks found. Add one with: irontask add \"Your task\"")
		return nil
	}

	// Group by project if listing all
	if listProject == "" {
		printTasksByProject(dbConn, tasks)
	} else {
		project, _ := dbConn.GetProject(context.Background(), listProject)
		name := listProject
		if project.Name != "" {
			name = project.Name
		}
		printTasks(name, tasks)
	}

	return nil
}

func printTasks(projectName string, tasks []database.Task) {
	pending := 0
	for _, t := range tasks {
		if t.Status.String != "done" {
			pending++
		}
	}

	fmt.Printf("\n📁 %s (%d pending)\n", projectName, pending)
	fmt.Println(strings.Repeat("─", 60))

	for i, t := range tasks {
		printTask(i+1, t)
	}
	fmt.Println()
}

func printTasksByProject(db *db.DB, tasks []database.Task) {
	// Group tasks by project
	byProject := make(map[string][]database.Task)
	for _, t := range tasks {
		byProject[t.ProjectID] = append(byProject[t.ProjectID], t)
	}

	for projectID, projectTasks := range byProject {
		project, _ := db.GetProject(context.Background(), projectID)
		name := projectID
		if project.Name != "" {
			name = project.Name
		}
		printTasks(name, projectTasks)
	}
}

func printTask(num int, t database.Task) {
	// Status icon
	icon := "[ ]"
	if t.Status.String == "done" {
		icon = "[x]"
	} else if t.Status.String == "ignore" {
		icon = "[-]"
	}

	// Priority indicator
	priority := fmt.Sprintf("P%d", t.Priority)
	switch t.Priority {
	case model.PriorityUrgent:
		priority = "▲ P1"
	case model.PriorityHigh:
		priority = "▲ P2"
	case model.PriorityMedium:
		priority = "  P3"
	case model.PriorityLow:
		priority = "  P4"
	}

	// Due date
	due := ""
	if t.DueDate.Valid {
		parsed, _ := time.Parse(time.RFC3339, t.DueDate.String)
		due = parsed.Format("Jan 2")
		if parsed.Before(time.Now()) {
			due = "" + due
		}
	}

	// Truncate content if too long
	content := t.Content
	if len(content) > 40 {
		content = content[:37] + "..."
	}

	// Short ID
	shortID := t.ID
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}

	fmt.Printf("  %s  %-8s  %-40s  %-10s  %s\n", icon, shortID, content, due, priority)
}
