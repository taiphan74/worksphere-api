CREATE TABLE workspace_members (
    id UUID PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_workspace_member_role
        CHECK (role IN ('OWNER', 'MEMBER')),

    CONSTRAINT uq_workspace_members_workspace_user
        UNIQUE (workspace_id, user_id)
);

CREATE INDEX idx_workspace_members_workspace_id
    ON workspace_members(workspace_id);

CREATE INDEX idx_workspace_members_user_id
    ON workspace_members(user_id);

CREATE INDEX idx_workspace_members_workspace_role
    ON workspace_members(workspace_id, role);
