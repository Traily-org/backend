SELECT id, email, password, name, username, display_name, role, avatar_url, total_points, created_at, updated_at
FROM "user"
WHERE id = $1;
