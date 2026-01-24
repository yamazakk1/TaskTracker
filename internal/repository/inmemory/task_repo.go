package inmemory

import (
	"context"
	repo "taskTracker/internal/repository"
	"sync"
	"taskTracker/internal/models"
	"time"
	"github.com/google/uuid"
)

type TaskStorage struct {
	storage map[uuid.UUID]*models.Task
	mtx     *sync.RWMutex
}


func NewTaskStorage() *TaskStorage {
	return &TaskStorage{
		storage: make(map[uuid.UUID]*models.Task),
		mtx:     &sync.RWMutex{},
	}
}

func (s *TaskStorage) Create(ctx context.Context, task *models.Task) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()


	task.CreatedAt = time.Now()
	s.storage[task.ID] = task
	return nil
}

func (s *TaskStorage) Update(ctx context.Context, task *models.Task) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()


	now := time.Now()
	task.UpdatedAt = &now
	s.storage[task.ID] = task
	return nil
}

func (s *TaskStorage) GetByID(ctx context.Context, id uuid.UUID) (*models.Task, error) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()


	task, ok := s.storage[id]
	if !ok {
		return nil, repo.ErrNotfound
	} else {
		return task, nil
	}
}

func (s *TaskStorage) Delete(ctx context.Context, id uuid.UUID) error {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	task, ok := s.storage[id]
	if !ok {
		return repo.ErrNotfound
	} else {
		now := time.Now()
		task.UpdatedAt = &now
		task.DeletedAt = &now
		task.Status = models.StatusDeleted

		return nil
	}
}

func (s *TaskStorage) GetWithLimit(ctx context.Context, limit int) ([]*models.Task, error) {

	number := 0
	res := make([]*models.Task, limit)
	for _, value := range s.storage {
		if number == limit {
			break
		}
		res[number] = value
		number++
	}
	return res, nil
}
