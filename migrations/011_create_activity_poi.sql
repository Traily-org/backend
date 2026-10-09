CREATE TABLE activity_poi (
    activity_id UUID NOT NULL
        REFERENCES activity(id)
        ON DELETE CASCADE,
    poi_id UUID NOT NULL
        REFERENCES poi(id)
        ON DELETE CASCADE,
    visited_at TIMESTAMPTZ,
    PRIMARY KEY (activity_id, poi_id)
);

CREATE INDEX idx_activity_poi_poi
    ON activity_poi(poi_id);
