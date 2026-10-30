-- Merge app_user into users, re-point every legacy user_id foreign key to
-- users (keeping the original ON DELETE rules), then drop app_user.

ALTER TABLE track DROP CONSTRAINT track_user_id_fkey;
ALTER TABLE trace DROP CONSTRAINT trace_user_id_fkey;
ALTER TABLE activity DROP CONSTRAINT activity_user_id_fkey;
ALTER TABLE publication DROP CONSTRAINT publication_user_id_fkey;
ALTER TABLE poi DROP CONSTRAINT poi_user_id_fkey;
ALTER TABLE herbier DROP CONSTRAINT herbier_user_id_fkey;

-- app_user rows whose email already exists in users are attached to that
-- account: their legacy rows are re-pointed to the existing users id.
CREATE TEMP TABLE app_user_mapping ON COMMIT DROP AS
SELECT a.id AS app_user_id, u.id AS user_id
FROM app_user a
JOIN users u ON lower(u.email) = lower(a.email);

UPDATE track t SET user_id = m.user_id FROM app_user_mapping m WHERE t.user_id = m.app_user_id;
UPDATE trace t SET user_id = m.user_id FROM app_user_mapping m WHERE t.user_id = m.app_user_id;
UPDATE activity t SET user_id = m.user_id FROM app_user_mapping m WHERE t.user_id = m.app_user_id;
UPDATE publication t SET user_id = m.user_id FROM app_user_mapping m WHERE t.user_id = m.app_user_id;
UPDATE poi t SET user_id = m.user_id FROM app_user_mapping m WHERE t.user_id = m.app_user_id;
UPDATE herbier t SET user_id = m.user_id FROM app_user_mapping m WHERE t.user_id = m.app_user_id;

-- The other app_user rows become users with the same id. They get an unusable
-- password ('!' is never a valid bcrypt hash) and must reset it to log in.
INSERT INTO users (id, email, password, name, username, display_name, avatar_url, created_at, updated_at)
SELECT a.id, a.email, '!', COALESCE(a.display_name, a.username), a.username, a.display_name, a.avatar_url, a.created_at, a.updated_at
FROM app_user a
WHERE NOT EXISTS (SELECT 1 FROM app_user_mapping m WHERE m.app_user_id = a.id)
  AND NOT EXISTS (SELECT 1 FROM users u WHERE u.username = a.username);

-- Any row still pointing to an unknown user (username clash) makes the
-- migration fail on the constraints below instead of silently losing data.
ALTER TABLE track
    ADD CONSTRAINT track_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE trace
    ADD CONSTRAINT trace_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE activity
    ADD CONSTRAINT activity_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE publication
    ADD CONSTRAINT publication_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE poi
    ADD CONSTRAINT poi_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE herbier
    ADD CONSTRAINT herbier_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

DROP TABLE app_user;
