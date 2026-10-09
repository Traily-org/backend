INSERT INTO track (user_id, route, distance_m, elevation_gain_m, elevation_loss_m, started_at, ended_at)
VALUES ($1, ST_GeomFromText($2, 4326), $3, $4, $5, $6, $7)
RETURNING id, user_id, ST_AsText(route), distance_m, elevation_gain_m, elevation_loss_m, started_at, ended_at, created_at;
