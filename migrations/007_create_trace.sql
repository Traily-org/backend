CREATE TABLE trace (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL
        REFERENCES "user"(id)
        ON DELETE CASCADE,
    track_id UUID NOT NULL
        REFERENCES track(id)
        ON DELETE RESTRICT,
    name VARCHAR(150) NOT NULL,
    description TEXT,
    difficulty trail_difficulty,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_trace_user
    ON trace(user_id);

CREATE INDEX idx_trace_track
    ON trace(track_id);
