package service

import (
	"context"
	"taskTracker/internal/models/task"
	"github.com/google/uuid"
	"time"
)

type TaskRepository interface {
	Create(context.Context, *task.Task) error
	Update(context.Context, *task.Task) error
	GetAllWithLimit(context.Context, int,int) ([]*task.Task, error)
	GetStatusedWithLimit(context.Context, int, int, task.Status) ([]*task.Task, error)
	GetFlaggedWithLimit(context.Context, int, int, task.Flag) ([]*task.Task, error)
	GetTasksDueBefore(context.Context, time.Time, int) ([]*task.Task, error)
	GetByID(context.Context, uuid.UUID) (*task.Task, error)
	DeleteSoft(context.Context, *task.Task) error
	DeleteFull(context.Context, uuid.UUID) error 
	HealthCheck(context.Context) error
}
