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

	"github.com/tphuc/irontask/internal/database"
	"github.com/tphuc/irontask/internal/db"
)

// SyncItem represents an item to sync
type SyncItem struct {
	ID            string `json:"id"`
	ClientID      string `json:"client_id"`
	Type          string `json:"type"` // "project" or "task"
	ProjectID     string `json:"project_id,omitempty"`
	EncryptedData string `json:"encrypted_data"`
	SyncVersion   int64  `json:"sync_version"`
	Deleted       bool   `json:"deleted"`
}

// SyncPullResponse is the response from pull
type SyncPullResponse struct {
	Items       []SyncItem `json:"items"`
	SyncVersion int64      `json:"sync_version"`
}

// SyncPushResponse is the response from push
type SyncPushResponse struct {
	Updated []SyncItem `json:"updated"`
}

// SyncResult holds sync statistics
type SyncResult struct {
	Pushed int
	Pulled int
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
		c.saveConfig()

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
		pushed, err := c.pushChanges(database)
		if err != nil {
			return nil, fmt.Errorf("push failed: %w", err)
		}
		result.Pushed = pushed

	default: // SyncModeMerge
		// 1. Push local changes
		pushed, err := c.pushChanges(database)
		if err != nil {
			return nil, fmt.Errorf("push failed: %w", err)
		}
		result.Pushed = pushed

		// 2. Pull remote changes
		pulled, err := c.pullChanges(database)
		if err != nil {
			return nil, fmt.Errorf("pull failed: %w", err)
		}
		result.Pulled = pulled
	}

	// Mark as synced once after first successful sync
	if !c.config.HasSyncedOnce {
		c.SetSyncedOnce()
	}

	return result, nil
}

// pushChanges sends local changes to server
func (c *Client) pushChanges(dbConn *db.DB) (int, error) {
	var items []SyncItem

	// Get projects that need syncing (updated since last sync)
	projects, _ := dbConn.ListProjects(context.Background())
	for _, p := range projects {
		color := ""
		if p.Color.Valid {
			color = p.Color.String
		}
		// Simple encoding - in production use encryption
		data, _ := json.Marshal(map[string]interface{}{
			"name":  p.Name,
			"color": color,
		})

		items = append(items, SyncItem{
			ClientID:      p.ID,
			Type:          "project",
			EncryptedData: base64.StdEncoding.EncodeToString(data),
			Deleted:       p.DeletedAt.Valid,
		})
	}

	// Get tasks that need syncing
	tasks, _ := dbConn.ListTasks(context.Background(), database.ListTasksParams{
		IncludeDone: true,
	})
	for _, t := range tasks {
		dueDate := ""
		if t.DueDate.Valid {
			dueDate = t.DueDate.String
		}
		data, _ := json.Marshal(map[string]interface{}{
			"content":  t.Content,
			"priority": t.Priority,
			"done":     t.Done,
			"due":      dueDate,
		})

		items = append(items, SyncItem{
			ClientID:      t.ID,
			Type:          "task",
			ProjectID:     t.ProjectID,
			EncryptedData: base64.StdEncoding.EncodeToString(data),
			Deleted:       t.DeletedAt.Valid,
		})
	}

	if len(items) == 0 {
		return 0, nil
	}

	// Send to server
	body, _ := json.Marshal(map[string]interface{}{
		"items": items,
	})

	req, _ := http.NewRequest("POST", c.config.ServerURL+"/api/v1/sync", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.Token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("server error: %s", string(respBody))
	}

	var result SyncPushResponse
	json.NewDecoder(resp.Body).Decode(&result)

	return len(result.Updated), nil
}

// pullChanges gets remote changes from server
func (c *Client) pullChanges(dbConn *db.DB) (int, error) {
	url := fmt.Sprintf("%s/api/v1/sync?since=%d", c.config.ServerURL, c.config.LastSync)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+c.config.Token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("server error: %s", string(respBody))
	}

	var result SyncPullResponse
	json.NewDecoder(resp.Body).Decode(&result)

	// Apply remote changes
	ctx := context.Background()
	for _, item := range result.Items {
		data, err := base64.StdEncoding.DecodeString(item.EncryptedData)
		if err != nil {
			continue
		}

		switch item.Type {
		case "project":
			var p struct {
				Name  string `json:"name"`
				Color string `json:"color"`
			}
			json.Unmarshal(data, &p)

			// Upsert project
			_, err := dbConn.GetProject(ctx, item.ClientID)
			if err != nil {
				// Not found, create
				dbConn.CreateProject(ctx, database.CreateProjectParams{
					ID:        item.ClientID,
					Name:      p.Name,
					Color:     sql.NullString{String: p.Color, Valid: true},
					CreatedAt: time.Now().Format(time.RFC3339),
					UpdatedAt: time.Now().Format(time.RFC3339),
				})
			} else {
				// Update existing
				// Note: UpdateProject query was not generated, need to add it if full update logic is needed.
				// For now assuming immutable or add UpdateProject query later.
			}

		case "task":
			var t struct {
				Content  string `json:"content"`
				Priority int    `json:"priority"`
				Done     bool   `json:"done"`
			}
			json.Unmarshal(data, &t)

			// Upsert task
			tExisting, err := dbConn.GetTask(ctx, item.ClientID)
			if err != nil {
				// Create
				dbConn.CreateTask(ctx, database.CreateTaskParams{
					ID:        item.ClientID,
					ProjectID: item.ProjectID,
					Content:   t.Content,
					Priority:  t.Priority,
					Done:      t.Done,
					CreatedAt: time.Now().Format(time.RFC3339),
					UpdatedAt: time.Now().Format(time.RFC3339),
				})
			} else {
				// Update
				dbConn.UpdateTask(ctx, database.UpdateTaskParams{
					ID:        tExisting.ID,
					ProjectID: item.ProjectID,
					Content:   t.Content,
					Priority:  t.Priority,
					Done:      t.Done,
					UpdatedAt: time.Now().Format(time.RFC3339),
				})
			}
		}
	}

	// Update last sync version
	if result.SyncVersion > c.config.LastSync {
		c.config.LastSync = result.SyncVersion
		c.saveConfig()
	}

	return len(result.Items), nil
}
