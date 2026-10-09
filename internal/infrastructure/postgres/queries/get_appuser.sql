SELECT id, username, email, display_name, avatar_url, created_at, updated_at
FROM app_user
WHERE id = $1;
