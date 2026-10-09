CREATE TABLE activity (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL
        REFERENCES users(id)
        ON DELETE CASCADE,
    track_id UUID NOT NULL UNIQUE
        REFERENCES track(id)
        ON DELETE RESTRICT,
    trace_id UUID
        REFERENCES trace(id)
        ON DELETE SET NULL,
    name VARCHAR(150),
    started_at TIMESTAMPTZ,
    ended_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT activity_dates_valid
        CHECK (
            ended_at IS NULL
            OR started_at IS NULL
            OR ended_at >= started_at
        )
);

CREATE INDEX idx_activity_user
    ON activity(user_id);

CREATE INDEX idx_activity_trace
    ON activity(trace_id);
