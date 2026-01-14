package repository

import "taskTracker/internal/models"
import "context"

type TaskRepository interface{
	Create(ctx context.Context, task *models.Task) error
	Update(ctx context.Context, task *models.Task) error
	GetByID(ctx context.Context, id string) (*models.Task, error)
	Delete(ctx context.Context, id string) error
}

