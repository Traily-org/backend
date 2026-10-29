CREATE TABLE user_points (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL
        REFERENCES users(id)
        ON DELETE CASCADE,
    points INTEGER NOT NULL,
    reason TEXT NOT NULL,
    reference_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_user_points_user
    ON user_points(user_id);

-- Keeps users.total_points equal to the sum of the user's user_points rows.
CREATE FUNCTION sync_user_total_points() RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP IN ('DELETE', 'UPDATE') THEN
        UPDATE users
        SET total_points = total_points - OLD.points
        WHERE id = OLD.user_id;
    END IF;

    IF TG_OP IN ('INSERT', 'UPDATE') THEN
        UPDATE users
        SET total_points = total_points + NEW.points
        WHERE id = NEW.user_id;
    END IF;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_user_points_sync_total
    AFTER INSERT OR DELETE OR UPDATE OF points, user_id ON user_points
    FOR EACH ROW
    EXECUTE FUNCTION sync_user_total_points();

CREATE TABLE badges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    icon_url TEXT,
    criteria JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE user_badges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL
        REFERENCES users(id)
        ON DELETE CASCADE,
    badge_id UUID NOT NULL
        REFERENCES badges(id)
        ON DELETE CASCADE,
    unlocked_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT user_badges_user_badge_unique UNIQUE (user_id, badge_id)
);

CREATE INDEX idx_user_badges_badge
    ON user_badges(badge_id);
