package cli

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/existflow/irontask/internal/config"
	"github.com/existflow/irontask/internal/database"
	"github.com/existflow/irontask/internal/db"
)

var deleteCmd = &cobra.Command{
	Use:     "delete [task-id]",
	Aliases: []string{"rm"},
	Short:   "Delete a task",
	Long: `Delete a task by its ID.

Examples:
  irontask delete abc123
  task rm abc123`,
	Args: cobra.ExactArgs(1),
	RunE: runDelete,
}

func runDelete(cmd *cobra.Command, args []string) error {
	dbConn, err := db.OpenDefault()
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer dbConn.Close()

	taskID := args[0]
	ctx := context.Background()

	// Get task to show content (and handle partial match)
	var task database.Task
	if len(taskID) < 36 { // Assuming full UUID is 36 chars
		// Partial match logic needs correct SQLC method usage if available,
		// but standard GetTask uses exact match.
		// Previous implementation supported partials in internal/db/queries.go.
		// SQLC generated GetTaskPartial uses LIKE.
		task, err = dbConn.GetTaskPartial(ctx, sql.NullString{String: taskID, Valid: true})
	} else {
		task, err = dbConn.GetTask(ctx, taskID)
	}

	if err != nil {
		return fmt.Errorf("task not found: %s", taskID)
	}

	// Check config
	cfg, _ := config.Load() // Ignore error, use defaults
	if cfg.ConfirmDelete {
		fmt.Printf("About to delete: \"%s\" (ID: %s)\n", task.Content, task.ID)
		fmt.Print("Are you sure? [y/N]: ")
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "y" && confirm != "Y" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	if err := dbConn.DeleteTask(ctx, database.DeleteTaskParams{
		ID:        task.ID,
		DeletedAt: sql.NullString{String: time.Now().Format(time.RFC3339), Valid: true},
		UpdatedAt: time.Now().Format(time.RFC3339),
	}); err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	fmt.Printf("🗑️  Deleted: \"%s\"\n", task.Content)
	return nil
}
