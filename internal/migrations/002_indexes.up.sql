CREATE INDEX idx_tasks_flag ON tasks(flag);

CREATE INDEX idx_tasks_active_created ON tasks(created_at DESC) 
WHERE flag = 'active';

CREATE INDEX idx_tasks_overdue ON tasks(due_time, status)
WHERE flag = 'active' AND status IN ('new', 'in progress');

CREATE INDEX idx_tasks_archived_created ON tasks(created_at DESC)
WHERE flag = 'archived';

CREATE INDEX idx_tasks_deleted_created ON tasks(created_at DESC)
WHERE flag = 'deleted';