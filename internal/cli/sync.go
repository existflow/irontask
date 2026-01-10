package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/existflow/irontask/internal/db"
	"github.com/existflow/irontask/internal/sync"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync tasks with server",
	Long: `Sync your tasks across devices.

Commands:
  irontask sync              # Sync now
  irontask sync status       # Show sync status`,
	RunE: runSync,
}

var syncStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show sync status",
	RunE:  runSyncStatus,
}
var syncKeyCmd = &cobra.Command{
	Use:   "key",
	Short: "Generate or show encryption key",
	RunE:  runSyncKey,
}

var syncConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure sync settings",
	RunE:  runSyncConfig,
}

func init() {
	syncCmd.AddCommand(syncStatusCmd)
	syncCmd.AddCommand(syncKeyCmd)
	syncCmd.AddCommand(syncConfigCmd)

	syncCmd.Flags().Bool("pull", false, "Force sync from remote (replaces local)")
	syncCmd.Flags().Bool("push", false, "Force sync from local (replaces remote)")

	syncConfigCmd.Flags().String("server", "", "Set server URL")
	syncConfigCmd.Flags().Bool("insecure", false, "Allow insecure (HTTP) connection")
}

func runSync(cmd *cobra.Command, args []string) error {
	client, err := sync.NewClient()
	if err != nil {
		return err
	}

	database, err := db.OpenDefault()
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	mode := sync.SyncModeMerge
	pull, _ := cmd.Flags().GetBool("pull")
	push, _ := cmd.Flags().GetBool("push")

	if pull && push {
		return fmt.Errorf("cannot use both --pull and --push")
	}

	if pull {
		mode = sync.SyncModeRemoteToLocal
		fmt.Println("⚠️  Forcing sync from remote (replacing local data)...")
	} else if push {
		mode = sync.SyncModeLocalToRemote
		fmt.Println("⚠️  Forcing sync from local (replacing remote data)...")
	} else {
		fmt.Println("🔄 Synchronizing...")
	}

	result, err := client.Sync(database, mode)
	if err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	fmt.Printf("✓ Sync complete! Pushed: %d, Pulled: %d\n", result.Pushed, result.Pulled)
	return nil
}

func runSyncStatus(cmd *cobra.Command, args []string) error {
	client, err := sync.NewClient()
	if err != nil {
		return err
	}

	serverURL, userID, lastSync := client.GetStatus()

	fmt.Printf("Server:    %s\n", serverURL)
	if client.IsLoggedIn() {
		fmt.Printf("User ID:   %s\n", userID)
		fmt.Printf("Last Sync: %d\n", lastSync)
		fmt.Println("Status:    ✓ Logged in")
	} else {
		fmt.Println("Status:    Not logged in")
	}

	return nil
}

func runSyncKey(cmd *cobra.Command, args []string) error {
	client, err := sync.NewClient()
	if err != nil {
		return err
	}

	key := client.GetEncryptionKey()
	if key == "" {
		// Generate new key
		fmt.Print("Enter encryption password: ")
		reader := bufio.NewReader(os.Stdin)
		password, _ := reader.ReadString('\n')
		password = strings.TrimSpace(password)

		if len(password) < 8 {
			return fmt.Errorf("password must be at least 8 characters")
		}

		key, err = client.GenerateEncryptionKey(password)
		if err != nil {
			return err
		}
		fmt.Println("\n✓ Encryption key generated!")
		fmt.Println("\n⚠️  IMPORTANT: Save this key! You need it to decrypt on other devices.")
	}

	fmt.Printf("\nEncryption Key: %s\n", key)
	return nil
}

func runSyncConfig(cmd *cobra.Command, args []string) error {
	client, err := sync.NewClient()
	if err != nil {
		return err
	}

	server, _ := cmd.Flags().GetString("server")
	if server != "" {
		if err := client.SetServer(server); err != nil {
			return err
		}
		fmt.Printf("✓ Server set to: %s\n", server)
	} else {
		// Just show config
		url, _, _ := client.GetStatus()
		fmt.Printf("Server: %s\n", url)
	}

	return nil
}
