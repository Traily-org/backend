CREATE TABLE IF NOT EXISTS observation (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL
        REFERENCES "user"(id)
        ON DELETE CASCADE,
    species_id UUID
        REFERENCES species(id)
        ON DELETE SET NULL,
    photo_id UUID
        REFERENCES photo(id)
        ON DELETE SET NULL,
    trail_id UUID
        REFERENCES trail(id)
        ON DELETE SET NULL,
    track_id UUID
        REFERENCES track(id)
        ON DELETE SET NULL,
    location GEOMETRY(Point, 4326) NOT NULL,
    altitude DOUBLE PRECISION,
    confidence_score DOUBLE PRECISION,
    validation_status validation_status NOT NULL DEFAULT 'pending',
    visibility visibility_level NOT NULL DEFAULT 'public',
    observed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_observation_user_id
    ON observation(user_id);

CREATE INDEX idx_observation_species_id
    ON observation(species_id);

CREATE INDEX idx_observation_location
    ON observation USING GIST(location);

CREATE INDEX idx_observation_status
    ON observation(validation_status);

CREATE INDEX idx_observation_photo
    ON observation(photo_id);

CREATE INDEX idx_observation_trail
    ON observation(trail_id);

CREATE INDEX idx_observation_track
    ON observation(track_id);
