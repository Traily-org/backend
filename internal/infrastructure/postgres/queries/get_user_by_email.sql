SELECT id, email, password, name, username, display_name, avatar_url, role, total_points, created_at, updated_at
FROM users
WHERE lower(email) = lower($1)
ORDER BY created_at, id
LIMIT 1;
