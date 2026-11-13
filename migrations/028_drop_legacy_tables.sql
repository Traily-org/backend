-- Drop the first-draft schema (migrations 005 to 014), replaced by trails,
-- waypoints, trail_sessions, species and observations. No code reads these
-- tables. trace, activity and publication also held ON DELETE RESTRICT keys
-- to trail_sessions, which blocked deleting a session.
-- Children first, so no CASCADE is needed and an unexpected dependency fails
-- the migration instead of being dropped silently.

DROP TABLE herbier_entry;
DROP TABLE herbier;
DROP TABLE plant_species;
DROP TABLE activity_poi;
DROP TABLE trace_poi;
DROP TABLE publication;
DROP TABLE activity;
DROP TABLE trace;
DROP TABLE poi;

-- Only used by poi. trail_difficulty stays: trails.difficulty uses it.
DROP TYPE poi_type;
