package service

import (
	"context"
	"taskTracker/internal/models/task"
	"github.com/google/uuid"
)

type TaskRepository interface {
	Create(context.Context, *task.Task) error
	Update(context.Context, *task.Task) error
	GetWithLimit(context.Context, int) ([]*task.Task, error)
	GetByID(context.Context, uuid.UUID) (*task.Task, error)
	Delete(context.Context, *task.Task) error
}
