UPDATE "user"
SET email = $2, password = $3, name = $4, username = $5, display_name = $6, avatar_url = $7, updated_at = now()
WHERE id = $1
RETURNING id, email, password, name, username, display_name, role, avatar_url, total_points, created_at, updated_at;
