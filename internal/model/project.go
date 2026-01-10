package model

import "time"

// Project represents a collection of tasks
type Project struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Color       string     `json:"color"`
	Archived    bool       `json:"archived"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at"`
	SyncVersion int64      `json:"sync_version"`
}

// DefaultInboxProject returns the default Inbox project
func DefaultInboxProject() Project {
	now := time.Now()
	return Project{
		ID:        "inbox",
		Name:      "Inbox",
		Color:     "#6C757D",
		CreatedAt: now,
		UpdatedAt: now,
	}
}
