package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

// SyncItem represents an encrypted item for sync
type SyncItem struct {
	ID            string `json:"id"`
	ClientID      string `json:"client_id"`
	Type          string `json:"type"` // "project" or "task"
	ProjectID     string `json:"project_id,omitempty"`
	EncryptedData string `json:"encrypted_data"` // Base64 encoded
	SyncVersion   int64  `json:"sync_version"`
	Deleted       bool   `json:"deleted"`
	UpdatedAt     string `json:"updated_at"`
}

// SyncPullResponse is the response for pull requests
type SyncPullResponse struct {
	Items       []SyncItem `json:"items"`
	SyncVersion int64      `json:"sync_version"`
}

// SyncPushRequest is the request for push
type SyncPushRequest struct {
	Items []SyncItem `json:"items"`
}

// SyncPushResponse is the response for push requests
type SyncPushResponse struct {
	Updated []SyncItem `json:"updated"`
}

// handleSyncPull returns items changed since last_sync_version
func (s *Server) handleSyncPull(c echo.Context) error {
	userID := c.Get("user_id").(string)

	// Get last sync version from query param
	lastVersion := int64(0)
	if v := c.QueryParam("since"); v != "" {
		lastVersion, _ = strconv.ParseInt(v, 10, 64)
	}

	var items []SyncItem

	// Get changed projects
	projectRows, err := s.db.Query(`
		SELECT id, client_id, name, COALESCE(encode(encrypted_data, 'base64'), ''), 
		       sync_version, deleted, updated_at
		FROM projects 
		WHERE user_id = $1 AND sync_version > $2
		ORDER BY sync_version ASC`,
		userID, lastVersion,
	)
	if err == nil {
		defer projectRows.Close()
		for projectRows.Next() {
			var item SyncItem
			var updatedAt time.Time
			var name string
			projectRows.Scan(&item.ID, &item.ClientID, &name, &item.EncryptedData,
				&item.SyncVersion, &item.Deleted, &updatedAt)
			item.Type = "project"
			item.UpdatedAt = updatedAt.Format(time.RFC3339)
			items = append(items, item)
		}
	}

	// Get changed tasks
	taskRows, err := s.db.Query(`
		SELECT id, client_id, project_id, COALESCE(encode(encrypted_data, 'base64'), ''),
		       sync_version, deleted, updated_at
		FROM tasks 
		WHERE user_id = $1 AND sync_version > $2
		ORDER BY sync_version ASC`,
		userID, lastVersion,
	)
	if err == nil {
		defer taskRows.Close()
		for taskRows.Next() {
			var item SyncItem
			var updatedAt time.Time
			taskRows.Scan(&item.ID, &item.ClientID, &item.ProjectID, &item.EncryptedData,
				&item.SyncVersion, &item.Deleted, &updatedAt)
			item.Type = "task"
			item.UpdatedAt = updatedAt.Format(time.RFC3339)
			items = append(items, item)
		}
	}

	// Get the latest sync version
	var maxVersion int64
	s.db.QueryRow(`
		SELECT COALESCE(MAX(sync_version), 0) FROM (
			SELECT sync_version FROM projects WHERE user_id = $1
			UNION ALL
			SELECT sync_version FROM tasks WHERE user_id = $1
		) combined`,
		userID,
	).Scan(&maxVersion)

	c.Logger().Infof("Sync pull for user %s: %d items since version %d", userID, len(items), lastVersion)

	return c.JSON(http.StatusOK, SyncPullResponse{
		Items:       items,
		SyncVersion: maxVersion,
	})
}

// handleSyncPush accepts changed items from client
func (s *Server) handleSyncPush(c echo.Context) error {
	userID := c.Get("user_id").(string)

	var req SyncPushRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	var updated []SyncItem

	for _, item := range req.Items {
		switch item.Type {
		case "project":
			// Upsert project
			var serverVersion int64
			err := s.db.QueryRow(`
				INSERT INTO projects (user_id, client_id, name, encrypted_data, deleted, sync_version, updated_at)
				VALUES ($1, $2, '', decode($3, 'base64'), $4, 
					(SELECT COALESCE(MAX(sync_version), 0) + 1 FROM projects WHERE user_id = $1), NOW())
				ON CONFLICT (user_id, client_id) DO UPDATE SET
					encrypted_data = decode($3, 'base64'),
					deleted = $4,
					sync_version = (SELECT COALESCE(MAX(sync_version), 0) + 1 FROM projects WHERE user_id = $1),
					updated_at = NOW()
				RETURNING sync_version`,
				userID, item.ClientID, item.EncryptedData, item.Deleted,
			).Scan(&serverVersion)

			if err == nil {
				item.SyncVersion = serverVersion
				updated = append(updated, item)
			}

		case "task":
			// Upsert task
			var serverVersion int64
			err := s.db.QueryRow(`
				INSERT INTO tasks (user_id, client_id, project_id, encrypted_data, deleted, sync_version, updated_at)
				VALUES ($1, $2, $3, decode($4, 'base64'), $5,
					(SELECT COALESCE(MAX(sync_version), 0) + 1 FROM tasks WHERE user_id = $1), NOW())
				ON CONFLICT (user_id, client_id) DO UPDATE SET
					project_id = $3,
					encrypted_data = decode($4, 'base64'),
					deleted = $5,
					sync_version = (SELECT COALESCE(MAX(sync_version), 0) + 1 FROM tasks WHERE user_id = $1),
					updated_at = NOW()
				RETURNING sync_version`,
				userID, item.ClientID, item.ProjectID, item.EncryptedData, item.Deleted,
			).Scan(&serverVersion)

			if err == nil {
				item.SyncVersion = serverVersion
				updated = append(updated, item)
			}
		}
	}

	c.Logger().Infof("Sync push for user %s: %d items updated", userID, len(updated))

	return c.JSON(http.StatusOK, SyncPushResponse{Updated: updated})
}
