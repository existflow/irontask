package cli

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/existflow/irontask/internal/database"
	"github.com/existflow/irontask/internal/db"
	"github.com/spf13/cobra"
)

var doneCmd = &cobra.Command{
	Use:   "done [task-id]",
	Short: "Mark a task as done",
	Long: `Mark a task as completed.

Examples:
  irontask done abc123
  irontask done abc123 --undo`,
	Args: cobra.ExactArgs(1),
	RunE: runDone,
}

var (
	doneUndo bool
	doneSync bool
)

func init() {
	doneCmd.Flags().BoolVar(&doneUndo, "undo", false, "Mark task as not done")
	doneCmd.Flags().BoolVarP(&doneSync, "sync", "s", false, "Sync with server after marking done")
}

func runDone(cmd *cobra.Command, args []string) error {
	dbConn, err := db.OpenDefault()
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer func() {
		_ = dbConn.Close()
	}()

	taskID := args[0]
	ctx := context.Background()

	// Get task to show content (and handle partial match logic if needed)
	var task database.Task
	if len(taskID) < 36 {
		task, err = dbConn.GetTaskPartial(ctx, sql.NullString{String: taskID, Valid: true})
	} else {
		task, err = dbConn.GetTask(ctx, taskID)
	}

	if err != nil {
		return fmt.Errorf("task not found: %s", taskID)
	}

	newStatus := "done"
	if doneUndo {
		newStatus = "process"
	}
	if err := dbConn.UpdateTaskStatus(ctx, database.UpdateTaskStatusParams{
		ID:        task.ID,
		Status:    sql.NullString{String: newStatus, Valid: true},
		UpdatedAt: time.Now().Format(time.RFC3339),
	}); err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	if newStatus == "done" {
		fmt.Printf("[OK] Completed: \"%s\"\n", task.Content)
	} else {
		fmt.Printf("[OK] Reopened: \"%s\"\n", task.Content)
	}

	// Sync after change if flag is set or auto-sync is due
	MaybeSyncAfterChange(dbConn, doneSync)

	return nil
}
