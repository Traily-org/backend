UPDATE track
SET user_id = $2, route = ST_GeomFromText($3, 4326), distance_m = $4, elevation_gain_m = $5, elevation_loss_m = $6, started_at = $7, ended_at = $8
WHERE id = $1
RETURNING id, user_id, ST_AsText(route), distance_m, elevation_gain_m, elevation_loss_m, started_at, ended_at, created_at;
