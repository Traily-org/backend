CREATE TABLE track (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL
        REFERENCES users(id)
        ON DELETE CASCADE,
    route GEOMETRY(LineString, 4326) NOT NULL,
    distance_m DOUBLE PRECISION,
    elevation_gain_m DOUBLE PRECISION,
    elevation_loss_m DOUBLE PRECISION,
    started_at TIMESTAMPTZ,
    ended_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT track_distance_positive
        CHECK (distance_m IS NULL OR distance_m >= 0),
    CONSTRAINT track_elevation_gain_positive
        CHECK (elevation_gain_m IS NULL OR elevation_gain_m >= 0),
    CONSTRAINT track_elevation_loss_positive
        CHECK (elevation_loss_m IS NULL OR elevation_loss_m >= 0),
    CONSTRAINT track_dates_valid
        CHECK (
            ended_at IS NULL
            OR started_at IS NULL
            OR ended_at >= started_at
        )
);

CREATE INDEX idx_track_user
    ON track(user_id);

CREATE INDEX idx_track_route
    ON track USING GIST(route);
