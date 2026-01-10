package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/existflow/irontask/internal/db"
	"github.com/spf13/cobra"
)

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Manage project context",
	Long: `Set or view the current project context.

When a context is set, new tasks are added to that project by default.

Examples:
  irontask context              # Show current context
  irontask context ls           # List all projects
  irontask context set work     # Set context to 'work' project
  irontask context clear        # Clear context (use Inbox)`,
	RunE: runContextShow,
}

var contextLsCmd = &cobra.Command{
	Use:     "ls",
	Aliases: []string{"list"},
	Short:   "List all projects",
	RunE:    runContextList,
}

var contextSetCmd = &cobra.Command{
	Use:   "set [project-id]",
	Short: "Set the current project context",
	Args:  cobra.ExactArgs(1),
	RunE:  runContextSet,
}

var contextClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear the current context",
	RunE:  runContextClear,
}

func init() {
	contextCmd.AddCommand(contextLsCmd)
	contextCmd.AddCommand(contextSetCmd)
	contextCmd.AddCommand(contextClearCmd)
}

// Context file path
func contextFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".irontask", "context"), nil
}

// GetCurrentContext returns the current project context (empty means inbox)
func GetCurrentContext() string {
	path, err := contextFilePath()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// SetContext saves the current context
func SetContext(projectID string) error {
	path, err := contextFilePath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(projectID), 0644)
}

// ClearContext removes the context file
func ClearContext() error {
	path, err := contextFilePath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func runContextShow(cmd *cobra.Command, args []string) error {
	ctx := GetCurrentContext()
	if ctx == "" {
		fmt.Println("📥 Current context: Inbox (default)")
		return nil
	}

	database, err := db.OpenDefault()
	if err != nil {
		return err
	}
	defer func() {
		_ = database.Close()
	}()

	project, err := database.GetProject(context.Background(), ctx)
	if err != nil {
		fmt.Printf("⚠️  Context set to '%s' but project not found\n", ctx)
		return nil
	}

	counts, _ := database.CountTasks(context.Background(), ctx)
	fmt.Printf("📁 Current context: %s (%d/%d tasks)\n", project.Name, counts.Count, counts.Count_2)
	return nil
}

func runContextList(cmd *cobra.Command, args []string) error {
	database, err := db.OpenDefault()
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer func() {
		_ = database.Close()
	}()

	projects, err := database.ListProjects(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	currentCtx := GetCurrentContext()
	if currentCtx == "" {
		currentCtx = "inbox"
	}

	fmt.Println()
	for _, p := range projects {
		counts, _ := database.CountTasks(context.Background(), p.ID)
		marker := "  "
		if p.ID == currentCtx {
			marker = "❯ "
		}
		fmt.Printf("%s%-15s  %-20s  %d/%d\n", marker, p.ID, p.Name, counts.Count, counts.Count_2)
	}
	fmt.Println()
	fmt.Println("Use 'irontask context set <project-id>' to switch context")

	return nil
}

func runContextSet(cmd *cobra.Command, args []string) error {
	projectID := args[0]

	database, err := db.OpenDefault()
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer func() {
		_ = database.Close()
	}()

	project, err := database.GetProject(context.Background(), projectID)
	if err != nil {
		return fmt.Errorf("project not found: %s", projectID)
	}

	if err := SetContext(projectID); err != nil {
		return fmt.Errorf("failed to set context: %w", err)
	}

	fmt.Printf("📁 Switched to: %s\n", project.Name)
	return nil
}

func runContextClear(cmd *cobra.Command, args []string) error {
	if err := ClearContext(); err != nil {
		return fmt.Errorf("failed to clear context: %w", err)
	}
	fmt.Println("📥 Context cleared, using Inbox")
	return nil
}
