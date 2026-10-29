CREATE TABLE photos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL
        REFERENCES users(id)
        ON DELETE CASCADE,
    trail_id UUID
        REFERENCES trails(id)
        ON DELETE SET NULL,
    waypoint_id UUID
        REFERENCES waypoints(id)
        ON DELETE SET NULL,
    storage_key TEXT NOT NULL UNIQUE,
    location GEOMETRY(Point, 4326),
    visibility visibility_level NOT NULL DEFAULT 'public',
    nsfw_status nsfw_status NOT NULL DEFAULT 'pending',
    taken_at TIMESTAMPTZ,
    confirmed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_photos_user
    ON photos(user_id);

CREATE INDEX idx_photos_trail
    ON photos(trail_id);

CREATE INDEX idx_photos_waypoint
    ON photos(waypoint_id);

CREATE INDEX idx_photos_location
    ON photos USING GIST(location);
