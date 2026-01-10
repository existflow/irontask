package cli

import (
	"fmt"
	"strings"

	"github.com/existflow/irontask/internal/db"
	"github.com/existflow/irontask/internal/sync"
	"github.com/spf13/cobra"
)

var clearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear all tasks and projects",
	Long: `Clear all tasks and projects from the local database or/and the sync server.
By default, it only clears the local database unless --remote or --all is specified.`,
	RunE: runClear,
}

func init() {
	clearCmd.Flags().Bool("local", true, "Clear local data (default)")
	clearCmd.Flags().Bool("remote", false, "Clear remote data on the sync server")
	clearCmd.Flags().Bool("all", false, "Clear both local and remote data")
	clearCmd.Flags().Bool("force", false, "Do not ask for confirmation")
}

func runClear(cmd *cobra.Command, args []string) error {
	local, _ := cmd.Flags().GetBool("local")
	remote, _ := cmd.Flags().GetBool("remote")
	all, _ := cmd.Flags().GetBool("all")
	force, _ := cmd.Flags().GetBool("force")

	if all {
		local = true
		remote = true
	}

	if !force {
		fmt.Printf("Are you sure you want to clear data? (y/N): ")
		var response string
		_, _ = fmt.Scanln(&response)
		if strings.ToLower(response) != "y" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	dbConn, err := db.OpenDefault()
	if err != nil {
		return err
	}
	defer func() {
		_ = dbConn.Close()
	}()

	client, err := sync.NewClient()
	if err != nil {
		return err
	}

	if local {
		fmt.Println("🧹 Clearing local data...")
		if err := client.ClearLocal(dbConn); err != nil {
			return fmt.Errorf("failed to clear local data: %w", err)
		}
		fmt.Println("Local data cleared.")
	}

	if remote {
		if !client.IsLoggedIn() {
			fmt.Println("Skipping remote clear: not logged in.")
		} else {
			fmt.Println("🌐 Clearing remote data...")
			if err := client.ClearRemote(); err != nil {
				return fmt.Errorf("failed to clear remote data: %w", err)
			}
			fmt.Println("Remote data cleared.")
		}
	}

	return nil
}
