package task

import (

	"time"
)

type TaskOption func(*Task)

func WithTitle(title string) TaskOption {
	return func(task *Task) {
		task.Title = title
	}
}

func WithDescription(description string) TaskOption {
	if description == "" {
		return nil
	}
	return func(task *Task) {
		task.Description = description
	}
}
func WithStatus(status Status) TaskOption {
	if status == "" {
		return nil
	}
	return func(task *Task) {
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
	return func(task *Task) {
		task.DueTime = dueTime
	}
}
