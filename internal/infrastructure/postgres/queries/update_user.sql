UPDATE users
SET email = $2,
    password = COALESCE(NULLIF($3, ''), password),
    name = $4,
    username = $5,
    display_name = $6,
    avatar_url = $7,
    updated_at = now()
WHERE id = $1
RETURNING id, email, password, name, username, display_name, avatar_url, role, total_points, created_at, updated_at;
