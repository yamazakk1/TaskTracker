package inmemory_test

import (
	"context"
	"fmt"
	"sync"
	"taskTracker/internal/models/task"
	"taskTracker/internal/repository"
	"taskTracker/internal/repository/task/inmemory"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTaskStorage_New тестирует создание хранилища
func TestTaskStorage_New(t *testing.T) {
	storage := inmemory.NewTaskStorage()
	assert.NotNil(t, storage)
}

// TestTaskStorage_HealthCheck тестирует проверку здоровья
func TestTaskStorage_HealthCheck(t *testing.T) {
	ctx := context.Background()
	storage := inmemory.NewTaskStorage()

	err := storage.HealthCheck(ctx)
	assert.NoError(t, err)
}

// TestTaskStorage_Create тестирует создание задачи
func TestTaskStorage_Create(t *testing.T) {
	ctx := context.Background()
	storage := inmemory.NewTaskStorage()

	taskToCreate := &task.Task{
		UUID:        uuid.New(),
		Title:       "Test Task",
		Description: "Test Description",
		Status:      task.StatusNew,
		DueTime:     time.Now().Add(24 * time.Hour),
	}

	err := storage.Create(ctx, taskToCreate)
	require.NoError(t, err)

	// Проверяем, что поля заполнены
	assert.False(t, taskToCreate.CreatedAt.IsZero())
	assert.Equal(t, task.FlagActive, taskToCreate.Flag)

	// Проверяем, что задача сохранена
	retrievedTask, err := storage.GetByID(ctx, taskToCreate.UUID)
	require.NoError(t, err)
	assert.Equal(t, "Test Task", retrievedTask.Title)
}

// TestTaskStorage_GetByID тестирует получение задачи по ID
func TestTaskStorage_GetByID(t *testing.T) {
	ctx := context.Background()
	storage := inmemory.NewTaskStorage()

	// Создаем задачу
	taskID := uuid.New()
	taskToCreate := &task.Task{
		UUID:    taskID,
		Title:   "Test Get Task",
		Status:  task.StatusInProgress,
		DueTime: time.Now().Add(24 * time.Hour),
	}

	err := storage.Create(ctx, taskToCreate)
	require.NoError(t, err)

	// Получаем задачу
	retrievedTask, err := storage.GetByID(ctx, taskID)
	require.NoError(t, err)
	assert.Equal(t, taskID, retrievedTask.UUID)
	assert.Equal(t, "Test Get Task", retrievedTask.Title)

	// Пытаемся получить несуществующую задачу
	nonExistentID := uuid.New()
	_, err = storage.GetByID(ctx, nonExistentID)
	assert.Error(t, err)
	assert.Equal(t, repository.ErrNotFound, err)
}

// TestTaskStorage_Update тестирует обновление задачи
func TestTaskStorage_Update(t *testing.T) {
	ctx := context.Background()
	storage := inmemory.NewTaskStorage()

	// Создаем задачу
	taskToCreate := &task.Task{
		UUID:    uuid.New(),
		Title:   "Original Title",
		Status:  task.StatusNew,
		DueTime: time.Now().Add(24 * time.Hour),
	}

	err := storage.Create(ctx, taskToCreate)
	require.NoError(t, err)

	// Обновляем задачу
	taskToCreate.Title = "Updated Title"
	taskToCreate.Description = "Updated Description"
	taskToCreate.Status = task.StatusInProgress
	taskToCreate.Version = 1 // Начальная версия

	err = storage.Update(ctx, taskToCreate)
	require.NoError(t, err)

	// Проверяем обновление
	retrievedTask, err := storage.GetByID(ctx, taskToCreate.UUID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", retrievedTask.Title)
	assert.Equal(t, "Updated Description", retrievedTask.Description)
	assert.Equal(t, task.StatusInProgress, retrievedTask.Status)
	assert.True(t, retrievedTask.UpdatedAt != nil)
	assert.Equal(t, 2, retrievedTask.Version) // Версия должна увеличиться
}

// TestTaskStorage_DeleteSoft тестирует мягкое удаление
func TestTaskStorage_DeleteSoft(t *testing.T) {
	ctx := context.Background()
	storage := inmemory.NewTaskStorage()

	// Создаем задачу
	taskToCreate := &task.Task{
		UUID:    uuid.New(),
		Title:   "Task to delete",
		Status:  task.StatusNew,
		DueTime: time.Now().Add(24 * time.Hour),
	}

	err := storage.Create(ctx, taskToCreate)
	require.NoError(t, err)

	// Мягко удаляем
	err = storage.DeleteSoft(ctx, taskToCreate)
	require.NoError(t, err)

	// Проверяем, что задача помечена как удаленная
	retrievedTask, err := storage.GetByID(ctx, taskToCreate.UUID)
	require.NoError(t, err)
	assert.Equal(t, task.FlagDeleted, retrievedTask.Flag)
	assert.True(t, retrievedTask.DeletedAt != nil)
	assert.True(t, retrievedTask.UpdatedAt != nil)
}

// TestTaskStorage_DeleteFull тестирует полное удаление
func TestTaskStorage_DeleteFull(t *testing.T) {
	ctx := context.Background()
	storage := inmemory.NewTaskStorage()

	// Создаем задачу
	taskID := uuid.New()
	taskToCreate := &task.Task{
		UUID:    taskID,
		Title:   "Task to purge",
		Status:  task.StatusNew,
		DueTime: time.Now().Add(24 * time.Hour),
	}

	err := storage.Create(ctx, taskToCreate)
	require.NoError(t, err)

	// Полное удаление
	err = storage.DeleteFull(ctx, taskID)
	require.NoError(t, err)

	// Проверяем, что задачи больше нет
	_, err = storage.GetByID(ctx, taskID)
	assert.Error(t, err)
	assert.Equal(t, repository.ErrNotFound, err)
}

// TestTaskStorage_GetAllWithLimit тестирует получение всех задач с пагинацией
func TestTaskStorage_GetAllWithLimit(t *testing.T) {
	ctx := context.Background()
	storage := inmemory.NewTaskStorage()

	// Создаем несколько задач
	for i := 1; i <= 5; i++ {
		taskToCreate := &task.Task{
			UUID:    uuid.New(),
			Title:   fmt.Sprintf("Task %d", i),
			Status:  task.StatusNew,
			DueTime: time.Now().Add(time.Duration(i) * 24 * time.Hour),
			Flag:    task.FlagActive,
		}
		err := storage.Create(ctx, taskToCreate)
		require.NoError(t, err)
	}

	// Создаем удаленную задачу
	deletedTask := &task.Task{
		UUID:    uuid.New(),
		Title:   "Deleted Task",
		Status:  task.StatusNew,
		DueTime: time.Now().Add(24 * time.Hour),
		Flag:    task.FlagActive,
	}
	err := storage.Create(ctx, deletedTask)
	require.NoError(t, err)
	err = storage.DeleteSoft(ctx, deletedTask)
	require.NoError(t, err)

	// Получаем все активные задачи (удаленные не должны включаться)
	tasks, err := storage.GetAllWithLimit(ctx, 1, 10)
	require.NoError(t, err)
	assert.Len(t, tasks, 5)

	// Тестируем пагинацию
	tasksPage1, err := storage.GetAllWithLimit(ctx, 1, 2)
	require.NoError(t, err)
	assert.Len(t, tasksPage1, 2)

	tasksPage2, err := storage.GetAllWithLimit(ctx, 2, 2)
	require.NoError(t, err)
	assert.Len(t, tasksPage2, 2)

	tasksPage3, err := storage.GetAllWithLimit(ctx, 3, 2)
	require.NoError(t, err)
	assert.Len(t, tasksPage3, 1)
}

// TestTaskStorage_GetTasksDueBefore тестирует получение просроченных задач
func TestTaskStorage_GetTasksDueBefore(t *testing.T) {
	ctx := context.Background()
	storage := inmemory.NewTaskStorage()
	now := time.Now()

	// Создаем просроченную задачу
	overdueTask := &task.Task{
		UUID:    uuid.New(),
		Title:   "Overdue Task",
		Status:  task.StatusNew,
		DueTime: now.Add(-24 * time.Hour), // Просрочена
		Flag:    task.FlagActive,
	}
	err := storage.Create(ctx, overdueTask)
	require.NoError(t, err)

	// Создаем задачу на будущее
	futureTask := &task.Task{
		UUID:    uuid.New(),
		Title:   "Future Task",
		Status:  task.StatusNew,
		DueTime: now.Add(24 * time.Hour), // На будущее
		Flag:    task.FlagActive,
	}
	err = storage.Create(ctx, futureTask)
	require.NoError(t, err)

	// Создаем завершенную задачу (не должна включаться)
	doneTask := &task.Task{
		UUID:    uuid.New(),
		Title:   "Done Task",
		Status:  task.StatusDone,
		DueTime: now.Add(-48 * time.Hour), // Просрочена, но завершена
		Flag:    task.FlagActive,
	}
	err = storage.Create(ctx, doneTask)
	require.NoError(t, err)

	// Создаем уже помеченную как просроченную задачу
	alreadyOverdueTask := &task.Task{
		UUID:    uuid.New(),
		Title:   "Already Overdue Task",
		Status:  task.StatusOverdue,
		DueTime: now.Add(-72 * time.Hour),
		Flag:    task.FlagActive,
	}
	err = storage.Create(ctx, alreadyOverdueTask)
	require.NoError(t, err)

	// Получаем просроченные задачи
	overdueTasks, err := storage.GetTasksDueBefore(ctx, now, 10)
	require.NoError(t, err)
	assert.Len(t, overdueTasks, 1) // Только одна новая просроченная
	assert.Equal(t, "Overdue Task", overdueTasks[0].Title)
}

// TestTaskStorage_ConcurrentAccess тестирует конкурентный доступ
func TestTaskStorage_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	storage := inmemory.NewTaskStorage()
	taskCount := 100
	goroutines := 10

	var wg sync.WaitGroup
	errors := make(chan error, taskCount*goroutines)

	// Создаем задачи конкурентно
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < taskCount/goroutines; j++ {
				taskToCreate := &task.Task{
					UUID:    uuid.New(),
					Title:   fmt.Sprintf("Task %d-%d", workerID, j),
					Status:  task.StatusNew,
					DueTime: time.Now().Add(time.Duration(j) * time.Hour),
					Flag:    task.FlagActive,
				}
				if err := storage.Create(ctx, taskToCreate); err != nil {
					errors <- err
				}
			}
		}(i)
	}
	wg.Wait()
	close(errors)

	// Проверяем, что нет ошибок
	for err := range errors {
		assert.NoError(t, err)
	}

	// Проверяем, что все задачи созданы
	tasks, err := storage.GetAllWithLimit(ctx, 1, taskCount*2)
	require.NoError(t, err)
	assert.Len(t, tasks, taskCount)
}

// TestTaskStorage_EdgeCases тестирует граничные случаи
func TestTaskStorage_EdgeCases(t *testing.T) {
	ctx := context.Background()
	storage := inmemory.NewTaskStorage()

	t.Run("empty storage operations", func(t *testing.T) {
		// Получение из пустого хранилища
		tasks, err := storage.GetAllWithLimit(ctx, 1, 10)
		require.NoError(t, err)
		assert.Empty(t, tasks)

		// Обновление несуществующей задачи
		nonExistentTask := &task.Task{UUID: uuid.New()}
		err = storage.Update(ctx, nonExistentTask)
		require.NoError(t, err) // В текущей реализации это не возвращает ошибку

		// Мягкое удаление несуществующей задачи
		err = storage.DeleteSoft(ctx, nonExistentTask)
		require.NoError(t, err) // В текущей реализации это не возвращает ошибку

		// Полное удаление несуществующей задачи
		err = storage.DeleteFull(ctx, uuid.New())
		require.NoError(t, err)
	})

	t.Run("pagination edge cases", func(t *testing.T) {
		// Создаем одну задачу
		taskToCreate := &task.Task{
			UUID:    uuid.New(),
			Title:   "Single Task",
			Status:  task.StatusNew,
			DueTime: time.Now().Add(24 * time.Hour),
			Flag:    task.FlagActive,
		}
		err := storage.Create(ctx, taskToCreate)
		require.NoError(t, err)

		// Страница за пределами данных
		tasks, err := storage.GetAllWithLimit(ctx, 100, 10)
		require.NoError(t, err)
		assert.Empty(t, tasks)

		// Нулевой лимит
		tasks, err = storage.GetAllWithLimit(ctx, 1, 0)
		require.NoError(t, err)
		assert.Empty(t, tasks)

		// Негативный offset
		tasks, err = storage.GetAllWithLimit(ctx, 0, 10)
		require.NoError(t, err)
		assert.Empty(t, tasks) // Так как offset = (0-1)*10 = -10, должно вернуть пустой список
	})

	t.Run("task with zero time", func(t *testing.T) {
		taskToCreate := &task.Task{
			UUID:    uuid.New(),
			Title:   "Task with zero time",
			Status:  task.StatusNew,
			DueTime: time.Time{}, // Zero time
			Flag:    task.FlagActive,
		}
		err := storage.Create(ctx, taskToCreate)
		require.NoError(t, err)

		// Задача с нулевым временем должна считаться просроченной
		overdueTasks, err := storage.GetTasksDueBefore(ctx, time.Now(), 10)
		require.NoError(t, err)
		assert.NotEmpty(t, overdueTasks)
	})
}

// TestTaskStorage_DeleteSoft_NotFound тестирует удаление несуществующей задачи
func TestTaskStorage_DeleteSoft_NotFound(t *testing.T) {
	ctx := context.Background()
	storage := inmemory.NewTaskStorage()

	// Создаем задачу
	taskToCreate := &task.Task{
		UUID:    uuid.New(),
		Title:   "Task to delete",
		Status:  task.StatusNew,
		DueTime: time.Now().Add(24 * time.Hour),
	}
	err := storage.Create(ctx, taskToCreate)
	require.NoError(t, err)

	// Сначала получаем задачу
	taskFromStorage, err := storage.GetByID(ctx, taskToCreate.UUID)
	require.NoError(t, err)

	// Удаляем другую задачу (несуществующую)
	anotherTask := &task.Task{UUID: uuid.New()}
	err = storage.DeleteSoft(ctx, anotherTask)
	// В текущей реализации DeleteSoft не возвращает ошибку для несуществующей задачи
	assert.NoError(t, err)

	// Проверяем, что исходная задача не затронута
	checkTask, err := storage.GetByID(ctx, taskToCreate.UUID)
	require.NoError(t, err)
	assert.Equal(t, task.FlagActive, checkTask.Flag)
	assert.Equal(t, taskFromStorage.Version, checkTask.Version)
}

// TestTaskStorage_Update_NonExistent тестирует обновление несуществующей задачи
func TestTaskStorage_Update_NonExistent(t *testing.T) {
	ctx := context.Background()
	storage := inmemory.NewTaskStorage()

	// Обновляем несуществующую задачу
	nonExistentTask := &task.Task{
		UUID:    uuid.New(),
		Title:   "Non-existent Task",
		Status:  task.StatusNew,
		DueTime: time.Now().Add(24 * time.Hour),
		Version: 1,
	}

	err := storage.Update(ctx, nonExistentTask)
	require.NoError(t, err) // В текущей реализации это не возвращает ошибку

	// Проверяем, что задача не создалась
	_, err = storage.GetByID(ctx, nonExistentTask.UUID)
	assert.Error(t, err)
	assert.Equal(t, repository.ErrNotFound, err)
}

// TestTaskStorage_Versioning тестирует версионирование
func TestTaskStorage_Versioning(t *testing.T) {
	ctx := context.Background()
	storage := inmemory.NewTaskStorage()

	// Создаем задачу
	taskToCreate := &task.Task{
		UUID:    uuid.New(),
		Title:   "Versioned Task",
		Status:  task.StatusNew,
		DueTime: time.Now().Add(24 * time.Hour),
	}
	err := storage.Create(ctx, taskToCreate)
	require.NoError(t, err)

	// Проверяем начальную версию
	taskV1, err := storage.GetByID(ctx, taskToCreate.UUID)
	require.NoError(t, err)
	assert.Equal(t, 0, taskV1.Version) // После создания version = 0

	// Обновляем задачу
	taskV1.Title = "Updated Title"
	err = storage.Update(ctx, taskV1)
	require.NoError(t, err)

	// Проверяем, что версия увеличилась
	taskV2, err := storage.GetByID(ctx, taskToCreate.UUID)
	require.NoError(t, err)
	assert.Equal(t, 1, taskV2.Version)

	// Еще одно обновление
	taskV2.Description = "Updated Description"
	err = storage.Update(ctx, taskV2)
	require.NoError(t, err)

	taskV3, err := storage.GetByID(ctx, taskToCreate.UUID)
	require.NoError(t, err)
	assert.Equal(t, 2, taskV3.Version)
}

// TestTaskStorage_IdsSliceConsistency тестирует согласованность среза IDs
func TestTaskStorage_IdsSliceConsistency(t *testing.T) {
	ctx := context.Background()
	storage := inmemory.NewTaskStorage()

	// Создаем несколько задач
	tasks := make([]*task.Task, 5)
	for i := 0; i < 5; i++ {
		tasks[i] = &task.Task{
			UUID:    uuid.New(),
			Title:   fmt.Sprintf("Task %d", i),
			Status:  task.StatusNew,
			DueTime: time.Now().Add(time.Duration(i) * time.Hour),
		}
		err := storage.Create(ctx, tasks[i])
		require.NoError(t, err)
	}

	// Удаляем задачу из середины
	err := storage.DeleteFull(ctx, tasks[2].UUID)
	require.NoError(t, err)

	// Проверяем, что остальные задачи доступны
	for i, task := range tasks {
		if i == 2 {
			_, err := storage.GetByID(ctx, task.UUID)
			assert.Error(t, err)
			continue
		}
		retrievedTask, err := storage.GetByID(ctx, task.UUID)
		require.NoError(t, err)
		assert.Equal(t, task.Title, retrievedTask.Title)
	}

	// Проверяем GetAllWithLimit
	allTasks, err := storage.GetAllWithLimit(ctx, 1, 10)
	require.NoError(t, err)
	assert.Len(t, allTasks, 4) // Одна задача удалена
}