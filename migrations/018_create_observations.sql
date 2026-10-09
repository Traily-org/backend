CREATE TABLE IF NOT EXISTS observations (
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
    trace_id UUID
        REFERENCES trace(id)
        ON DELETE SET NULL,
    activity_id UUID
        REFERENCES activity(id)
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

CREATE INDEX idx_observations_user_id
    ON observations(user_id);

CREATE INDEX idx_observations_species_id
    ON observations(species_id);

CREATE INDEX idx_observations_location
    ON observations USING GIST(location);

CREATE INDEX idx_observations_status
    ON observations(validation_status);
