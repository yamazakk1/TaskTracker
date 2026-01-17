package inmemory

import (
	"context"
	"errors"
	"sync"
	"taskTracker/internal/models"
	"time"
	"github.com/google/uuid"
)

type UserStorage struct {
	storage map[uuid.UUID]*models.Task
	mtx     *sync.RWMutex
}



func NewUserStorage() *UserStorage {
	return &UserStorage{
		storage: make(map[uuid.UUID]*models.Task),
	}
}

func (s *UserStorage) Create(ctx context.Context, task *models.Task) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	task.CreatedAt = time.Now()
	s.storage[task.ID] = task
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

func (s *UserStorage) GetByID(ctx context.Context, id uuid.UUID) (*models.Task, error) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	if id == uuid.Nil {
		return nil, errors.New("id не может быть пустым")
	}

	task, ok := s.storage[id]
	if !ok {
		return nil, errors.New("нет задачи с таким id")
	} else {
		return task, nil
	}
}

func (s *UserStorage) Delete(ctx context.Context, id uuid.UUID) error {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	if id == uuid.Nil {
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

func (s *UserStorage) GetWithLimit(ctx context.Context, limit int) ([]*models.Task,error){
	if limit == 0{
		return nil, errors.New("limit не может быть 0")
	}
	number := 0
	res := make([]*models.Task,limit)
	for _, value := range s.storage{
		if number == limit{
			break
		}
		res[number] =value 
		number++
	}
	return res, nil
}
