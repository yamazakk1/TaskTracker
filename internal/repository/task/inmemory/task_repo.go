package inmemory

import (
	"context"
	"sync"
	"taskTracker/internal/models/task"
	repo "taskTracker/internal/repository"
	"time"

	"github.com/google/uuid"
)

type TaskStorage struct {
	storage map[uuid.UUID]*task.Task
	mtx     *sync.RWMutex
}

func NewTaskStorage() *TaskStorage {
	return &TaskStorage{
		storage: make(map[uuid.UUID]*task.Task),
		mtx:     &sync.RWMutex{},
	}
}

func (s *TaskStorage) Create(ctx context.Context, task *task.Task) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	task.CreatedAt = time.Now()
	s.storage[task.UUID] = task
	return nil
}

func (s *TaskStorage) Update(ctx context.Context, task *task.Task) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	now := time.Now()
	task.UpdatedAt = &now
	s.storage[task.UUID] = task
	return nil
}

func (s *TaskStorage) GetByID(ctx context.Context, id uuid.UUID) (*task.Task, error) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	task, ok := s.storage[id]
	if !ok {
		return nil, repo.ErrNotfound
	} else {
		return task, nil
	}
}

func (s *TaskStorage) Delete(ctx context.Context, task1 *task.Task) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	task1, ok := s.storage[task1.UUID]
	if !ok {
		return repo.ErrNotfound
	} else {
		now := time.Now()
		task1.UpdatedAt = &now
		task1.DeletedAt = &now
		task1.Status = task.StatusDeleted

		return nil
	}
}

func (s *TaskStorage) GetWithLimit(ctx context.Context, limit int) ([]*task.Task, error) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	res := []*task.Task{}

	for _, task1 := range s.storage {
		if task1.Status == task.StatusDeleted {
			continue
		}
		if len(res) >= limit {
			break
		}
		res = append(res, task1)
	}
	return res, nil
}
