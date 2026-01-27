package handlers

import "context"
import "github.com/google/uuid"
import "time"
import "taskTracker/internal/models/task"

type Service interface {
    CreateTask(context.Context, string, string, time.Time) (*task.Task, error)
    GetActiveTasks(context.Context, int, int) ([]*task.Task, error)
    GetAllTasks(context.Context, int, int) ([]*task.Task, error)
    GetArchivedTasks(context.Context, int, int) ([]*task.Task, error)
    GetOverdueTasks(context.Context, int, int) ([]*task.Task, error)
    GetDeletedTasks(context.Context, int, int) ([]*task.Task, error)
    GetTaskByID(context.Context, uuid.UUID) (*task.Task, error)
    UpdateTask(context.Context, uuid.UUID, ...task.TaskOption) (*task.Task, error)
    DeleteTask(context.Context, uuid.UUID) error
    ArchiveTask(context.Context, uuid.UUID) (*task.Task, error)
    UnarchiveTask(context.Context, uuid.UUID) (*task.Task, error)
    RestoreTask(context.Context, uuid.UUID) (*task.Task, error)
    PurgeTask(context.Context, uuid.UUID) error
	HealthCheck(context.Context) error
}