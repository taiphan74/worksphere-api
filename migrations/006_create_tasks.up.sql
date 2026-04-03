CREATE TABLE tasks (
    id UUID PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    creator_id UUID NOT NULL REFERENCES users(id),
    assignee_id UUID REFERENCES users(id),
    title VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(30) NOT NULL DEFAULT 'TODO',
    priority VARCHAR(20) NOT NULL DEFAULT 'MEDIUM',
    due_at TIMESTAMPTZ,
    sprint_points INT,
    parent_id UUID REFERENCES tasks(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT chk_tasks_status
        CHECK (status IN ('TODO', 'IN_PROGRESS', 'IN_REVIEW', 'DONE')),

    CONSTRAINT chk_tasks_priority
        CHECK (priority IN ('LOW', 'MEDIUM', 'HIGH', 'URGENT')),

    CONSTRAINT chk_tasks_sprint_points
        CHECK (sprint_points IS NULL OR sprint_points >= 0)
);

CREATE INDEX idx_tasks_workspace_id
    ON tasks(workspace_id);

CREATE INDEX idx_tasks_assignee_id
    ON tasks(assignee_id);

CREATE INDEX idx_tasks_status
    ON tasks(status);

CREATE INDEX idx_tasks_due_at
    ON tasks(due_at);

CREATE INDEX idx_tasks_workspace_status
    ON tasks(workspace_id, status);

CREATE INDEX idx_tasks_parent_id
    ON tasks(parent_id);

CREATE INDEX idx_tasks_deleted_at
    ON tasks(deleted_at)
    WHERE deleted_at IS NULL;
