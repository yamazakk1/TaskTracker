package service

import (
	"context"
	"taskTracker/internal/models"
	"taskTracker/internal/repository/inter"
	"time"

	"github.com/google/uuid"
)

// здесь происходит проверка ошибок бизнес-логики

type TaskService struct {
	repo inter.TaskRepository
}

func NewTaskService(repo inter.TaskRepository) TaskService {
	return TaskService{
		repo: repo,
	}
}

func (s *TaskService) CreateNewTask(ctx context.Context, title, description string, dueTime time.Time) error {
	id := uuid.New()
	task := &models.Task{
		ID:          id,
		Title:       title,
		Description: description,
		Status:      models.StatusNew,
		DueTime:     dueTime,
		CreatedAt:   time.Now(),
	}

	return s.repo.Create(ctx, task)

}

func (s *TaskService) GetTasksWIthLimit(ctx context.Context, pagination int) ([]*models.Task, error) {

	tasks, err := s.repo.GetWithLimit(ctx, pagination)
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

func (s *TaskService) GetTaskByID(ctx context.Context, id uuid.UUID) (*models.Task, error) {


	task, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return task, nil
}

func (s *TaskService) UpdateTaskByID(ctx context.Context, id uuid.UUID, options ...TaskOption) error {


	task, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	for _, opt := range options {
		opt(task)
	}
	return s.repo.Update(ctx, task)
}

func (s *TaskService) DeleteTaskByID(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
