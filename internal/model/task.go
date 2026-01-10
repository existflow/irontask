package model

import "time"

// Priority levels for tasks
const (
	PriorityUrgent = 1 // Red - Urgent
	PriorityHigh   = 2 // Orange - High
	PriorityMedium = 3 // Yellow - Medium
	PriorityLow    = 4 // Blue - Low (default)
)

// Task represents a single todo item
type Task struct {
	ID          string     `json:"id"`
	ProjectID   string     `json:"project_id"`
	Content     string     `json:"content"`
	Done        bool       `json:"done"`
	Priority    int        `json:"priority"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	Tags        []string   `json:"tags,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
	SyncVersion int64      `json:"sync_version"`
}

// NewTask creates a new task with defaults
func NewTask(id, projectID, content string) Task {
	now := time.Now()
	return Task{
		ID:        id,
		ProjectID: projectID,
		Content:   content,
		Done:      false,
		Priority:  PriorityLow,
		Tags:      []string{},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// IsDue returns true if the task is due today or overdue
func (t *Task) IsDue() bool {
	if t.DueDate == nil {
		return false
	}
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return t.DueDate.Before(today.Add(24 * time.Hour))
}

// IsOverdue returns true if the task is past its due date
func (t *Task) IsOverdue() bool {
	if t.DueDate == nil {
		return false
	}
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return t.DueDate.Before(today)
}
