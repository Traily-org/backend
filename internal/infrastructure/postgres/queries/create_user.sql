INSERT INTO "user" (email, password, name, username, display_name, avatar_url)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, email, password, name, username, display_name, role, avatar_url, total_points, created_at, updated_at;
