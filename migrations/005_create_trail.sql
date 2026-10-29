CREATE TABLE trail (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID
        REFERENCES "user"(id)
        ON DELETE SET NULL,
    name TEXT NOT NULL,
    description TEXT,
    difficulty trail_difficulty NOT NULL DEFAULT 'moderate',
    distance_m DOUBLE PRECISION NOT NULL,
    elevation_gain_m DOUBLE PRECISION NOT NULL,
    elevation_loss_m DOUBLE PRECISION NOT NULL,
    estimated_duration_s INTEGER NOT NULL,
    path GEOMETRY(LineString, 4326) NOT NULL,
    is_official BOOLEAN NOT NULL DEFAULT false,
    is_published BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_trail_path
    ON trail USING GIST(path);

CREATE TABLE waypoint (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trail_id UUID NOT NULL
        REFERENCES trail(id)
        ON DELETE CASCADE,
    label TEXT NOT NULL,
    description TEXT,
    order_index INTEGER NOT NULL,
    location GEOMETRY(Point, 4326) NOT NULL,
    photo_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_waypoint_trail_order
    ON waypoint(trail_id, order_index);

CREATE INDEX idx_waypoint_location
    ON waypoint USING GIST(location);

CREATE INDEX idx_trail_user
    ON trail(user_id);
