CREATE TABLE herbier_entry (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    herbier_id UUID NOT NULL
        REFERENCES herbier(id)
        ON DELETE CASCADE,
    activity_id UUID
        REFERENCES activity(id)
        ON DELETE SET NULL,
    poi_id UUID
        REFERENCES poi(id)
        ON DELETE SET NULL,
    plant_species_id UUID NOT NULL
        REFERENCES plant_species(id)
        ON DELETE RESTRICT,
    photo_url TEXT,
    notes TEXT,
    location GEOMETRY(Point, 4326),
    observed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_herbier_entry_herbier
    ON herbier_entry(herbier_id);

CREATE INDEX idx_herbier_entry_activity
    ON herbier_entry(activity_id);

CREATE INDEX idx_herbier_entry_species
    ON herbier_entry(plant_species_id);

CREATE INDEX idx_herbier_entry_location
    ON herbier_entry USING GIST(location);
