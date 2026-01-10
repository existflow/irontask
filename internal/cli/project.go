package cli

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/existflow/irontask/internal/database"
	"github.com/existflow/irontask/internal/db"
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage projects",
	Long:  `Create, list, and manage projects for organizing tasks.`,
}

var projectNewCmd = &cobra.Command{
	Use:   "new [name]",
	Short: "Create a new project",
	Long: `Create a new project for organizing tasks.

Examples:
  irontask project new "Work"
  irontask project new "Personal" --color "#FF6B6B"`,
	Args: cobra.ExactArgs(1),
	RunE: runProjectNew,
}

var projectListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all projects",
	RunE:    runProjectList,
}

var projectDeleteCmd = &cobra.Command{
	Use:     "delete [project-id]",
	Aliases: []string{"rm"},
	Short:   "Delete a project",
	Args:    cobra.ExactArgs(1),
	RunE:    runProjectDelete,
}

var projectColor string

func init() {
	projectNewCmd.Flags().StringVarP(&projectColor, "color", "c", "#4ECDC4", "Project color (hex)")

	projectCmd.AddCommand(projectNewCmd)
	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectDeleteCmd)
}

func runProjectNew(cmd *cobra.Command, args []string) error {
	dbConn, err := db.OpenDefault()
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer dbConn.Close()

	name := args[0]
	id := strings.ToLower(strings.ReplaceAll(name, " ", "-"))

	// Check if ID already exists, if so use UUID
	if existing, _ := dbConn.GetProject(context.Background(), id); existing.ID != "" {
		id = uuid.New().String()[:8]
	}

	now := time.Now().Format(time.RFC3339)
	if err := dbConn.CreateProject(context.Background(), database.CreateProjectParams{
		ID:          id,
		Name:        name,
		Color:       sql.NullString{String: projectColor, Valid: true},
		CreatedAt:   now,
		UpdatedAt:   now,
		SyncVersion: sql.NullInt64{Int64: 0, Valid: true},
	}); err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}

	fmt.Printf("✓ Created project: %s (id: %s)\n", name, id)
	return nil
}

func runProjectList(cmd *cobra.Command, args []string) error {
	database, err := db.OpenDefault()
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	projects, err := database.ListProjects(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	if len(projects) == 0 {
		fmt.Println("No projects found.")
		return nil
	}

	fmt.Println()
	fmt.Printf("  %-15s  %-20s  %s\n", "ID", "Name", "Tasks")
	fmt.Println(strings.Repeat("─", 50))

	totalPending := 0
	for _, p := range projects {
		counts, _ := database.CountTasks(context.Background(), p.ID)
		pending := counts.Count
		total := counts.Count_2
		totalPending += int(pending)
		fmt.Printf("  %-15s  %-20s  %d/%d\n", p.ID, p.Name, pending, total)
	}

	fmt.Println(strings.Repeat("─", 50))
	fmt.Printf("  %d projects, %d pending tasks\n\n", len(projects), totalPending)

	return nil
}

func runProjectDelete(cmd *cobra.Command, args []string) error {
	dbConn, err := db.OpenDefault()
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer dbConn.Close()

	projectID := args[0]

	if projectID == "inbox" {
		return fmt.Errorf("cannot delete the Inbox project")
	}

	project, err := dbConn.GetProject(context.Background(), projectID)
	if err != nil {
		return fmt.Errorf("project not found: %s", projectID)
	}

	if err := dbConn.DeleteProject(context.Background(), database.DeleteProjectParams{
		ID:        projectID,
		DeletedAt: sql.NullString{String: time.Now().Format(time.RFC3339), Valid: true},
	}); err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	fmt.Printf("🗑️  Deleted project: %s\n", project.Name)
	return nil
}
