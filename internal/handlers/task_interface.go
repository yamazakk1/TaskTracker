package handlers

import "context"
import "github.com/google/uuid"
import "time"
import "taskTracker/internal/models/task"

type TaskService interface{
	CreateNewTask(context.Context, string, string, time.Time) (uuid.UUID,error)
	GetTasksWithLimit(context.Context,int) ([]*task.Task, error)
	GetTaskByID(context.Context, uuid.UUID) (*task.Task, error) 
	UpdateTaskByID(context.Context, uuid.UUID, ...task.TaskOption) error 
	DeleteTaskByID(context.Context, uuid.UUID) error 
}