CREATE TABLE trails (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID
        REFERENCES users(id)
        ON DELETE SET NULL,
    name TEXT NOT NULL,
    description TEXT,
    difficulty trail_difficulty NOT NULL DEFAULT 'moderate',
    distance_meters DOUBLE PRECISION NOT NULL
        CONSTRAINT trails_distance_positive CHECK (distance_meters >= 0),
    elevation_gain_meters DOUBLE PRECISION NOT NULL
        CONSTRAINT trails_elevation_gain_positive CHECK (elevation_gain_meters >= 0),
    elevation_loss_meters DOUBLE PRECISION NOT NULL
        CONSTRAINT trails_elevation_loss_positive CHECK (elevation_loss_meters >= 0),
    estimated_duration_seconds INTEGER,
    path GEOMETRY(LineString, 4326) NOT NULL,
    is_official BOOLEAN NOT NULL DEFAULT false,
    is_published BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_trails_path
    ON trails USING GIST(path);

CREATE INDEX idx_trails_user
    ON trails(user_id);

CREATE TABLE waypoints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trail_id UUID NOT NULL
        REFERENCES trails(id)
        ON DELETE CASCADE,
    label TEXT NOT NULL,
    description TEXT,
    photo_url TEXT,
    location GEOMETRY(Point, 4326) NOT NULL,
    order_index INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT waypoints_trail_order_unique UNIQUE (trail_id, order_index)
);

CREATE INDEX idx_waypoints_location
    ON waypoints USING GIST(location);

ALTER TABLE trail_sessions
    ADD CONSTRAINT trail_sessions_trail_id_fkey
        FOREIGN KEY (trail_id) REFERENCES trails(id) ON DELETE SET NULL;

CREATE INDEX idx_trail_sessions_trail
    ON trail_sessions(trail_id);
