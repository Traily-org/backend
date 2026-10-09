INSERT INTO app_user (username, email, display_name, avatar_url)
VALUES ($1, $2, $3, $4)
RETURNING id, username, email, display_name, avatar_url, created_at, updated_at;
