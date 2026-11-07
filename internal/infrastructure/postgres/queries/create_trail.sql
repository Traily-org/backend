INSERT INTO trails (user_id, name, description, difficulty, distance_meters, elevation_gain_meters, elevation_loss_meters, estimated_duration_seconds, path, is_published)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, ST_GeomFromText($9, 4326), $10)
RETURNING id, user_id, name, description, difficulty, distance_meters, elevation_gain_meters, elevation_loss_meters, estimated_duration_seconds, ST_AsGeoJSON(path), is_official, is_published, created_at, updated_at;
