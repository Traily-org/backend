SELECT id, user_id, ST_AsText(route), distance_m, elevation_gain_m, elevation_loss_m, started_at, ended_at, created_at
FROM track
WHERE user_id = $1
ORDER BY created_at DESC;
