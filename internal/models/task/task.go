package task

import (
	"time"

	"github.com/google/uuid"
)

type Task struct {
	UUID          uuid.UUID  `json:"uuid" db:"uuid"`
	Title       string     `json:"title" db:"title"`
	Description string     `json:"description" db:"description"`
	Status      Status     `json:"status" db:"status"`
	DueTime     time.Time  `json:"due_time" db:"due_time"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty" db:"updated_at,omitempty"`
	Version     int       `db:"version" json:"version"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty" db:"deleted_at,omitempty"`
}

type Status string

const StatusNew Status = "new"
const StatusDone Status = "done"
const StatusInProgress Status = "in progress"
const StatusDeleted Status = "deleted"
