-- route accepts WKT or GeoJSON; a missing value keeps the stored one.
UPDATE track
SET user_id = $2,
    trail_id = $3,
    route = CASE WHEN left($4, 1) = '{' THEN ST_SetSRID(ST_GeomFromGeoJSON($4), 4326) ELSE ST_GeomFromText($4, 4326) END,
    distance_m = COALESCE($5, distance_m),
    elevation_gain_m = COALESCE($6, elevation_gain_m),
    elevation_loss_m = COALESCE($7, elevation_loss_m),
    started_at = COALESCE($8, started_at),
    ended_at = COALESCE($9, ended_at),
    is_private = $10,
    updated_at = now()
WHERE id = $1
RETURNING id, user_id, trail_id, ST_AsText(route), distance_m, elevation_gain_m, elevation_loss_m, started_at, ended_at, is_private, created_at, updated_at;
