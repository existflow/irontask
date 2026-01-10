package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/tphuc/irontask/internal/sync"
	"golang.org/x/term"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
	Long:  `Manage authentication with the sync server.`,
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to the sync server",
	RunE:  runLogin,
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout from the sync server",
	RunE:  runLogout,
}

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Create a new account on the sync server",
	RunE:  runRegister,
}

func init() {
	authCmd.AddCommand(loginCmd)
	authCmd.AddCommand(logoutCmd)
	authCmd.AddCommand(registerCmd)

	loginCmd.Flags().String("email", "", "Login using magic link for this email")
	loginCmd.Flags().String("token", "", "Verify magic link token")
}

func runLogin(cmd *cobra.Command, args []string) error {
	client, err := sync.NewClient()
	if err != nil {
		return err
	}

	// Check for magic link flags
	email, _ := cmd.Flags().GetString("email")
	token, _ := cmd.Flags().GetString("token")

	if token != "" {
		fmt.Printf("🔄 Verifying magic link token...\n")
		if err := client.VerifyMagicLink(token); err != nil {
			return err
		}
		fmt.Println("✅ Logged in successfully!")
		return nil
	}

	if email != "" {
		fmt.Printf("🔄 Requesting magic link for %s...\n", email)
		token, err := client.RequestMagicLink(email)
		if err != nil {
			return err
		}
		fmt.Println("📬 Magic link requested! Check your email (or server logs in dev).")
		if token != "" {
			fmt.Printf("🔑 Development Token: %s\n", token)
		}

		// Prompt for token interactively
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter Magic Link Token: ")
		inputToken, _ := reader.ReadString('\n')
		inputToken = strings.TrimSpace(inputToken)

		if inputToken == "" {
			fmt.Println("❌ Token required.")
			return nil
		}

		fmt.Printf("🔄 Verifying magic link...\n")
		if err := client.VerifyMagicLink(inputToken); err != nil {
			return err
		}
		fmt.Println("✅ Logged in successfully!")
		return nil
	}

	// Normal password login
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

	fmt.Println("✅ Logged in successfully!")
	return nil
}

func runLogout(cmd *cobra.Command, args []string) error {
	client, err := sync.NewClient()
	if err != nil {
		return err
	}

	if !client.IsLoggedIn() {
		fmt.Println("Not logged in.")
		return nil
	}

	fmt.Println("🔄 Logging out...")
	if err := client.Logout(); err != nil {
		return err
	}

	fmt.Println("✅ Logged out successfully.")
	return nil
}

func runRegister(cmd *cobra.Command, args []string) error {
	client, err := sync.NewClient()
	if err != nil {
		return err
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

	fmt.Println("✅ Account created and logged in!")
	return nil
}
