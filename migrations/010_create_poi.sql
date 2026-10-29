CREATE TABLE poi (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL
        REFERENCES "user"(id)
        ON DELETE CASCADE,
    name VARCHAR(150) NOT NULL,
    description TEXT,
    type poi_type NOT NULL DEFAULT 'other',
    location GEOMETRY(Point, 4326) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_poi_location
    ON poi USING GIST(location);

CREATE INDEX idx_poi_user
    ON poi(user_id);
