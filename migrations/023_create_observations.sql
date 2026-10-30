CREATE TABLE observations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL
        REFERENCES users(id)
        ON DELETE CASCADE,
    species_id UUID
        REFERENCES species(id)
        ON DELETE SET NULL,
    photo_id UUID
        REFERENCES photos(id)
        ON DELETE SET NULL,
    trail_id UUID
        REFERENCES trails(id)
        ON DELETE SET NULL,
    trail_session_id UUID
        REFERENCES trail_sessions(id)
        ON DELETE SET NULL,
    location GEOMETRY(Point, 4326) NOT NULL,
    altitude_meters DOUBLE PRECISION,
    confidence_score DOUBLE PRECISION
        CONSTRAINT observations_confidence_score_range
            CHECK (confidence_score >= 0 AND confidence_score <= 1),
    validation_status validation_status NOT NULL DEFAULT 'pending',
    visibility visibility_level NOT NULL DEFAULT 'public',
    observed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_observations_user
    ON observations(user_id);

CREATE INDEX idx_observations_species
    ON observations(species_id);

CREATE INDEX idx_observations_photo
    ON observations(photo_id);

CREATE INDEX idx_observations_trail
    ON observations(trail_id);

CREATE INDEX idx_observations_trail_session
    ON observations(trail_session_id);

CREATE INDEX idx_observations_validation_status
    ON observations(validation_status);

CREATE INDEX idx_observations_location
    ON observations USING GIST(location);
