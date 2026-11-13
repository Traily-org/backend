-- Temporary or permanent bans of an account, and the identifiers (hashed
-- normalized emails) that stop a banned person from signing up again, even
-- after deleting the account.

CREATE TABLE user_bans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL
        REFERENCES users(id)
        ON DELETE CASCADE,
    -- The report that led to the ban. The API always sets it; it only becomes
    -- NULL when the report is deleted, e.g. with the reporter's account.
    report_id UUID
        REFERENCES reports(id)
        ON DELETE SET NULL,
    reason TEXT NOT NULL,
    banned_by UUID
        REFERENCES users(id)
        ON DELETE SET NULL,
    -- NULL means permanent.
    expires_at TIMESTAMPTZ,
    lifted_at TIMESTAMPTZ,
    lifted_by UUID
        REFERENCES users(id)
        ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT user_bans_expires_after_creation
        CHECK (expires_at IS NULL OR expires_at > created_at)
);

-- A ban is active while lifted_at IS NULL AND (expires_at IS NULL OR
-- expires_at > now()); this index serves that lookup.
CREATE INDEX idx_user_bans_user_active
    ON user_bans(user_id)
    WHERE lifted_at IS NULL;

CREATE INDEX idx_user_bans_report
    ON user_bans(report_id);

-- Not tied to users: these rows must survive the deletion of the account.
CREATE TABLE banned_identifiers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type TEXT NOT NULL
        CONSTRAINT banned_identifiers_type_valid CHECK (type IN ('email')),
    -- SHA-256 hex of the normalized value; the value itself is never stored.
    value_hash TEXT NOT NULL,
    user_ban_id UUID
        REFERENCES user_bans(id)
        ON DELETE SET NULL,
    -- Copied from the ban. NULL means permanent.
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT banned_identifiers_type_value_unique UNIQUE (type, value_hash)
);

CREATE INDEX idx_banned_identifiers_user_ban
    ON banned_identifiers(user_ban_id);

-- Logged in moderation_logs when a ban is lifted. Not used in this migration,
-- so adding it inside the migration transaction is allowed.
ALTER TYPE mod_action ADD VALUE IF NOT EXISTS 'unban';
