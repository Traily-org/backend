-- A user can belong to at most one team at a time.

-- Stop with an explicit message when existing memberships collide so they
-- can be resolved by hand before migrating.
DO $$
DECLARE
    duplicates TEXT;
BEGIN
    SELECT string_agg(user_id::text, ', ') INTO duplicates
    FROM (
        SELECT user_id FROM team_members GROUP BY user_id HAVING count(*) > 1
    ) d;
    IF duplicates IS NOT NULL THEN
        RAISE EXCEPTION 'users belonging to multiple teams must be resolved first: %', duplicates;
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'team_members_user_id_key'
          AND conrelid = 'public.team_members'::regclass
    ) THEN
        ALTER TABLE team_members
            ADD CONSTRAINT team_members_user_id_key UNIQUE (user_id);
    END IF;
END $$;

-- Covered by the unique index backing team_members_user_id_key.
DROP INDEX IF EXISTS idx_team_members_user;
