package service

import (
	"context"
	"errors"
	"fmt"
	"taskTracker/internal/logger"
	"taskTracker/internal/models/task"
	rep "taskTracker/internal/repository"

	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// здесь происходит проверка ошибок бизнес-логики

type TaskService struct {
	repo TaskRepository
}

func NewTaskService(repo TaskRepository) TaskService {
	return TaskService{
		repo: repo,
	}
}

func (s *TaskService) GetTasksWithLimit(ctx context.Context, pagination int) ([]*task.Task, error) {
	tasks, err := s.repo.GetWithLimit(ctx, pagination)
	if err != nil {
		return nil, fmt.Errorf("получение задач: %w", err)
	}

	return tasks, nil
}

func (s *TaskService) GetTaskByID(ctx context.Context, id uuid.UUID) (*task.Task, error) {
	task, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, rep.ErrNotfound) {
			logger.Info("Service: Задача не найдена", zap.String("target_id", id.String()))
			return nil, fmt.Errorf("задача %s не найдена: %w", id.String(), err)
		}
		return nil, fmt.Errorf("получение задачи: %w", err)
	}

	return task, nil
}

func (s *TaskService) UpdateTaskByID(ctx context.Context, id uuid.UUID, options ...task.TaskOption) error {
	task, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, rep.ErrNotfound) {
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
	task1, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, rep.ErrNotfound) {
			logger.Info("Service: Задача не найдена", zap.String("target_id", id.String()))
			return fmt.Errorf("задача %s не найдена: %w", id.String(), err)
		}
		return fmt.Errorf("получение задачи: %w", err)
	}

	if task1.Status == task.StatusDeleted{
		logger.Warn("Repository: Задача уже удалена")
		return errors.New("мягкое удаление задачи: задача уже удалена")
	}

	task1.Status = task.StatusDeleted
	return s.repo.Delete(ctx, task1)
}

func (s *TaskService) CreateNewTask(ctx context.Context, title string, description string, dueTime time.Time) (uuid.UUID, error) {
	uuid := uuid.New()
	task := &task.Task{
		UUID:        uuid,
		Title:       title,
		Description: description,
		Status:      task.StatusNew,
		DueTime:     dueTime,
	}

	return uuid, s.repo.Create(ctx, task)

}
