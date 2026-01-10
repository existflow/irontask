package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/tphuc/irontask/internal/db"
	"github.com/tphuc/irontask/internal/sync"
	"golang.org/x/term"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync tasks with server",
	Long: `Sync your tasks across devices.

Commands:
  task sync              # Sync now
  task sync register     # Create account
  task sync login        # Login to existing account
  task sync logout       # Logout
  task sync status       # Show sync status`,
	RunE: runSync,
}

var syncRegisterCmd = &cobra.Command{
	Use:   "register",
	Short: "Create a new account",
	RunE:  runSyncRegister,
}

var syncLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to existing account",
	RunE:  runSyncLogin,
}

var syncLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout and clear credentials",
	RunE:  runSyncLogout,
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
	syncCmd.AddCommand(syncRegisterCmd)
	syncCmd.AddCommand(syncLoginCmd)
	syncCmd.AddCommand(syncLogoutCmd)
	syncCmd.AddCommand(syncStatusCmd)
	syncCmd.AddCommand(syncKeyCmd)
	syncCmd.AddCommand(syncConfigCmd)

	syncRegisterCmd.Flags().String("server", "", "Server URL (default: http://localhost:8080)")
	syncConfigCmd.Flags().String("server", "", "Set server URL")
	syncConfigCmd.Flags().Bool("insecure", false, "Allow insecure (HTTP) connection")
}

func runSync(cmd *cobra.Command, args []string) error {
	client, err := sync.NewClient()
	if err != nil {
		return err
	}

	if !client.IsLoggedIn() {
		fmt.Println("Not logged in. Run 'task sync login' or 'task sync register' first.")
		return nil
	}

	// Open database
	database, err := db.OpenDefault()
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	fmt.Println("🔄 Syncing...")
	result, err := client.Sync(database)
	if err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	fmt.Printf("✓ Sync complete! Pushed: %d, Pulled: %d\n", result.Pushed, result.Pulled)
	return nil
}

func runSyncRegister(cmd *cobra.Command, args []string) error {
	client, err := sync.NewClient()
	if err != nil {
		return err
	}

	// Check for server flag
	server, _ := cmd.Flags().GetString("server")
	if server != "" {
		if err := client.SetServer(server); err != nil {
			return err
		}
		fmt.Printf("✓ Server set to: %s\n", server)
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Username: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)

	fmt.Print("Email: ")
	email, _ := reader.ReadString('\n')
	email = strings.TrimSpace(email)

	fmt.Print("Password: ")
	passwordBytes, _ := term.ReadPassword(int(syscall.Stdin))
	password := string(passwordBytes)
	fmt.Println()

	fmt.Print("Confirm Password: ")
	confirmBytes, _ := term.ReadPassword(int(syscall.Stdin))
	confirm := string(confirmBytes)
	fmt.Println()

	if password != confirm {
		return fmt.Errorf("passwords do not match")
	}

	fmt.Println("🔄 Creating account...")
	if err := client.Register(username, email, password); err != nil {
		return err
	}

	fmt.Println("✓ Account created! You are now logged in.")
	return nil
}

func runSyncLogin(cmd *cobra.Command, args []string) error {
	client, err := sync.NewClient()
	if err != nil {
		return err
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Username: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)

	fmt.Print("Password: ")
	passwordBytes, _ := term.ReadPassword(int(syscall.Stdin))
	password := string(passwordBytes)
	fmt.Println()

	fmt.Println("🔄 Logging in...")
	if err := client.Login(username, password); err != nil {
		return err
	}

	fmt.Println("✓ Logged in successfully!")
	return nil
}

func runSyncLogout(cmd *cobra.Command, args []string) error {
	client, err := sync.NewClient()
	if err != nil {
		return err
	}

	if err := client.Logout(); err != nil {
		return err
	}

	fmt.Println("✓ Logged out")
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
