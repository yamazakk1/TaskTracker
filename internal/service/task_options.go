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
	if description == "" {
		return nil
	}
	return func(task *models.Task) {
		task.Description = description
	}
}
func WithStatus(status models.Status) TaskOption {
	if status == "" {
		return nil
	}
	return func(task *models.Task) {
		task.Status = status
	}
}

func WithDueTime(dueTime time.Time) TaskOption {
	if dueTime.IsZero() {
		return nil
	}
	if time.Now().After(dueTime) {
		return nil
	}
	return func(task *models.Task) {
		task.DueTime = dueTime
	}
}
