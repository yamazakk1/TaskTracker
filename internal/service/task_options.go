package service

import (
	"taskTracker/internal/models"
	"time"
)

// есть тип функции, которая возвращает тот-же объект
// грубо говоря это функция подтверждения обновления - она вернёт наш объект из другой функции
type TaskOption func(*models.Task)

// пример функции, которая получает какой то параметр и возвращает функцию обновления
func WithTitle(title string) TaskOption {
	return func(task *models.Task) {
		task.Title = title
	}
}

func WithDescription(description string) TaskOption {
	return func(task *models.Task) {
		task.Description = description
	}
}
func WithStatus(status models.Status) TaskOption {
	return func(task *models.Task) {
		task.Status = status
	}
}

func WithDueTime(dueTime time.Time) TaskOption {
	return func(task *models.Task) {
		task.DueTime = dueTime
	}
}
