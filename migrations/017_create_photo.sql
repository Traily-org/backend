CREATE TABLE IF NOT EXISTS photo (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL
        REFERENCES "user"(id)
        ON DELETE CASCADE,
    storage_key TEXT NOT NULL,
    url TEXT NOT NULL,
    location GEOMETRY(Point, 4326),
    visibility visibility_level NOT NULL DEFAULT 'public',
    nsfw_status nsfw_status NOT NULL DEFAULT 'pending',
    taken_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_photo_location
    ON photo USING GIST(location);

CREATE INDEX idx_photo_user
    ON photo(user_id);
