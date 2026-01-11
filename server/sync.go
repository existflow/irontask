package server

import (
	"database/sql"
	"encoding/base64"
	"net/http"
	"strconv"
	"time"

	"github.com/existflow/irontask/internal/logger"
	"github.com/existflow/irontask/server/database"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// SyncItem represents an encrypted item for sync
type SyncItem struct {
	ID               string `json:"id"`
	ClientID         string `json:"client_id"`
	Type             string `json:"type"` // "project" or "task"
	Slug             string `json:"slug,omitempty"`
	Name             string `json:"name,omitempty"`
	ProjectID        string `json:"project_id,omitempty"`
	EncryptedData    string `json:"encrypted_data,omitempty"`    // For projects
	EncryptedContent string `json:"encrypted_content,omitempty"` // For tasks (content only)
	Status           string `json:"status,omitempty"`
	Priority         int32  `json:"priority,omitempty"`
	DueDate          string `json:"due_date,omitempty"`
	SyncVersion      int64  `json:"sync_version"`
	Deleted          bool   `json:"deleted"`
	ClientUpdatedAt  string `json:"client_updated_at,omitempty"` // Client timestamp for conflict detection
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
// ConflictItem represents a conflicting item
type ConflictItem struct {
	ClientID      string   `json:"client_id"`
	Type          string   `json:"type"`
	ServerVersion int64    `json:"server_version"`
	ServerData    SyncItem `json:"server_data"`
	ClientData    SyncItem `json:"client_data"`
}

// SyncPushResponse is the response for push requests
type SyncPushResponse struct {
	Updated   []SyncItem     `json:"updated"`
	Conflicts []ConflictItem `json:"conflicts,omitempty"`
}

// handleSyncPull returns items changed since last_sync_version
func (s *Server) handleSyncPull(c echo.Context) error {
	userID := c.Get("user_id").(string)
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid user id"})
	}

	// Get last sync version from query param
	lastVersion := int64(0)
	if v := c.QueryParam("since"); v != "" {
		val, _ := strconv.ParseInt(v, 10, 64)
		lastVersion = val
	}

	// Get projects changed
	projects, err := s.queries.GetProjectsChanged(c.Request().Context(), database.GetProjectsChangedParams{
		UserID:      userUUID,
		SyncVersion: sql.NullInt64{Int64: lastVersion, Valid: true},
	})
	if err != nil && err != sql.ErrNoRows {
		logger.Error("sync pull: get projects failed", logger.F("error", err), logger.F("user", userID[:8]))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	var items []SyncItem
	for _, p := range projects {
		items = append(items, SyncItem{
			ID:            p.ClientID,
			ClientID:      p.ClientID,
			Type:          p.Type,
			Slug:          p.Slug,
			Name:          p.Name,
			EncryptedData: base64.StdEncoding.EncodeToString(p.EncryptedData),
			SyncVersion:   p.SyncVersion.Int64,
			Deleted:       p.Deleted.Bool,
		})
	}

	// Get tasks changed
	tasks, err := s.queries.GetTasksChanged(c.Request().Context(), database.GetTasksChangedParams{
		UserID:      userUUID,
		SyncVersion: sql.NullInt64{Int64: lastVersion, Valid: true},
	})
	if err != nil && err != sql.ErrNoRows {
		logger.Error("sync pull: get tasks failed", logger.F("error", err), logger.F("user", userID[:8]))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	for _, t := range tasks {
		dueDate := ""
		if t.DueDate.Valid {
			dueDate = t.DueDate.String
		}
		status := "process"
		if t.Status.Valid {
			status = t.Status.String
		}

		items = append(items, SyncItem{
			ID:               t.ClientID,
			ClientID:         t.ClientID,
			ProjectID:        t.ProjectID,
			Type:             t.Type,
			EncryptedContent: base64.StdEncoding.EncodeToString(t.EncryptedContent),
			Status:           status,
			Priority:         t.Priority.Int32,
			DueDate:          dueDate,
			SyncVersion:      t.SyncVersion.Int64,
			Deleted:          t.Deleted.Bool,
		})
	}

	// Calculate max version
	maxVersion := lastVersion
	for _, item := range items {
		if item.SyncVersion > maxVersion {
			maxVersion = item.SyncVersion
		}
	}

	logger.Debug("sync pull",
		logger.F("user", userID[:8]),
		logger.F("since", lastVersion),
		logger.F("items", len(items)))

	return c.JSON(http.StatusOK, SyncPullResponse{
		Items:       items,
		SyncVersion: maxVersion,
	})
}

func (s *Server) handleSyncPush(c echo.Context) error {
	userID := c.Get("user_id").(string)
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid user id"})
	}

	var req SyncPushRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	logger.Debug("sync push received",
		logger.F("user", userID[:8]),
		logger.F("items", len(req.Items)))

	var updated []SyncItem
	var conflicts []ConflictItem

	for _, item := range req.Items {

		// Check for conflicts
		var clientTime time.Time
		if item.ClientUpdatedAt != "" {
			var err error
			clientTime, err = time.Parse(time.RFC3339, item.ClientUpdatedAt)
			if err != nil {
				logger.Warn("sync push: invalid timestamp", logger.F("timestamp", item.ClientUpdatedAt))
			}
		}

		// Conflict Detection Logic
		hasConflict := false
		var serverItem SyncItem
		var serverUpdatedAt time.Time

		if item.Type == "task" {
			current, err := s.queries.GetTaskForConflict(c.Request().Context(), database.GetTaskForConflictParams{
				UserID:   userUUID,
				ClientID: item.ClientID,
			})
			if err == nil {
				// Item exists on server
				if current.UpdatedAt.Valid {
					serverUpdatedAt = current.UpdatedAt.Time
					// If server has a newer version AND client timestamp is valid
					if !clientTime.IsZero() && serverUpdatedAt.After(clientTime) {
						hasConflict = true
						// Populate full server data for conflict response
						dueDate := ""
						if current.DueDate.Valid {
							dueDate = current.DueDate.String
						}
						serverItem = SyncItem{
							ID:               item.ClientID,
							ClientID:         item.ClientID,
							Type:             "task",
							ProjectID:        current.ProjectID,
							EncryptedContent: base64.StdEncoding.EncodeToString(current.EncryptedContent),
							Status:           current.Status.String,
							Priority:         current.Priority.Int32,
							DueDate:          dueDate,
							SyncVersion:      current.SyncVersion.Int64,
							Deleted:          current.Deleted.Bool,
						}
					}
				}
			}
		} else if item.Type == "project" {
			current, err := s.queries.GetProjectForConflict(c.Request().Context(), database.GetProjectForConflictParams{
				UserID:   userUUID,
				ClientID: item.ClientID,
			})
			if err == nil {
				if current.UpdatedAt.Valid {
					serverUpdatedAt = current.UpdatedAt.Time
					if !clientTime.IsZero() && serverUpdatedAt.After(clientTime) {
						hasConflict = true
						// Populate full server data for conflict response
						serverItem = SyncItem{
							ID:            item.ClientID,
							ClientID:      item.ClientID,
							Type:          "project",
							Slug:          current.Slug,
							Name:          current.Name,
							EncryptedData: base64.StdEncoding.EncodeToString(current.EncryptedData),
							SyncVersion:   current.SyncVersion.Int64,
							Deleted:       current.Deleted.Bool,
						}
					}
				}
			}
		}

		if hasConflict {
			logger.Info("sync conflict detected",
				logger.F("type", item.Type),
				logger.F("id", item.ClientID[:8]),
				logger.F("serverTime", serverUpdatedAt.Format(time.RFC3339)),
				logger.F("clientTime", clientTime.Format(time.RFC3339)))

			conflicts = append(conflicts, ConflictItem{
				ClientID:      item.ClientID,
				Type:          item.Type,
				ServerVersion: serverItem.SyncVersion,
				ServerData:    serverItem,
				ClientData:    item,
			})
			continue // Skip upsert for conflicting item
		}

		switch item.Type {
		case "project":
			data, err := base64.StdEncoding.DecodeString(item.EncryptedData)
			if err != nil {
				logger.Error("sync push: base64 decode failed", logger.F("id", item.ClientID[:8]))
				continue
			}

			slug := item.Slug
			if slug == "" {
				slug = item.ClientID // Fallback
			}
			name := item.Name
			if name == "" {
				name = slug
			}

			// Parse client timestamp
			var clientUpdatedAt sql.NullTime
			if !clientTime.IsZero() {
				clientUpdatedAt = sql.NullTime{Time: clientTime, Valid: true}
			}

			version, err := s.queries.UpsertProject(c.Request().Context(), database.UpsertProjectParams{
				UserID:          userUUID,
				ClientID:        item.ClientID,
				Slug:            slug,
				Name:            name,
				Color:           sql.NullString{String: "", Valid: true},
				EncryptedData:   data,
				Deleted:         sql.NullBool{Bool: item.Deleted, Valid: true},
				ClientUpdatedAt: clientUpdatedAt,
			})
			if err != nil {
				logger.Error("sync push: upsert project failed",
					logger.F("id", item.ClientID[:8]),
					logger.F("error", err))
			} else {
				item.SyncVersion = version.Int64
				updated = append(updated, item)
			}

		case "task":
			contentData, err := base64.StdEncoding.DecodeString(item.EncryptedContent)
			if err != nil {
				logger.Error("sync push: base64 decode failed", logger.F("id", item.ClientID[:8]))
				continue
			}

			status := item.Status
			if status == "" {
				status = "process"
			}

			// Parse client timestamp for task
			var taskClientUpdatedAt sql.NullTime
			if !clientTime.IsZero() {
				taskClientUpdatedAt = sql.NullTime{Time: clientTime, Valid: true}
			}

			version, err := s.queries.UpsertTask(c.Request().Context(), database.UpsertTaskParams{
				UserID:           userUUID,
				ClientID:         item.ClientID,
				ProjectID:        item.ProjectID,
				EncryptedContent: contentData,
				Status:           sql.NullString{String: status, Valid: true},
				Priority:         sql.NullInt32{Int32: item.Priority, Valid: true},
				DueDate:          sql.NullString{String: item.DueDate, Valid: item.DueDate != ""},
				Deleted:          sql.NullBool{Bool: item.Deleted, Valid: true},
				ClientUpdatedAt:  taskClientUpdatedAt,
			})

			if err != nil {
				logger.Error("sync push: upsert task failed",
					logger.F("id", item.ClientID[:8]),
					logger.F("error", err))
			} else {
				item.SyncVersion = version.Int64
				updated = append(updated, item)
			}
		}
	}

	logger.Info("sync push complete",
		logger.F("user", userID[:8]),
		logger.F("updated", len(updated)),
		logger.F("conflicts", len(conflicts)))

	return c.JSON(http.StatusOK, SyncPushResponse{
		Updated:   updated,
		Conflicts: conflicts,
	})
}

// handleClear wipes all data for the user
func (s *Server) handleClear(c echo.Context) error {
	userIDStr := c.Get("user_id").(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid user id"})
	}

	if err := s.queries.ClearTasks(c.Request().Context(), userID); err != nil {
		logger.Error("clear tasks failed", logger.F("user", userIDStr[:8]), logger.F("error", err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to clear tasks"})
	}

	if err := s.queries.ClearProjects(c.Request().Context(), userID); err != nil {
		logger.Error("clear projects failed", logger.F("user", userIDStr[:8]), logger.F("error", err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to clear projects"})
	}

	logger.Info("user data cleared", logger.F("user", userIDStr[:8]))
	return c.JSON(http.StatusOK, map[string]string{"message": "all data cleared successfully"})
}
