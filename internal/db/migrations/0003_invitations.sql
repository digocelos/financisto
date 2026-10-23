CREATE TABLE IF NOT EXISTS invitations (
    id UUID PRIMARY KEY,
    family_id UUID NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    member_id UUID NOT NULL REFERENCES family_members(id) ON DELETE CASCADE,
    invited_by UUID NOT NULL REFERENCES users(id),
    email TEXT NOT NULL,
    token_hash BYTEA NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS invitations_member_idx ON invitations (member_id, created_at DESC);
CREATE INDEX IF NOT EXISTS invitations_active_token_idx ON invitations (token_hash, expires_at)
    WHERE accepted_at IS NULL AND revoked_at IS NULL;
