SELECT id, user_id, route, distance_m, elevation_gain_m, elevation_loss_m, started_at, ended_at, created_at
FROM track
WHERE id = $1;
