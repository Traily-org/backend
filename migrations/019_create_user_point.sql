CREATE TABLE IF NOT EXISTS user_point (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL
        REFERENCES "user"(id)
        ON DELETE CASCADE,
    points INTEGER NOT NULL,
    reason TEXT NOT NULL,
    reference_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_point_user_id
    ON user_point(user_id);
