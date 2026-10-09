CREATE TYPE trace_difficulty AS ENUM (
    'easy',
    'moderate',
    'hard',
    'expert'
);

CREATE TYPE poi_type AS ENUM (
    'viewpoint',
    'waterfall',
    'lake',
    'river',
    'mountain',
    'cave',
    'refuge',
    'monument',
    'historical',
    'restaurant',
    'parking',
    'camping',
    'other'
);

CREATE TYPE species_category AS ENUM (
    'flora',
    'fauna',
    'insect'
);

CREATE TYPE protection_level AS ENUM (
    'none',
    'concern',
    'vulnerable',
    'endangered',
    'strictly_protected'
);

CREATE TYPE rarity_level AS ENUM (
    'common',
    'uncommon',
    'rare',
    'legendary'
);

CREATE TYPE visibility_level AS ENUM (
    'private',
    'friends',
    'public'
);

CREATE TYPE nsfw_status AS ENUM (
    'pending',
    'safe',
    'flagged',
    'rejected'
);

CREATE TYPE validation_status AS ENUM (
    'pending',
    'validated',
    'rejected'
);

CREATE TYPE team_role AS ENUM (
    'admin',
    'member'
);

CREATE TYPE report_target AS ENUM (
    'photo',
    'observation',
    'trail',
    'user'
);

CREATE TYPE report_status AS ENUM (
    'pending',
    'reviewed',
    'dismissed'
);

CREATE TYPE mod_action AS ENUM (
    'approve',
    'reject',
    'delete',
    'ban'
);
