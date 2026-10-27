-- Plantes et champignons uniquement (décision du 27/10/2026) : la faune et les insectes sont retirés,
-- et l'herbier devient une vue calculée sur observations + species.
-- Après le DELETE, il ne reste que des espèces 'flora', toutes converties en 'plant'.
DROP TABLE IF EXISTS herbier_entry;
DROP TABLE IF EXISTS herbier;
DROP TABLE IF EXISTS plant_species;

DELETE FROM species WHERE category IN ('fauna', 'insect');

ALTER TYPE species_category RENAME TO species_category_old;

CREATE TYPE species_category AS ENUM (
    'plant',
    'fungus'
);

ALTER TABLE species
    ALTER COLUMN category TYPE species_category
    USING 'plant'::species_category;

DROP TYPE species_category_old;
