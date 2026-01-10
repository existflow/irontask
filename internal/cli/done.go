package cli

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/tphuc/irontask/internal/database"
	"github.com/tphuc/irontask/internal/db"
)

var doneCmd = &cobra.Command{
	Use:   "done [task-id]",
	Short: "Mark a task as done",
	Long: `Mark a task as completed.

Examples:
  task done abc123
  task done abc123 --undo`,
	Args: cobra.ExactArgs(1),
	RunE: runDone,
}

var doneUndo bool

func init() {
	doneCmd.Flags().BoolVar(&doneUndo, "undo", false, "Mark task as not done")
}

func runDone(cmd *cobra.Command, args []string) error {
	dbConn, err := db.OpenDefault()
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer dbConn.Close()

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

	done := !doneUndo
	if err := dbConn.MarkTaskDone(ctx, database.MarkTaskDoneParams{
		ID:        task.ID,
		Done:      done,
		UpdatedAt: time.Now().Format(time.RFC3339),
	}); err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	if done {
		fmt.Printf("✓ Completed: \"%s\"\n", task.Content)
	} else {
		fmt.Printf("○ Reopened: \"%s\"\n", task.Content)
	}

	return nil
}
