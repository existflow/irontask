package cli

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/existflow/irontask/internal/database"
	"github.com/existflow/irontask/internal/db"
	"github.com/existflow/irontask/internal/model"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add [content]",
	Short: "Add a new task",
	Long: `Add a new task to a project.

Examples:
  irontask add "Buy groceries"
  irontask add "Meeting with team" -p 1
  irontask add "Feature work" --project work -p 2`,
	Args: cobra.MinimumNArgs(1),
	RunE: runAdd,
}

var (
	addProject  string
	addPriority int
	addDue      string
	addSync     bool
)

func init() {
	addCmd.Flags().StringVarP(&addProject, "project", "P", "inbox", "Project to add task to")
	addCmd.Flags().IntVarP(&addPriority, "priority", "p", 4, "Priority (1=urgent, 4=low)")
	addCmd.Flags().StringVarP(&addDue, "due", "d", "", "Due date (e.g., 'tomorrow', '2024-01-15')")
	addCmd.Flags().BoolVarP(&addSync, "sync", "s", false, "Sync with server after adding")
}

func runAdd(cmd *cobra.Command, args []string) error {
	dbConn, err := db.OpenDefault()
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer func() {
		_ = dbConn.Close()
	}()

	content := args[0]
	if len(args) > 1 {
		for _, arg := range args[1:] {
			content += " " + arg
		}
	}

	// Use context if no project specified
	projectID := addProject
	if !cmd.Flags().Changed("project") {
		ctx := GetCurrentContext()
		if ctx != "" {
			projectID = ctx
		}
	}

	// Validate priority
	if addPriority < 1 || addPriority > 4 {
		addPriority = model.PriorityLow
	}

	// Create task
	now := time.Now().Format(time.RFC3339)
	err = dbConn.CreateTask(context.Background(), database.CreateTaskParams{
		ID:          uuid.New().String(),
		ProjectID:   projectID,
		Content:     content,
		Status:      sql.NullString{String: "process", Valid: true},
		Priority:    addPriority,
		CreatedAt:   now,
		UpdatedAt:   now,
		SyncVersion: sql.NullInt64{Int64: 1, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	// Get project name for display
	project, _ := dbConn.GetProject(context.Background(), projectID)
	projectName := projectID
	if project.Name != "" {
		projectName = project.Name
	}

	fmt.Printf("[OK] Added to [%s]: \"%s\" (P%d)\n", projectName, content, addPriority)

	// Sync after change if flag is set or auto-sync is due
	MaybeSyncAfterChange(dbConn, addSync)

	return nil
}
