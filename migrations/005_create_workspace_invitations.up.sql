-- Create workspace_invitations table
-- Only pending invitations exist as records.
-- When accepted or declined, the record is deleted.
CREATE TABLE workspace_invitations (
    id UUID PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    token VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Only one invitation per email per workspace
    CONSTRAINT uq_workspace_invitations_workspace_email
        UNIQUE (workspace_id, email)
);

-- Index for looking up invitations by token (for accept/decline links)
CREATE INDEX idx_workspace_invitations_token
    ON workspace_invitations(token);

-- Index for looking up invitations by workspace
CREATE INDEX idx_workspace_invitations_workspace
    ON workspace_invitations(workspace_id);

-- Index for looking up invitations by email
CREATE INDEX idx_workspace_invitations_email
    ON workspace_invitations(email);
