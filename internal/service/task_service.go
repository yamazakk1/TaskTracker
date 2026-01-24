package service

import (
	"context"
	"errors"
	"fmt"
	"taskTracker/internal/logger"
	"taskTracker/internal/models"
	rep "taskTracker/internal/repository"
	"taskTracker/internal/repository/inter"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
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

func (s *TaskService) CreateNewTask(ctx context.Context, title, description string, dueTime time.Time) (uuid.UUID,error) {
	id := uuid.New()
	task := &models.Task{
		ID:          id,
		Title:       title,
		Description: description,
		Status:      models.StatusNew,
		DueTime:     dueTime,
		CreatedAt:   time.Now(),
	}

	return id, s.repo.Create(ctx, task)

}

func (s *TaskService) GetTasksWIthLimit(ctx context.Context, pagination int) ([]*models.Task, error) {
	tasks, err := s.repo.GetWithLimit(ctx, pagination)
	if err != nil {
		return nil, fmt.Errorf("получение задач: %w", err)
	}

	return tasks, nil
}

func (s *TaskService) GetTaskByID(ctx context.Context, id uuid.UUID) (*models.Task, error) {
	task, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, rep.ErrNotfound){
			logger.Info("Service: Задача не найдена", zap.String("target_id", id.String()))
			return nil, fmt.Errorf("задача %s не найдена: %w", id.String(), err)
		}
		return nil, fmt.Errorf("получение задачи: %w", err)
	}
	return task, nil
}

func (s *TaskService) UpdateTaskByID(ctx context.Context, id uuid.UUID, options ...TaskOption) error {
	task, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, rep.ErrNotfound){
			logger.Info("Service: Задача не найдена", zap.String("target_id", id.String()))
			return fmt.Errorf("задача %s не найдена: %w", id.String(), err)
		}
		return fmt.Errorf("получение задачи: %w", err)
	}

	for _, opt := range options {
		opt(task)
	}
	return s.repo.Update(ctx, task)
}

func (s *TaskService) DeleteTaskByID(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
