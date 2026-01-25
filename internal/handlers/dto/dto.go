package dto

import "time"
import "taskTracker/internal/models/task"

type CreateTaskRequest struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	DueTime     time.Time `json:"due_time"`
}

type UpdateTaskRequest struct{
	Title *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Status *task.Status `json:"status,omitempty"`
	DueTime *time.Time  `json:"due_time,omitempty"`
}
