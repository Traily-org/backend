SELECT id, email, password, name, username, display_name, avatar_url, role, total_points, created_at, updated_at
FROM users
WHERE id = $1;
