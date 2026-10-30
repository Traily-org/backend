-- Emails and usernames are compared case-insensitively: forbid two accounts
-- that only differ by case, and store emails lowercased. The email index also
-- serves the lookup by lower(email) used at login.

-- Stop with an explicit message instead of a constraint error when existing
-- accounts collide; they must be merged by hand before migrating.
DO $$
DECLARE
    duplicates TEXT;
BEGIN
    SELECT string_agg(key, ', ') INTO duplicates
    FROM (
        SELECT lower(trim(email)) AS key FROM users GROUP BY 1 HAVING count(*) > 1
    ) d;
    IF duplicates IS NOT NULL THEN
        RAISE EXCEPTION 'users differing only by email case must be merged first: %', duplicates;
    END IF;

    SELECT string_agg(key, ', ') INTO duplicates
    FROM (
        SELECT lower(username) AS key FROM users WHERE username IS NOT NULL GROUP BY 1 HAVING count(*) > 1
    ) d;
    IF duplicates IS NOT NULL THEN
        RAISE EXCEPTION 'users differing only by username case must be renamed first: %', duplicates;
    END IF;
END $$;

UPDATE users SET email = lower(trim(email)) WHERE email <> lower(trim(email));

CREATE UNIQUE INDEX users_email_lower_key
    ON users (lower(email));

CREATE UNIQUE INDEX users_username_lower_key
    ON users (lower(username));
