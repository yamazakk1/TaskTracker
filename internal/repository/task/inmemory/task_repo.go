package inmemory

import (
	"context"
	"sync"
	"taskTracker/internal/logger"
	"taskTracker/internal/models/task"
	repo "taskTracker/internal/repository"
	"time"

	"github.com/google/uuid"
)

type TaskStorage struct {
	storage map[uuid.UUID]*task.Task
	mtx     *sync.RWMutex
	ids     []uuid.UUID
}

func NewTaskStorage() *TaskStorage {
	return &TaskStorage{
		storage: make(map[uuid.UUID]*task.Task),
		mtx:     &sync.RWMutex{},
		ids:     []uuid.UUID{},
	}
}

func (s *TaskStorage) HealthCheck(ctx context.Context) error {
	logger.Info("Repository: Соединение стабильно")
	return nil
}

func (s *TaskStorage) Create(ctx context.Context, taskToCreate *task.Task) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	taskToCreate.CreatedAt = time.Now()
	taskToCreate.Flag = task.FlagActive

	s.storage[taskToCreate.UUID] = taskToCreate
	s.ids = append(s.ids, taskToCreate.UUID)
	return nil
}

func (s *TaskStorage) Update(ctx context.Context, taskToUpdate *task.Task) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	now := time.Now()
	taskToUpdate.UpdatedAt = &now
	taskToUpdate.Version++
	s.storage[taskToUpdate.UUID] = taskToUpdate

	return nil
}

func (s *TaskStorage) GetByID(ctx context.Context, id uuid.UUID) (*task.Task, error) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	taskToGet, ok := s.storage[id]
	if !ok {
		return nil, repo.ErrNotFound
	}
	return taskToGet, nil

}

// мягкое удаление с изменением флага
func (s *TaskStorage) DeleteSoft(ctx context.Context, taskToDelete *task.Task) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	taskExisted, ok := s.storage[taskToDelete.UUID]
	if !ok {
		return repo.ErrNotFound
	}

	now := time.Now()
	taskExisted.UpdatedAt = &now
	taskExisted.DeletedAt = &now
	taskExisted.Flag = task.FlagDeleted

	return nil

}

// полное удаление
func (s *TaskStorage) DeleteFull(ctx context.Context, uuid uuid.UUID) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	delete(s.storage, uuid)
	for ind, val := range s.ids {
		if val == uuid {
			s.ids = append(s.ids[:ind], s.ids[ind+1:]...)
		}
	}
	return nil
}

// получение задач с флагами active или archived
func (s *TaskStorage) GetAllWithLimit(ctx context.Context, page, limit int) ([]*task.Task, error) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	res := []*task.Task{}
	offset := (page - 1) * limit

	for i := offset; i < len(s.ids); i++ {
		if len(res) >= limit {
			break
		}

		taskToGet := s.storage[s.ids[i]]
		if taskToGet.Flag == task.FlagDeleted {
			continue
		}

		res = append(res, taskToGet)
	}

	return res, nil
}

// получение задач с определённым флагом
func (s *TaskStorage) GetFlaggedWithLimit(ctx context.Context, page, limit int, flag task.Flag) ([]*task.Task, error) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	res := []*task.Task{}
	offset := (page - 1) * limit

	for i := offset; i < len(s.ids); i++ {
		if len(res) >= limit {
			break
		}

		taskToGet := s.storage[s.ids[i]]
		if taskToGet.Flag != flag {
			continue
		}

		res = append(res, taskToGet)
	}

	return res, nil
}

// получение задач с определённым статусом
func (s *TaskStorage) GetStatusedWithLimit(ctx context.Context, page, limit int, status task.Status) ([]*task.Task, error) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	offset := (page - 1) * limit
	res := []*task.Task{}

	for i := offset; i < len(s.ids); i++ {
		if len(res) >= limit {
			break
		}

		taskToGet := s.storage[s.ids[i]]
		if taskToGet.Status != status {
			continue
		}

		res = append(res, taskToGet)
	}

	return res, nil
}

func (s *TaskStorage) GetTasksDueBefore(ctx context.Context, deadline time.Time, limit int) ([]*task.Task, error) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	var tasks []*task.Task
	found := 0

	for i := 0; i < len(s.ids); i++ {
		if found >= limit {
			break
		}

		t := s.storage[s.ids[i]]

		if t.Flag == task.FlagActive &&
			t.Status != task.StatusDone &&
			t.Status != task.StatusOverdue &&
			t.DueTime.Before(deadline) {

			tasks = append(tasks, t)
			found++
		}
	}

	return tasks, nil
}
