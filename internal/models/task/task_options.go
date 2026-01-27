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
	return func(task *Task) {
		task.Description = description
	}
}
func WithStatus(status Status) TaskOption {
	return func(task *Task) {
		task.Status = status
	}
}

func WithDueTime(dueTime time.Time) TaskOption {
	return func(task *Task) {
		task.DueTime = dueTime
	}
}

func WithFlag (flag Flag) TaskOption{
	return func(task *Task){
		task.Flag = flag
	}
}