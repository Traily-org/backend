CREATE TABLE IF NOT EXISTS species (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category species_category NOT NULL,
    common_name VARCHAR(150) NOT NULL,
    scientific_name VARCHAR(200),
    family TEXT,
    description TEXT,
    protection_status protection_level NOT NULL DEFAULT 'none',
    rarity rarity_level NOT NULL DEFAULT 'common',
    ethical_advice TEXT,
    observation_months INTEGER[] DEFAULT '{}',
    thumbnail_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_species_category
    ON species(category);
