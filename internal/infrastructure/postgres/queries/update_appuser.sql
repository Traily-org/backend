UPDATE app_user
SET username = $2, email = $3, display_name = $4, avatar_url = $5, updated_at = NOW()
WHERE id = $1
RETURNING id, username, email, display_name, avatar_url, created_at, updated_at;
