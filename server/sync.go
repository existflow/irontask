package server

import (
	"database/sql"
	"encoding/base64"
	"net/http"
	"strconv"

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
		c.Logger().Error("get projects error:", err)
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
		c.Logger().Error("get tasks error:", err)
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

	c.Logger().Infof("Sync pull for user %s: %d items since version %d", userID, len(items), lastVersion)

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

	var updated []SyncItem

	for _, item := range req.Items {
		switch item.Type {
		case "project":
			data, err := base64.StdEncoding.DecodeString(item.EncryptedData)
			if err != nil {
				c.Logger().Error("base64 decode error:", err)
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

			version, err := s.queries.UpsertProject(c.Request().Context(), database.UpsertProjectParams{
				UserID:        userUUID,
				ClientID:      item.ClientID,
				Slug:          slug,
				Name:          name,
				Color:         sql.NullString{String: "", Valid: true},
				EncryptedData: data,
				Deleted:       sql.NullBool{Bool: item.Deleted, Valid: true},
			})
			if err == nil {
				item.SyncVersion = version.Int64
				updated = append(updated, item)
			} else {
				c.Logger().Error("upsert project error:", err)
			}

		case "task":
			contentData, err := base64.StdEncoding.DecodeString(item.EncryptedContent)
			if err != nil {
				c.Logger().Error("base64 decode error:", err)
				continue
			}

			status := item.Status
			if status == "" {
				status = "process"
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
			})
			if err == nil {
				item.SyncVersion = version.Int64
				updated = append(updated, item)
			} else {
				c.Logger().Error("upsert task error:", err)
			}
		}
	}

	c.Logger().Infof("Sync push for user %s: %d items updated", userID, len(updated))

	return c.JSON(http.StatusOK, SyncPushResponse{Updated: updated})
}

// handleClear wipes all data for the user
func (s *Server) handleClear(c echo.Context) error {
	userIDStr := c.Get("user_id").(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid user id"})
	}

	if err := s.queries.ClearTasks(c.Request().Context(), userID); err != nil {
		c.Logger().Error("clear tasks error:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to clear tasks"})
	}

	if err := s.queries.ClearProjects(c.Request().Context(), userID); err != nil {
		c.Logger().Error("clear projects error:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to clear projects"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "all data cleared successfully"})
}
