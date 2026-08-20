package models

import "time"

type CreateTaskRequest struct {
	Title       string     `json:"title" binding:"required"`
	Description string     `json:"description"`
	Assignee    string     `json:"assignee"`
	DueDate     *time.Time `json:"due_date"`
}

type UpdateTaskRequest struct {
	Title       *string    `json:"title"`
	Description *string    `json:"description"`
	Status      *string    `json:"status" binding:"omitempty,oneof=new in_progress done cancelled"`
	Assignee    *string    `json:"assignee"`
	DueDate     *time.Time `json:"due_date"`
	Version     int        `json:"version" binding:"required"`
}
