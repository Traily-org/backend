ALTER TABLE users
    ADD COLUMN username VARCHAR(50) UNIQUE,
    ADD COLUMN display_name VARCHAR(100),
    ADD COLUMN avatar_url TEXT,
    ADD COLUMN role TEXT NOT NULL DEFAULT 'user'
        CONSTRAINT users_role_valid CHECK (role IN ('user', 'moderator', 'admin')),
    ADD COLUMN total_points INTEGER NOT NULL DEFAULT 0;
