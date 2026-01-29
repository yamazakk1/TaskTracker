package service

import (
	"context"
	"fmt"
	"taskTracker/internal/logger"
	"taskTracker/internal/models/task"
	"taskTracker/internal/repository"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type TaskService struct {
	Repo     TaskRepository
	RepoType RepoType
}

type RepoType string

const DBType RepoType = "DB"
const InMemoryType RepoType = "IM"

func NewTaskService(repo TaskRepository, repoType RepoType) TaskService {
	return TaskService{
		Repo:     repo,
		RepoType: repoType,
	}
}

func (s *TaskService) HealthCheck(ctx context.Context) error {
	if err := s.Repo.HealthCheck(ctx); err != nil {
		return fmt.Errorf("проверка здоровья сервиса: %w", err)
	}
	return nil
}

// POST /tasks/{id}/archive
func (s *TaskService) ArchiveTask(ctx context.Context, id uuid.UUID) (*task.Task, error) {
	// Получаем задачу
	taskToArchive, err := s.Repo.GetByID(ctx, id)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, NewNotFound(s.RepoType, id.String())
		}
		return nil, fmt.Errorf("получение задачи: %w", err)
	}

	// Бизнес-правила архивации
	if taskToArchive.Flag == task.FlagArchived {
		return nil, NewBusinessError(
			"ALREADY_ARCHIVED",
			"Задача уже находится в архиве",
			ToDetail("task_id", id.String()),
			ToDetail("archived_at", taskToArchive.UpdatedAt),
		)
	}

	if taskToArchive.Flag == task.FlagDeleted {
		return nil, NewBusinessError(
			"TASK_DELETED",
			"Невозможно архивировать удаленную задачу",
			ToDetail("task_id", id.String()),
			ToDetail("deleted_at", taskToArchive.DeletedAt),
			ToDetail("can_restore", true),
		)
	}

	if taskToArchive.Flag != task.FlagActive {
		return nil, NewBusinessError(
			"INVALID_FLAG",
			fmt.Sprintf("Невозможно архивировать задачу с флагом '%s'", taskToArchive.Flag),
			ToDetail("task_id", id.String()),
			ToDetail("current_flag", taskToArchive.Flag),
			ToDetail("allowed_flags", []task.Flag{task.FlagActive}),
		)
	}

	// Выполняем архивацию
	taskToArchive.Flag = task.FlagArchived
	now := time.Now()
	taskToArchive.UpdatedAt = &now

	if err := s.Repo.Update(ctx, taskToArchive); err != nil {
		if err == repository.ErrVersionConflict {
			return nil, NewBusinessError(
				"VERSION_CONFLICT",
				"Задача была изменена другим пользователем",
				ToDetail("task_id", id.String()),
				ToDetail("suggestion", "Обновите страницу и попробуйте снова"),
			)
		}
		return nil, fmt.Errorf("обновление задачи при архивации: %w", err)
	}

	return taskToArchive, nil
}

// POST /tasks/{id}/unarchive
func (s *TaskService) UnarchiveTask(ctx context.Context, id uuid.UUID) (*task.Task, error) {
	taskToUnarchive, err := s.Repo.GetByID(ctx, id)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, NewNotFound(s.RepoType, id.String())
		}
		return nil, fmt.Errorf("получение задачи: %w", err)
	}

	// Бизнес-правила разархивации
	if taskToUnarchive.Flag == task.FlagActive {
		return nil, NewBusinessError(
			"NOT_ARCHIVED",
			"Задача уже активна",
			ToDetail("task_id", id.String()),
		)
	}

	if taskToUnarchive.Flag == task.FlagDeleted {
		return nil, NewBusinessError(
			"TASK_DELETED",
			"Невозможно разархивировать удаленную задачу",
			ToDetail("task_id", id.String()),
			ToDetail("deleted_at", taskToUnarchive.DeletedAt),
			ToDetail("can_restore", true),
		)
	}

	if taskToUnarchive.Flag != task.FlagArchived {
		return nil, NewBusinessError(
			"INVALID_FLAG",
			fmt.Sprintf("Можно разархивировать только архивные задачи. Текущий флаг: '%s'", taskToUnarchive.Flag),
			ToDetail("task_id", id.String()),
			ToDetail("current_flag", taskToUnarchive.Flag),
			ToDetail("allowed_flags", []task.Flag{task.FlagArchived}),
		)
	}

	// Выполняем разархивацию
	taskToUnarchive.Flag = task.FlagActive
	now := time.Now()
	taskToUnarchive.UpdatedAt = &now

	if err := s.Repo.Update(ctx, taskToUnarchive); err != nil {
		if err == repository.ErrVersionConflict {
			return nil, NewBusinessError(
				"VERSION_CONFLICT",
				"Задача была изменена другим пользователем",
				ToDetail("task_id", id.String()),
			)
		}
		return nil, fmt.Errorf("обновление задачи при разархивации: %w", err)
	}

	return taskToUnarchive, nil
}

// GET /tasks/all
func (s *TaskService) GetAllTasks(ctx context.Context, page, limit int) ([]*task.Task, error) {
	tasks, err := s.Repo.GetAllWithLimit(ctx, page, limit)
	if err != nil {
		return nil, fmt.Errorf("получение всех задач: %w", err)
	}
	return tasks, nil
}

// POST /admin/tasks/{id}/restore
func (s *TaskService) RestoreTask(ctx context.Context, id uuid.UUID) (*task.Task, error) {
	taskToRestore, err := s.Repo.GetByID(ctx, id)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, NewNotFound(s.RepoType, id.String())
		}
		return nil, fmt.Errorf("получение задачи: %w", err)
	}

	if taskToRestore.Flag != task.FlagDeleted {
		return nil, NewBusinessError(
			"NOT_DELETED",
			fmt.Sprintf("Задача не была удалена. Текущий флаг: '%s'", taskToRestore.Flag),
			ToDetail("task_id", id.String()),
			ToDetail("current_flag", taskToRestore.Flag),
		)
	}

	if taskToRestore.DeletedAt != nil {
		restoreDeadline := taskToRestore.DeletedAt.Add(30 * 24 * time.Hour)
		if time.Now().After(restoreDeadline) {
			return nil, NewBusinessError(
				"RESTORE_EXPIRED",
				"Срок восстановления истек (30 дней)",
				ToDetail("task_id", id.String()),
				ToDetail("deleted_at", taskToRestore.DeletedAt),
				ToDetail("restore_deadline", restoreDeadline),
			)
		}
	}

	taskToRestore.Flag = task.FlagActive
	taskToRestore.DeletedAt = nil
	now := time.Now()
	taskToRestore.UpdatedAt = &now

	if err := s.Repo.Update(ctx, taskToRestore); err != nil {
		if err == repository.ErrVersionConflict {
			return nil, NewBusinessError(
				"VERSION_CONFLICT",
				"Задача была изменена в корзине",
				ToDetail("task_id", id.String()),
			)
		}
		return nil, fmt.Errorf("восстановление задачи: %w", err)
	}

	return taskToRestore, nil
}

// DELETE /admin/tasks/{id}/purge
func (s *TaskService) PurgeTask(ctx context.Context, id uuid.UUID) error {
	// Проверяем, что задача существует и удалена
	taskToPurge, err := s.Repo.GetByID(ctx, id)
	if err != nil {
		if err == repository.ErrNotFound {
			return NewNotFound(s.RepoType, id.String())
		}
		return fmt.Errorf("получение задачи: %w", err)
	}

	if taskToPurge.Flag != task.FlagDeleted {
		return NewBusinessError(
			"NOT_DELETED",
			fmt.Sprintf("Можно полностью удалять только удаленные задачи. Текущий флаг: '%s'", taskToPurge.Flag),
			ToDetail("task_id", id.String()),
			ToDetail("current_flag", taskToPurge.Flag),
			ToDetail("suggestion", "Сначала выполните мягкое удаление"),
		)
	}

	// Полное удаление
	if err := s.Repo.DeleteFull(ctx, id); err != nil {
		return fmt.Errorf("полное удаление задачи: %w", err)
	}

	return nil
}

// DELETE /tasks/{id}
func (s *TaskService) DeleteTask(ctx context.Context, id uuid.UUID) error {
	taskToDelete, err := s.Repo.GetByID(ctx, id)
	if err != nil {
		if err == repository.ErrNotFound {
			return NewNotFound(s.RepoType, id.String())
		}
		return fmt.Errorf("получение задачи: %w", err)
	}

	if taskToDelete.Flag == task.FlagDeleted {
		return NewBusinessError(
			"ALREADY_DELETED",
			"Задача уже удалена",
			ToDetail("task_id", id.String()),
			ToDetail("deleted_at", taskToDelete.DeletedAt),
		)
	}

	if taskToDelete.Status == task.StatusInProgress {
		return NewBusinessError(
			"IN_PROGRESS",
			"Нельзя удалять задачу в процессе выполнения",
			ToDetail("task_id", id.String()),
			ToDetail("current_status", taskToDelete.Status),
			ToDetail("suggestion", "Завершите задачу перед удалением"),
		)
	}

	now := time.Now()
	taskToDelete.Flag = task.FlagDeleted
	taskToDelete.DeletedAt = &now
	taskToDelete.UpdatedAt = &now

	if err := s.Repo.DeleteSoft(ctx, taskToDelete); err != nil {
		if err == repository.ErrVersionConflict {
			return NewBusinessError(
				"VERSION_CONFLICT",
				"Задача была изменена другим пользователем",
				ToDetail("task_id", id.String()),
			)
		}
		return fmt.Errorf("мягкое удаление задачи: %w", err)
	}

	return nil
}

// POST /tasks
func (s *TaskService) CreateTask(ctx context.Context, title, description string, dueTime time.Time) (*task.Task, error) {
	// Бизнес-логика: если дедлайн близко, сразу ставим "в работе"
    now := time.Now()
	status := task.StatusNew
	if time.Until(dueTime) < 24*time.Hour {
		status = task.StatusInProgress
	}else if dueTime.Before(now){
        status = task.StatusOverdue
    }

	newTask := &task.Task{
		UUID:        uuid.New(),
		Title:       title,
		Description: description,
		Status:      status,
		DueTime:     dueTime,
		CreatedAt:   time.Now(),
		Flag:        task.FlagActive,
		Version:     1,
	}

	if err := s.Repo.Create(ctx, newTask); err != nil {
		return nil, fmt.Errorf("создание задачи: %w", err)
	}

	return newTask, nil
}

// PUT /tasks/{id}
func (s *TaskService) UpdateTask(ctx context.Context, id uuid.UUID, options ...task.TaskOption) (*task.Task, error) {
	taskToUpdate, err := s.Repo.GetByID(ctx, id)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, NewNotFound(s.RepoType, id.String())
		}
		return nil, fmt.Errorf("получение задачи: %w", err)
	}

	if taskToUpdate.Flag != task.FlagActive {
		return nil, NewBusinessError(
			"INVALID_FLAG",
			fmt.Sprintf("Можно обновлять только активные задачи. Текущий флаг: '%s'", taskToUpdate.Flag),
			ToDetail("task_id", id.String()),
			ToDetail("current_flag", taskToUpdate.Flag),
		)
	}

	for _, opt := range options {
		opt(taskToUpdate)
	}

	if taskToUpdate.Status != task.StatusDone &&
		taskToUpdate.DueTime.Before(time.Now()) {
		taskToUpdate.Status = task.StatusOverdue
	}

	if taskToUpdate.Status == task.StatusDone {

		if taskToUpdate.CreatedAt.Add(30 * 24 * time.Hour).Before(time.Now()) {
			taskToUpdate.Flag = task.FlagArchived
		}
	}

	now := time.Now()
	taskToUpdate.UpdatedAt = &now

	if err := s.Repo.Update(ctx, taskToUpdate); err != nil {
		if err == repository.ErrVersionConflict {
			return nil, NewBusinessError(
				"VERSION_CONFLICT",
				"Задача была изменена другим пользователем",
				ToDetail("task_id", id.String()),
			)
		}
		return nil, fmt.Errorf("обновление задачи: %w", err)
	}

	return taskToUpdate, nil
}

// GET /tasks/{id}
func (s *TaskService) GetTaskByID(ctx context.Context, id uuid.UUID) (*task.Task, error) {
	taskGot, err := s.Repo.GetByID(ctx, id)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, NewNotFound(s.RepoType, id.String())
		}
		return nil, fmt.Errorf("получение задачи: %w", err)
	}

	// Бизнес-правило: не отдаем удаленные через основной API
	if taskGot.Flag == task.FlagDeleted {
		return nil, NewBusinessError(
			"TASK_DELETED",
			"Задача была удалена",
			ToDetail("task_id", id.String()),
			ToDetail("deleted_at", taskGot.DeletedAt),
			ToDetail("can_restore", true),
			ToDetail("restore_url", fmt.Sprintf("/admin/tasks/%s/restore", id)),
		)
	}

	// Автоматически помечаем просроченные
	if taskGot.Flag == task.FlagActive &&
		taskGot.Status != task.StatusDone &&
		taskGot.Status != task.StatusOverdue &&
		taskGot.DueTime.Before(time.Now()) {
		taskGot.Status = task.StatusOverdue
	}

	return taskGot, nil
}

// ТУТ НАДО ДОБАВИТ  ИНДЕКС
func (s *TaskService) GetOverdueTasks(ctx context.Context, page, limit int) ([]*task.Task, error) {
	overdueTasks, err := s.Repo.GetTasksDueBefore(ctx, time.Now(), limit)
	if err != nil {
		return nil, fmt.Errorf("получение просроченных задач: %w", err)
	}

	for _, t := range overdueTasks {
		if t.Status != task.StatusOverdue {
			t.Status = task.StatusOverdue
			if err := s.Repo.Update(ctx, t); err != nil {
				logger.Warn("Не удалось обновить статус задачи как просроченной",
					zap.String("task_id", t.UUID.String()),
					zap.Error(err))
			}
		}
	}

	return overdueTasks, nil
}

// ТУТ НАДО ДОБАВИТЬ ИНДЕКС
func (s *TaskService) GetArchivedTasks(ctx context.Context, page, limit int) ([]*task.Task, error) {
	tasks, err := s.Repo.GetFlaggedWithLimit(ctx, page, limit, task.FlagArchived)
	if err != nil {
		return nil, fmt.Errorf("получение архивных задач: %w", err)
	}
	return tasks, nil
}

// ТУТ НАДО ДОБАВИТЬ ИНДЕКС
func (s *TaskService) GetDeletedTasks(ctx context.Context, page, limit int) ([]*task.Task, error) {
	tasks, err := s.Repo.GetFlaggedWithLimit(ctx, page, limit, task.FlagDeleted)
	if err != nil {
		return nil, fmt.Errorf("получение удаленных задач: %w", err)
	}
	return tasks, nil
}

// ТУТ НАДО ДОБАВИТ ИНДЕКС
func (s *TaskService) GetActiveTasks(ctx context.Context, page, limit int) ([]*task.Task, error) {
	tasks, err := s.Repo.GetFlaggedWithLimit(ctx, page, limit, task.FlagActive)
	if err != nil {
		return nil, fmt.Errorf("получение активных задач: %w", err)
	}

	now := time.Now()
	for _, t := range tasks {
		if t.Status != task.StatusDone &&
			t.Status != task.StatusOverdue &&
			t.DueTime.Before(now) {
			t.Status = task.StatusOverdue
		}
	}

	return tasks, nil
}
