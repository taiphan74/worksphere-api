-- Create workspace_invitations table
CREATE TABLE workspace_invitations (
    id UUID PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    token VARCHAR(255) NOT NULL UNIQUE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    accepted_at TIMESTAMPTZ,
    declined_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_invitation_status
        CHECK (status IN ('pending', 'accepted', 'declined', 'cancelled')),

    -- Unique constraint: only one pending invitation per email per workspace
    CONSTRAINT uq_workspace_invitations_workspace_email_pending
        UNIQUE (workspace_id, email, status)
);

-- Index for looking up invitations by token (for accept/decline links)
CREATE INDEX idx_workspace_invitations_token
    ON workspace_invitations(token);

-- Index for looking up pending invitations by workspace
CREATE INDEX idx_workspace_invitations_workspace_status
    ON workspace_invitations(workspace_id, status);

-- Index for looking up invitations by email
CREATE INDEX idx_workspace_invitations_email
    ON workspace_invitations(email);
