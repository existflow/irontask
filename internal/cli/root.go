package cli

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/existflow/irontask/internal/config"
	"github.com/existflow/irontask/internal/db"
	"github.com/existflow/irontask/internal/logger"
	"github.com/existflow/irontask/internal/tui"
	"github.com/spf13/cobra"
)

var (
	logLevel   string
	logFile    string
	logConsole bool
)

var rootCmd = &cobra.Command{
	Use:   "task",
	Short: "IronTask - Terminal todo app with sync",
	Long: `IronTask is a terminal-based todo application with project organization 
and cross-device sync capabilities.

Run 'task' without arguments to launch the interactive TUI.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Load config from file (or defaults if not exists)
		cfg, err := config.Load()
		if err != nil {
			logger.Warn("Failed to load config, using defaults", logger.F("error", err))
			cfg = config.DefaultConfig()
		}

		// Override with CLI flags if provided
		configChanged := false
		if cmd.Flags().Changed("log-level") {
			cfg.LogLevel = logLevel
			configChanged = true
		}
		if cmd.Flags().Changed("log-file") {
			cfg.LogFile = logFile
			configChanged = true
		}
		if cmd.Flags().Changed("log-console") {
			cfg.LogConsole = logConsole
			configChanged = true
		}

		// Save config if changed via CLI flags
		if configChanged {
			if err := cfg.Save(); err != nil {
				logger.Warn("Failed to save config", logger.F("error", err))
			}
		}

		logConfig := logger.Config{
			Level:      logger.ParseLevel(cfg.LogLevel),
			FilePath:   cfg.LogFile,
			MaxSize:    10 * 1024 * 1024, // 10MB
			MaxAge:     7,
			MaxBackups: 5,
			Console:    cfg.LogConsole,
		}

		if err := logger.Init(logConfig); err != nil {
			return fmt.Errorf("failed to initialize logger: %w", err)
		}

		logger.Info("IronTask started", logger.F("command", cmd.Name()))
		return nil
	},

	RunE: func(cmd *cobra.Command, args []string) error {
		// Launch TUI
		database, err := db.OpenDefault()
		if err != nil {
			logger.Error("Failed to open database", logger.F("error", err))
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer func() {
			_ = database.Close()
			logger.Info("Database closed")
		}()

		logger.Info("Launching TUI")
		m := tui.NewModel(database)
		p := tea.NewProgram(m, tea.WithAltScreen())

		if _, err := p.Run(); err != nil {
			logger.Error("TUI error", logger.F("error", err))
			return fmt.Errorf("failed to run TUI: %w", err)
		}

		logger.Info("TUI exited normally")
		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		logger.Info("IronTask exiting", logger.F("command", cmd.Name()))
		logger.Close()
	},
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Add logging flags
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "", "Log level (DEBUG, INFO, WARN, ERROR)")
	rootCmd.PersistentFlags().StringVar(&logFile, "log-file", "", "Path to log file")
	rootCmd.PersistentFlags().BoolVar(&logConsole, "log-console", false, "Enable console logging")

	// Add subcommands
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(doneCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(projectCmd)
	rootCmd.AddCommand(contextCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(clearCmd)
}
