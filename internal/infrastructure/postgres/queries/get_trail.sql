SELECT id, user_id, name, description, difficulty, distance_meters, elevation_gain_meters, elevation_loss_meters, estimated_duration_seconds, ST_AsGeoJSON(path), is_official, is_published, created_at, updated_at
FROM trails
WHERE id = $1;
