package service

import (
	"context"
	"errors"
	"taskTracker/internal/models"
	"taskTracker/internal/repository/inter"
	"time"

	"github.com/google/uuid"
)

type TaskService struct {
	repo inter.TaskRepository
}

func NewTaskService(repo inter.TaskRepository) TaskService {
	return TaskService{
		repo: repo,
	}
}

func (s *TaskService) CreateNewTask(ctx context.Context, title, description string, dueTime time.Time) error {
	if title == "" {
		return errors.New("Название не может быть пустым")
	}

	if description == "" {
		return errors.New("Описание не может быть пустым")
	}

	if time.Now().After(dueTime) {
		return errors.New("Срок выполнения не может быть в прошлом")
	}

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
	if pagination == 0 {
		return nil, errors.New("Пагинация не может быть равна 0")
	}

	tasks, err := s.repo.GetWithLimit(ctx, pagination)
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

func (s *TaskService) GetTaskByID(ctx context.Context, id uuid.UUID) (*models.Task, error) {
	if id == uuid.Nil {
		return nil, errors.New("ID не может быть пустым")
	}

	task, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return task, nil
}

func (s *TaskService) UpdateTaskByID(ctx context.Context, id uuid.UUID, options ...TaskOption) error {
	if id == uuid.Nil {
		return errors.New("ID не может быть пустым")
	}

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
	if id == uuid.Nil {
		return errors.New("ID не может быть пустым")
	}
	return s.repo.Delete(ctx, id)
}
