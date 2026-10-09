CREATE TABLE plant_species (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    common_name VARCHAR(150) NOT NULL,
    scientific_name VARCHAR(200),
    description TEXT
);

CREATE INDEX idx_plant_species_common_name
    ON plant_species(common_name);
