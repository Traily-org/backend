CREATE TABLE track (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL
        REFERENCES "user"(id)
        ON DELETE CASCADE,
    trail_id UUID
        REFERENCES trail(id)
        ON DELETE SET NULL,
    route GEOMETRY(LineString, 4326) NOT NULL,
    distance_m DOUBLE PRECISION NOT NULL,
    elevation_gain_m DOUBLE PRECISION NOT NULL,
    elevation_loss_m DOUBLE PRECISION NOT NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_private BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT track_distance_positive
        CHECK (distance_m >= 0),
    CONSTRAINT track_elevation_gain_positive
        CHECK (elevation_gain_m >= 0),
    CONSTRAINT track_elevation_loss_positive
        CHECK (elevation_loss_m >= 0),
    CONSTRAINT track_dates_valid
        CHECK (ended_at >= started_at)
);

CREATE INDEX idx_track_user
    ON track(user_id);

CREATE INDEX idx_track_trail
    ON track(trail_id);

CREATE INDEX idx_track_route
    ON track USING GIST(route);
