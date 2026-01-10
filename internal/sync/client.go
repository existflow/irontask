package sync

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/existflow/irontask/internal/db"
)

// Config holds sync configuration
type Config struct {
	ServerURL     string `json:"server_url"`
	Token         string `json:"token"`
	UserID        string `json:"user_id"`
	LastSync      int64  `json:"last_sync"`
	LastSyncTime  int64  `json:"last_sync_time"` // Unix timestamp of last sync
	HasSyncedOnce bool   `json:"has_synced_once"`
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
	defaultServer := os.Getenv("DEFAULT_SERVER_URL")
	if defaultServer == "" {
		defaultServer = "https://irontask-server-dev.onrender.com"
	}

	data, err := os.ReadFile(c.configPath)
	if err != nil {
		c.config = &Config{
			ServerURL: defaultServer,
		}
		return
	}

	c.config = &Config{}
	_ = json.Unmarshal(data, c.config)

	// If loaded config has empty URL (unlikely but safe), apply default
	if c.config.ServerURL == "" {
		c.config.ServerURL = defaultServer
	}
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

// CanAutoSync returns true if auto-sync is allowed (logged in AND has synced once)
func (c *Client) CanAutoSync() bool {
	return c.IsLoggedIn() && c.config.HasSyncedOnce
}

// SetSyncedOnce marks that the user has completed a full sync
func (c *Client) SetSyncedOnce() error {
	c.config.HasSyncedOnce = true
	return c.saveConfig()
}

// ShouldAutoSync returns true if auto-sync is due (every 12 hours)
func (c *Client) ShouldAutoSync() bool {
	if !c.IsLoggedIn() {
		return false
	}
	twelveHours := int64(12 * 60 * 60)
	return time.Now().Unix()-c.config.LastSyncTime > twelveHours
}

// UpdateSyncTime updates the last sync timestamp
func (c *Client) UpdateSyncTime() error {
	c.config.LastSyncTime = time.Now().Unix()
	return c.saveConfig()
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
	defer func() {
		_ = resp.Body.Close()
	}()

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
	c.config.HasSyncedOnce = false
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
	defer func() {
		_ = resp.Body.Close()
	}()

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
	c.config.HasSyncedOnce = false
	return c.saveConfig()
}

// RequestMagicLink requests a login link via email
func (c *Client) RequestMagicLink(email string) (string, error) {
	body, _ := json.Marshal(map[string]string{
		"email": email,
	})

	resp, err := c.httpClient.Post(
		c.config.ServerURL+"/api/v1/magic-link",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return "", fmt.Errorf("failed to connect: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("request failed: %s", string(respBody))
	}

	var result struct {
		Token string `json:"token"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&result)

	return result.Token, nil
}

// VerifyMagicLink verifies the token and logs in
func (c *Client) VerifyMagicLink(token string) error {
	resp, err := c.httpClient.Get(
		c.config.ServerURL + "/api/v1/magic-link/" + token,
	)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("verification failed: %s", string(respBody))
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
	c.config.HasSyncedOnce = false
	return c.saveConfig()
}

// Logout clears the session
func (c *Client) Logout() error {
	if c.config.Token != "" {
		// Call server logout
		req, err := http.NewRequest("POST", c.config.ServerURL+"/api/v1/logout", nil)
		if err == nil {
			req.Header.Set("Authorization", "Bearer "+c.config.Token)
			_, _ = c.httpClient.Do(req) // We don't care about the response/error much here, just best effort
		}
	}

	c.config.Token = ""
	c.config.UserID = ""
	c.config.LastSync = 0
	c.config.HasSyncedOnce = false
	return c.saveConfig()
}

// ClearLocal wipes all local data
func (c *Client) ClearLocal(dbConn *db.DB) error {
	ctx := context.Background()
	if err := dbConn.ClearTasks(ctx); err != nil {
		return err
	}
	if err := dbConn.ClearProjects(ctx); err != nil {
		return err
	}
	return nil
}

// ClearRemote wipes all remote data
func (c *Client) ClearRemote() error {
	if !c.IsLoggedIn() {
		return fmt.Errorf("not logged in")
	}

	req, err := http.NewRequest("POST", c.config.ServerURL+"/api/v1/clear", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.config.Token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("remote clear failed: %s", string(body))
	}

	return nil
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
		return nil, fmt.Errorf("no encryption key configured, run 'irontask sync key' first")
	}

	salt, err := base64.StdEncoding.DecodeString(c.config.Salt)
	if err != nil {
		return nil, err
	}

	return NewCrypto(password, salt), nil
}
