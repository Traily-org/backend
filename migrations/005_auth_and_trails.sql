-- 005_auth_and_trails.sql
-- Authentification (refresh tokens, rôles) et catalogue de randonnées avec PostGIS

-- Rôles et champs profil utilisateur
ALTER TABLE users ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'user';
ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_url TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS total_points INTEGER NOT NULL DEFAULT 0;

-- Tokens révocables pour le rafraîchissement d'authentification
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);

-- Difficulté des parcours
DO $$ 
BEGIN
    CREATE TYPE trail_difficulty AS ENUM ('easy', 'moderate', 'hard', 'expert');
EXCEPTION 
    WHEN duplicate_object THEN NULL; 
END $$;

-- Catalogue des randonnées
CREATE TABLE IF NOT EXISTS trails (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    description TEXT,
    difficulty trail_difficulty NOT NULL DEFAULT 'moderate',
    distance_m DOUBLE PRECISION NOT NULL,
    elevation_gain_m DOUBLE PRECISION NOT NULL,
    elevation_loss_m DOUBLE PRECISION NOT NULL,
    estimated_duration_s INTEGER NOT NULL,
    path GEOMETRY(LINESTRING, 4326) NOT NULL,
    is_official BOOLEAN NOT NULL DEFAULT false,
    is_published BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_trails_path ON trails USING GIST (path);

-- Points de passage du parcours
CREATE TABLE IF NOT EXISTS waypoints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trail_id UUID NOT NULL REFERENCES trails(id) ON DELETE CASCADE,
    label TEXT NOT NULL,
    description TEXT,
    order_index INTEGER NOT NULL,
    location GEOMETRY(POINT, 4326) NOT NULL,
    photo_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_waypoints_trail_order ON waypoints(trail_id, order_index);
CREATE INDEX IF NOT EXISTS idx_waypoints_location ON waypoints USING GIST (location);

-- Liaison entre la trace enregistrée et le trail
ALTER TABLE track ADD COLUMN IF NOT EXISTS trail_id UUID REFERENCES trails(id) ON DELETE SET NULL;
ALTER TABLE track ADD COLUMN IF NOT EXISTS is_private BOOLEAN NOT NULL DEFAULT false;
