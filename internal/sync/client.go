package sync

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Config holds sync configuration
type Config struct {
	ServerURL     string `json:"server_url"`
	Token         string `json:"token"`
	UserID        string `json:"user_id"`
	LastSync      int64  `json:"last_sync"`
	EncryptionKey string `json:"encryption_key,omitempty"` // Base64 encoded
	Salt          string `json:"salt,omitempty"`           // Base64 encoded salt for key derivation
}

// Client is the sync client
type Client struct {
	config     *Config
	configPath string
	httpClient *http.Client
}

// NewClient creates a new sync client
func NewClient() (*Client, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(home, ".irontask", "sync.json")

	c := &Client{
		configPath: configPath,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}

	// Load existing config
	c.loadConfig()

	return c, nil
}

func (c *Client) loadConfig() {
	data, err := os.ReadFile(c.configPath)
	if err != nil {
		c.config = &Config{
			ServerURL: "http://localhost:8080",
		}
		return
	}

	c.config = &Config{}
	json.Unmarshal(data, c.config)
}

func (c *Client) saveConfig() error {
	dir := filepath.Dir(c.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c.config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(c.configPath, data, 0600)
}

// SetServer sets the sync server URL
func (c *Client) SetServer(url string) error {
	c.config.ServerURL = url
	return c.saveConfig()
}

// IsLoggedIn returns true if user is logged in
func (c *Client) IsLoggedIn() bool {
	return c.config.Token != ""
}

// Register creates a new account
func (c *Client) Register(username, email, password string) error {
	body, _ := json.Marshal(map[string]string{
		"username": username,
		"email":    email,
		"password": password,
	})

	resp, err := c.httpClient.Post(
		c.config.ServerURL+"/api/v1/register",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("register failed: %s", string(respBody))
	}

	var result struct {
		Token  string `json:"token"`
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	c.config.Token = result.Token
	c.config.UserID = result.UserID
	return c.saveConfig()
}

// Login authenticates with username and password
func (c *Client) Login(username, password string) error {
	body, _ := json.Marshal(map[string]string{
		"username": username,
		"password": password,
	})

	resp, err := c.httpClient.Post(
		c.config.ServerURL+"/api/v1/login",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("login failed: %s", string(respBody))
	}

	var result struct {
		Token  string `json:"token"`
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	c.config.Token = result.Token
	c.config.UserID = result.UserID
	return c.saveConfig()
}

// Logout clears the session
func (c *Client) Logout() error {
	c.config.Token = ""
	c.config.UserID = ""
	c.config.LastSync = 0
	return c.saveConfig()
}

// GetStatus returns current sync status
func (c *Client) GetStatus() (string, string, int64) {
	return c.config.ServerURL, c.config.UserID, c.config.LastSync
}

// GetEncryptionKey returns the display key (first 16 chars)
func (c *Client) GetEncryptionKey() string {
	return c.config.EncryptionKey
}

// GenerateEncryptionKey generates encryption key from password
func (c *Client) GenerateEncryptionKey(password string) (string, error) {
	salt, err := GenerateSalt()
	if err != nil {
		return "", err
	}

	// Store salt as base64
	c.config.Salt = base64.StdEncoding.EncodeToString(salt)

	// Generate display key
	c.config.EncryptionKey = DeriveKeyDisplay(password, salt)

	if err := c.saveConfig(); err != nil {
		return "", err
	}

	return c.config.EncryptionKey, nil
}

// GetCrypto returns a Crypto instance for encryption/decryption
func (c *Client) GetCrypto(password string) (*Crypto, error) {
	if c.config.Salt == "" {
		return nil, fmt.Errorf("no encryption key configured, run 'task sync key' first")
	}

	salt, err := base64.StdEncoding.DecodeString(c.config.Salt)
	if err != nil {
		return nil, err
	}

	return NewCrypto(password, salt), nil
}
