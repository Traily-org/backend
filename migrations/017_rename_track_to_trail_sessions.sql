ALTER TYPE trace_difficulty RENAME TO trail_difficulty;

ALTER TABLE track RENAME TO trail_sessions;
ALTER TABLE trail_sessions RENAME COLUMN route TO recorded_path;

ALTER TABLE trail_sessions
    ADD COLUMN trail_id UUID,
    ADD COLUMN is_private BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

ALTER INDEX idx_track_user RENAME TO idx_trail_sessions_user;
ALTER INDEX idx_track_route RENAME TO idx_trail_sessions_route;

-- Keep constraint names consistent with the new table name.
ALTER TABLE trail_sessions RENAME CONSTRAINT track_pkey TO trail_sessions_pkey;
ALTER TABLE trail_sessions RENAME CONSTRAINT track_user_id_fkey TO trail_sessions_user_id_fkey;
ALTER TABLE trail_sessions RENAME CONSTRAINT track_distance_positive TO trail_sessions_distance_positive;
ALTER TABLE trail_sessions RENAME CONSTRAINT track_elevation_gain_positive TO trail_sessions_elevation_gain_positive;
ALTER TABLE trail_sessions RENAME CONSTRAINT track_elevation_loss_positive TO trail_sessions_elevation_loss_positive;
ALTER TABLE trail_sessions RENAME CONSTRAINT track_dates_valid TO trail_sessions_dates_valid;
