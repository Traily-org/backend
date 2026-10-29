-- route accepts WKT or GeoJSON; distance and elevations default to 0 and dates to now().
INSERT INTO track (user_id, trail_id, route, distance_m, elevation_gain_m, elevation_loss_m, started_at, ended_at, is_private)
VALUES (
    $1,
    $2,
    CASE WHEN left($3, 1) = '{' THEN ST_SetSRID(ST_GeomFromGeoJSON($3), 4326) ELSE ST_GeomFromText($3, 4326) END,
    COALESCE($4::double precision, 0),
    COALESCE($5::double precision, 0),
    COALESCE($6::double precision, 0),
    COALESCE($7::timestamptz, now()),
    COALESCE($8::timestamptz, $7::timestamptz, now()),
    $9
)
RETURNING id, user_id, trail_id, ST_AsText(route), distance_m, elevation_gain_m, elevation_loss_m, started_at, ended_at, is_private, created_at, updated_at;
