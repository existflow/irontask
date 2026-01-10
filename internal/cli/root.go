package cli

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/tphuc/irontask/internal/db"
	"github.com/tphuc/irontask/internal/tui"
)

var rootCmd = &cobra.Command{
	Use:   "task",
	Short: "IronTask - Terminal todo app with sync",
	Long: `IronTask is a terminal-based todo application with project organization 
and cross-device sync capabilities.

Run 'task' without arguments to launch the interactive TUI.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Launch TUI
		database, err := db.OpenDefault()
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer database.Close()

		m := tui.NewModel(database)
		p := tea.NewProgram(m, tea.WithAltScreen())

		if _, err := p.Run(); err != nil {
			return fmt.Errorf("failed to run TUI: %w", err)
		}
		return nil
	},
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Add subcommands
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(doneCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(projectCmd)
	rootCmd.AddCommand(contextCmd)
	rootCmd.AddCommand(syncCmd)
}
