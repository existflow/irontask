package sync

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/existflow/irontask/internal/database"
	"github.com/existflow/irontask/internal/db"
	"github.com/existflow/irontask/internal/logger"
)

// SyncItem represents an item to sync
type SyncItem struct {
	ID               string `json:"id"`
	ClientID         string `json:"client_id"`
	Type             string `json:"type"` // "project" or "task"
	Slug             string `json:"slug,omitempty"`
	Name             string `json:"name,omitempty"`
	ProjectID        string `json:"project_id,omitempty"`
	EncryptedData    string `json:"encrypted_data,omitempty"`    // For projects (legacy)
	EncryptedContent string `json:"encrypted_content,omitempty"` // For tasks (content only)
	Status           string `json:"status,omitempty"`
	Priority         int    `json:"priority,omitempty"`
	DueDate          string `json:"due_date,omitempty"`
	SyncVersion      int64  `json:"sync_version"`
	Deleted          bool   `json:"deleted"`
	ClientUpdatedAt  string `json:"client_updated_at,omitempty"` // Client timestamp for conflict detection
}

// ConflictItem represents a conflicting item
type ConflictItem struct {
	ClientID      string   `json:"client_id"`
	Type          string   `json:"type"`
	ServerVersion int64    `json:"server_version"`
	ServerData    SyncItem `json:"server_data"`
	ClientData    SyncItem `json:"client_data"`
}

// SyncPullResponse is the response from pull
type SyncPullResponse struct {
	Items       []SyncItem `json:"items"`
	SyncVersion int64      `json:"sync_version"`
}

// SyncPushResponse is the response from push
type SyncPushResponse struct {
	Updated   []SyncItem     `json:"updated"`
	Conflicts []ConflictItem `json:"conflicts,omitempty"`
}

// SyncResult holds sync statistics
type SyncResult struct {
	Pushed    int
	Pulled    int
	Conflicts []ConflictItem
}

// SyncMode defines how the sync should be performed
type SyncMode int

const (
	SyncModeMerge         SyncMode = iota // Default: Push local, then pull remote
	SyncModeRemoteToLocal                 // Wipe local, then pull all from remote
	SyncModeLocalToRemote                 // Wipe remote, then push all from local
)

// Sync performs sync with server based on the specified mode
func (c *Client) Sync(database *db.DB, mode SyncMode) (*SyncResult, error) {
	if !c.IsLoggedIn() {
		return nil, fmt.Errorf("not logged in")
	}

	result := &SyncResult{}

	switch mode {
	case SyncModeRemoteToLocal:
		// 1. Wipe local data
		if err := c.ClearLocal(database); err != nil {
			return nil, fmt.Errorf("failed to clear local data: %w", err)
		}
		// 2. Clear last sync version to pull everything
		c.config.LastSync = 0
		_ = c.saveConfig()

		// 3. Pull remote changes
		pulled, err := c.pullChanges(database)
		if err != nil {
			return nil, fmt.Errorf("pull failed: %w", err)
		}
		result.Pulled = pulled

	case SyncModeLocalToRemote:
		// 1. Wipe remote data
		if err := c.ClearRemote(); err != nil {
			return nil, fmt.Errorf("failed to clear remote data: %w", err)
		}
		// 2. Push local changes
		pushed, conflicts, err := c.pushChanges(database)
		if err != nil {
			return nil, fmt.Errorf("push failed: %w", err)
		}
		result.Pushed = pushed
		result.Conflicts = conflicts

	default: // SyncModeMerge
		// 1. Push local changes
		pushed, conflicts, err := c.pushChanges(database)
		if err != nil {
			return nil, fmt.Errorf("push failed: %w", err)
		}
		result.Pushed = pushed
		result.Conflicts = conflicts

		// 2. Pull remote changes
		pulled, err := c.pullChanges(database)
		if err != nil {
			return nil, fmt.Errorf("pull failed: %w", err)
		}
		result.Pulled = pulled
	}

	// Mark as synced once after first successful sync
	if !c.config.HasSyncedOnce {
		_ = c.SetSyncedOnce()
	}

	return result, nil
}

// pushChanges sends local changes to server
func (c *Client) pushChanges(dbConn *db.DB) (int, []ConflictItem, error) {
	logger.Debug("Starting push changes")
	var items []SyncItem

	// Get projects that need syncing (sync_version is NULL means dirty)
	projects, _ := dbConn.GetProjectsToSync(context.Background())

	logger.Debug("Found projects to sync", logger.F("count", len(projects)), logger.F("lastSync", c.config.LastSync),
		logger.F("projects", projects),
	)
	for _, p := range projects {
		logger.Debug("Processing project for sync", logger.F("id", p.ID), logger.F("name", p.Name))
		// Prepare data (including legacy color info)
		color := ""
		if p.Color.Valid {
			color = p.Color.String
		}
		data, _ := json.Marshal(map[string]interface{}{
			"name":  p.Name,
			"color": color,
		})

		items = append(items, SyncItem{
			ClientID:        p.ID,
			Type:            "project",
			Slug:            p.Slug,
			Name:            p.Name,
			EncryptedData:   base64.StdEncoding.EncodeToString(data),
			SyncVersion:     p.SyncVersion.Int64,
			Deleted:         p.DeletedAt.Valid,
			ClientUpdatedAt: p.UpdatedAt, // Send client timestamp for conflict detection
		})
	}

	// Get tasks that need syncing (sync_version is NULL means dirty)
	tasks, _ := dbConn.GetTasksToSync(context.Background())
	for _, t := range tasks {
		dueDate := ""
		if t.DueDate.Valid {
			dueDate = t.DueDate.String
		}
		status := "process"
		if t.Status.Valid {
			status = t.Status.String
		}

		// Only encrypt content
		contentData, _ := json.Marshal(map[string]interface{}{
			"content": t.Content,
		})

		items = append(items, SyncItem{
			ClientID:         t.ID,
			Type:             "task",
			ProjectID:        t.ProjectID,
			EncryptedContent: base64.StdEncoding.EncodeToString(contentData),
			Status:           status,
			Priority:         t.Priority,
			DueDate:          dueDate,
			SyncVersion:      t.SyncVersion.Int64,
			Deleted:          t.DeletedAt.Valid,
			ClientUpdatedAt:  t.UpdatedAt, // Send client timestamp for conflict detection
		})
	}

	if len(items) == 0 {
		logger.Debug("No items to push")
		return 0, nil, nil
	}

	logger.Info("Pushing changes to server", logger.F("itemCount", len(items)))

	// Send to server
	body, _ := json.Marshal(map[string]interface{}{
		"items": items,
	})

	url := c.config.ServerURL + "/api/v1/sync"
	logger.Debug("HTTP Request",
		logger.F("method", "POST"),
		logger.F("url", url),
		logger.F("bodySize", len(body)))

	req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.Token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		logger.Error("HTTP request failed", logger.F("error", err), logger.F("url", url))
		return 0, nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	logger.Debug("HTTP Response",
		logger.F("status", resp.StatusCode),
		logger.F("statusText", resp.Status))

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		logger.Error("Push failed",
			logger.F("status", resp.StatusCode),
			logger.F("response", string(respBody)))
		return 0, nil, fmt.Errorf("server error: %s", string(respBody))
	}

	var result SyncPushResponse
	_ = json.NewDecoder(resp.Body).Decode(&result)

	logger.Info("Push completed",
		logger.F("updated", len(result.Updated)),
		logger.F("conflicts", len(result.Conflicts)))

	// Update local sync_version with server-assigned values
	ctx := context.Background()
	for _, item := range result.Updated {
		if item.Type == "project" {
			_ = dbConn.UpdateProjectSyncVersion(ctx, database.UpdateProjectSyncVersionParams{
				ID:          item.ClientID,
				SyncVersion: sql.NullInt64{Int64: item.SyncVersion, Valid: true},
			})
			logger.Debug("Updated project sync_version", logger.F("id", item.ClientID), logger.F("version", item.SyncVersion))
		} else if item.Type == "task" {
			_ = dbConn.UpdateTaskSyncVersion(ctx, database.UpdateTaskSyncVersionParams{
				ID:          item.ClientID,
				SyncVersion: sql.NullInt64{Int64: item.SyncVersion, Valid: true},
			})
			logger.Debug("Updated task sync_version", logger.F("id", item.ClientID), logger.F("version", item.SyncVersion))
		}
	}

	return len(result.Updated), result.Conflicts, nil
}

// pullChanges gets remote changes from server
func (c *Client) pullChanges(dbConn *db.DB) (int, error) {
	url := fmt.Sprintf("%s/api/v1/sync?since=%d", c.config.ServerURL, c.config.LastSync)

	logger.Debug("Pulling changes from server", logger.F("since", c.config.LastSync))
	logger.Debug("HTTP Request",
		logger.F("method", "GET"),
		logger.F("url", url))

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+c.config.Token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		logger.Error("HTTP request failed", logger.F("error", err), logger.F("url", url))
		return 0, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	logger.Debug("HTTP Response",
		logger.F("status", resp.StatusCode),
		logger.F("statusText", resp.Status))

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		logger.Error("Pull failed",
			logger.F("status", resp.StatusCode),
			logger.F("response", string(respBody)))
		return 0, fmt.Errorf("server error: %s", string(respBody))
	}

	var result SyncPullResponse
	_ = json.NewDecoder(resp.Body).Decode(&result)

	logger.Info("Received items from server",
		logger.F("itemCount", len(result.Items)),
		logger.F("syncVersion", result.SyncVersion))

	// Apply remote changes
	ctx := context.Background()
	for _, item := range result.Items {
		logger.Debug("Processing sync item",
			logger.F("type", item.Type),
			logger.F("clientID", item.ClientID),
			logger.F("deleted", item.Deleted))

		switch item.Type {
		case "project":
			// Use name directly from item, encrypted_data contains legacy color info
			name := item.Name
			color := "#4ECDC4"
			slug := item.Slug
			if slug == "" {
				slug = item.ClientID // Fallback for old data
			}
			if name == "" {
				// Fallback: try to parse from encrypted data
				data, err := base64.StdEncoding.DecodeString(item.EncryptedData)
				if err == nil {
					var p struct {
						Name  string `json:"name"`
						Color string `json:"color"`
					}
					_ = json.Unmarshal(data, &p)
					name = p.Name
					if p.Color != "" {
						color = p.Color
					}
				}
			}

			// Upsert project with server sync_version
			_, err := dbConn.GetProject(ctx, item.ClientID)
			if err != nil {
				// Not found, create
				logger.Debug("Creating project from sync", logger.F("id", item.ClientID), logger.F("name", name), logger.F("syncVersion", item.SyncVersion))
				_ = dbConn.CreateProject(ctx, database.CreateProjectParams{
					ID:        item.ClientID,
					Slug:      slug,
					Name:      name,
					Color:     sql.NullString{String: color, Valid: true},
					CreatedAt: time.Now().Format(time.RFC3339),
					UpdatedAt: time.Now().Format(time.RFC3339),
				})
				// Set sync_version from server
				_ = dbConn.UpdateProjectSyncVersion(ctx, database.UpdateProjectSyncVersionParams{
					ID:          item.ClientID,
					SyncVersion: sql.NullInt64{Int64: item.SyncVersion, Valid: true},
				})
			} else {
				// Exists, update with server data and sync_version
				logger.Debug("Updating project from sync", logger.F("id", item.ClientID), logger.F("name", name), logger.F("syncVersion", item.SyncVersion))
				_ = dbConn.OverwriteProject(ctx, database.OverwriteProjectParams{
					ID:          item.ClientID,
					Slug:        slug,
					Name:        name,
					Color:       sql.NullString{String: color, Valid: true},
					UpdatedAt:   time.Now().Format(time.RFC3339),
					SyncVersion: sql.NullInt64{Int64: item.SyncVersion, Valid: true},
				})
			}

		case "task":
			// Decrypt content
			content := ""
			if item.EncryptedContent != "" {
				data, err := base64.StdEncoding.DecodeString(item.EncryptedContent)
				if err == nil {
					var c struct {
						Content string `json:"content"`
					}
					_ = json.Unmarshal(data, &c)
					content = c.Content
				}
			}

			status := item.Status
			if status == "" {
				status = "process"
			}

			// Upsert task with server sync_version
			tExisting, err := dbConn.GetTask(ctx, item.ClientID)
			if err != nil {
				// Create
				logger.Debug("Creating task from sync", logger.F("id", item.ClientID), logger.F("content", content), logger.F("syncVersion", item.SyncVersion))
				_ = dbConn.CreateTask(ctx, database.CreateTaskParams{
					ID:        item.ClientID,
					ProjectID: item.ProjectID,
					Content:   content,
					Status:    sql.NullString{String: status, Valid: true},
					Priority:  item.Priority,
					DueDate:   sql.NullString{String: item.DueDate, Valid: item.DueDate != ""},
					CreatedAt: time.Now().Format(time.RFC3339),
					UpdatedAt: time.Now().Format(time.RFC3339),
				})
				// Set sync_version from server
				_ = dbConn.UpdateTaskSyncVersion(ctx, database.UpdateTaskSyncVersionParams{
					ID:          item.ClientID,
					SyncVersion: sql.NullInt64{Int64: item.SyncVersion, Valid: true},
				})
			} else {
				// Exists, update with server data and sync_version using OverwriteTask
				logger.Debug("Updating task from sync", logger.F("id", item.ClientID), logger.F("content", content), logger.F("syncVersion", item.SyncVersion))
				_ = dbConn.OverwriteTask(ctx, database.OverwriteTaskParams{
					ID:          tExisting.ID,
					ProjectID:   item.ProjectID,
					Content:     content,
					Status:      sql.NullString{String: status, Valid: true},
					Priority:    item.Priority,
					DueDate:     sql.NullString{String: item.DueDate, Valid: item.DueDate != ""},
					UpdatedAt:   time.Now().Format(time.RFC3339),
					SyncVersion: sql.NullInt64{Int64: item.SyncVersion, Valid: true},
				})
			}
		}
	}

	// Update last sync version
	if result.SyncVersion > c.config.LastSync {
		logger.Debug("Updating last sync version",
			logger.F("old", c.config.LastSync),
			logger.F("new", result.SyncVersion))
		c.config.LastSync = result.SyncVersion
		_ = c.saveConfig()
	}

	logger.Info("Pull completed", logger.F("itemsProcessed", len(result.Items)))
	return len(result.Items), nil
}
