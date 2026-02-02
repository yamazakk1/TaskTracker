package dto

import (
	"taskTracker/internal/models/task"
	"time"

	"github.com/google/uuid"
)

type CreateTaskRequest struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	DueTime     time.Time `json:"due_time"`
}

type UpdateTaskRequest struct {
	Title       *string      `json:"title,omitempty"`
	Description *string      `json:"description,omitempty"`
	Status      *task.Status `json:"status,omitempty"`
	DueTime     *time.Time   `json:"due_time,omitempty"`
}

type TaskResponse struct {
	UUID          uuid.UUID  `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	DueDate     time.Time  `json:"due_date"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
	IsOverdue   bool       `json:"is_overdue"` 
}

func FromTask(t *task.Task) TaskResponse {
	return TaskResponse{
		UUID:          t.UUID,
		Title:       t.Title,
		Description: t.Description,
		Status:      string(t.Status),
		DueDate:     t.DueTime,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
		IsOverdue: t.Status == task.StatusOverdue ||
			(t.Status != task.StatusDone && t.DueTime.Before(time.Now())),
	}
}

func FromTaskList(tasks []*task.Task) []TaskResponse {
	result := make([]TaskResponse, len(tasks))
	for i, t := range tasks {
		result[i] = FromTask(t)
	}
	return result
}
