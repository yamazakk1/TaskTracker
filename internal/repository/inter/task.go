package inter

import (
	"context"
	"taskTracker/internal/models"
	"github.com/google/uuid"
)

type TaskRepository interface {
	Create(context.Context, *models.Task) error
	Update(context.Context, *models.Task) error
	GetWithLimit(context.Context, int) ([]*models.Task, error)
	GetByID(context.Context, uuid.UUID) (*models.Task, error)
	Delete(context.Context, uuid.UUID) error
}
