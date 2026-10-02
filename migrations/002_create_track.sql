CREATE EXTENSION IF NOT EXISTS postgis;

CREATE TABLE IF NOT EXISTS track (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    route GEOMETRY(LINESTRING, 4326) NOT NULL,
    distance_m double precision NOT NULL,
    elevation_gain_m double precision NOT NULL,
    elevation_loss_m double precision NOT NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ended_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);