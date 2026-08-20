package models

import (
	"time"

	"github.com/google/uuid"
)

type Task struct {
	ID          uuid.UUID  `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	Assignee    string     `json:"assignee"`
	DueDate     *time.Time `json:"due_date"`
	Version     int        `json:"version"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

type TaskHistory struct {
	ID        int64     `json:"id"`
	TaskID    uuid.UUID `json:"task_id"`
	FieldName string    `json:"field_name"`
	OldValue  *string   `json:"old_value"`
	NewValue  *string   `json:"new_value"`
	ChangedAt time.Time `json:"changed_at"`
}

type TaskFilter struct {
	Status      string    `form:"status"`
	Assignee    string    `form:"assignee"`
	DueDateFrom time.Time `form:"due_date_from"`
	DueDateTo   time.Time `form:"due_date_to"`
	Offset      int       `form:"offset,default=0"`
	Limit       int       `form:"limit,default=20"`
}
