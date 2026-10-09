DO $$ 
BEGIN
    CREATE TYPE poi_type AS ENUM (
        'viewpoint',
        'waterfall',
        'lake',
        'river',
        'mountain',
        'cave',
        'refuge',
        'monument',
        'historical',
        'restaurant',
        'parking',
        'camping',
        'other'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS poi (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    type poi_type NOT NULL,
    description TEXT,
    location GEOMETRY(POINT, 4326) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);