-- 006_species_and_observations.sql
-- Référentiel unifié flore / faune / insectes, photos géolocalisées et observations naturalistes

DO $$ BEGIN
    CREATE TYPE species_category AS ENUM ('flora', 'fauna', 'insect');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE protection_level AS ENUM ('none', 'concern', 'vulnerable', 'endangered', 'strictly_protected');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE rarity_level AS ENUM ('common', 'uncommon', 'rare', 'legendary');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE visibility_level AS ENUM ('private', 'friends', 'public');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE nsfw_status AS ENUM ('pending', 'safe', 'flagged', 'rejected');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE validation_status AS ENUM ('pending', 'validated', 'rejected');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

-- Référentiel unifié flore, faune et insectes (Herbier et Bestiaire)
CREATE TABLE IF NOT EXISTS species (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category species_category NOT NULL,
    common_name TEXT NOT NULL,
    scientific_name TEXT,
    family TEXT,
    description TEXT,
    protection_status protection_level NOT NULL DEFAULT 'none',
    rarity rarity_level NOT NULL DEFAULT 'common',
    ethical_advice TEXT,
    observation_months INTEGER[] DEFAULT '{}',
    thumbnail_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_species_category ON species(category);

-- Métadonnées des photos stockées sur S3
CREATE TABLE IF NOT EXISTS photos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    storage_key TEXT NOT NULL,
    url TEXT NOT NULL,
    location GEOMETRY(POINT, 4326),
    visibility visibility_level NOT NULL DEFAULT 'public',
    nsfw_status nsfw_status NOT NULL DEFAULT 'pending',
    taken_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_photos_location ON photos USING GIST (location);

-- Observations terrain (Herbier / Bestiaire avec IA + validation)
CREATE TABLE IF NOT EXISTS observations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    species_id UUID REFERENCES species(id) ON DELETE SET NULL,
    photo_id UUID REFERENCES photos(id) ON DELETE SET NULL,
    trail_id UUID REFERENCES trails(id) ON DELETE SET NULL,
    track_id UUID REFERENCES track(id) ON DELETE SET NULL,
    location GEOMETRY(POINT, 4326) NOT NULL,
    altitude DOUBLE PRECISION,
    confidence_score DOUBLE PRECISION,
    validation_status validation_status NOT NULL DEFAULT 'pending',
    visibility visibility_level NOT NULL DEFAULT 'public',
    observed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_observations_user_id ON observations(user_id);
CREATE INDEX IF NOT EXISTS idx_observations_species_id ON observations(species_id);
CREATE INDEX IF NOT EXISTS idx_observations_location ON observations USING GIST (location);
CREATE INDEX IF NOT EXISTS idx_observations_status ON observations(validation_status);
