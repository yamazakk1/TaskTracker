package inmemory

import (
	"context"
	"errors"
	"sync"
	"taskTracker/internal/models"
	"time"
)

type UserStorage struct {
	storage map[string]*models.Task
	mtx     *sync.RWMutex
}

func NewUserStorage() *UserStorage {
	return &UserStorage{
		storage: make(map[string]*models.Task),
	}
}

func (s *UserStorage) Create(ctx context.Context, task *models.Task) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	task.CreatedAt = time.Now()
	return nil
}

func (s *UserStorage) Update(ctx context.Context, task *models.Task) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	now := time.Now()
	task.UpdatedAt = &now
	s.storage[task.ID] = task
	return nil
}

func (s *UserStorage) GetByID(ctx context.Context, id string) (*models.Task, error) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	if id == "" {
		return nil, errors.New("id не может быть пустым")
	}

	task, ok := s.storage[id]
	if !ok {
		return nil, errors.New("нет задачи с таким id")
	} else {
		return task, nil
	}
}

func (s *UserStorage) Delete(ctx context.Context, id string) error {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	if id == "" {
		return errors.New("id не может быть пустым")
	}

	task, ok := s.storage[id]
	if !ok {
		return errors.New("нет задачи с таким id")
	} else {
		now := time.Now()
		task.UpdatedAt = &now
		task.DeletedAt = &now
		return nil
	}
}
