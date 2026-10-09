CREATE TABLE IF NOT EXISTS photos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL
        REFERENCES users(id)
        ON DELETE CASCADE,
    storage_key TEXT NOT NULL,
    url TEXT NOT NULL,
    location GEOMETRY(Point, 4326),
    visibility visibility_level NOT NULL DEFAULT 'public',
    nsfw_status nsfw_status NOT NULL DEFAULT 'pending',
    taken_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_photos_location
    ON photos USING GIST(location);
