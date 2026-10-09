CREATE TABLE publication (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL
        REFERENCES users(id)
        ON DELETE CASCADE,
    track_id UUID NOT NULL
        REFERENCES track(id)
        ON DELETE RESTRICT,
    trace_id UUID
        REFERENCES trace(id)
        ON DELETE SET NULL,
    title VARCHAR(200),
    content TEXT,
    published_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_publication_user
    ON publication(user_id);

CREATE INDEX idx_publication_track
    ON publication(track_id);

CREATE INDEX idx_publication_trace
    ON publication(trace_id);

CREATE INDEX idx_publication_date
    ON publication(published_at DESC);
