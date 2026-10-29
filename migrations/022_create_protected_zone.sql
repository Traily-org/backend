CREATE TABLE IF NOT EXISTS protected_zone (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    category TEXT NOT NULL,
    description TEXT,
    regulations TEXT,
    boundary GEOMETRY(MultiPolygon, 4326) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_protected_zone_boundary
    ON protected_zone USING GIST(boundary);
