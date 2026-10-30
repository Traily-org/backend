CREATE TABLE species (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scientific_name TEXT NOT NULL UNIQUE,
    common_name TEXT NOT NULL,
    category species_category NOT NULL,
    family TEXT,
    description TEXT,
    ethical_advice TEXT,
    protection_level protection_level NOT NULL DEFAULT 'none',
    rarity rarity_level NOT NULL DEFAULT 'common',
    observation_months SMALLINT[]
        CONSTRAINT species_observation_months_valid
            CHECK (observation_months <@ ARRAY[1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]::SMALLINT[]),
    distribution GEOMETRY(MultiPolygon, 4326),
    image_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_species_category
    ON species(category);

CREATE INDEX idx_species_distribution
    ON species USING GIST(distribution);
