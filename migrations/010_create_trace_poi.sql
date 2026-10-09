CREATE TABLE trace_poi (
    trace_id UUID NOT NULL
        REFERENCES trace(id)
        ON DELETE CASCADE,
    poi_id UUID NOT NULL
        REFERENCES poi(id)
        ON DELETE CASCADE,
    order_index INTEGER,
    distance_from_start_m DOUBLE PRECISION,
    PRIMARY KEY (trace_id, poi_id)
);

CREATE INDEX idx_trace_poi_poi
    ON trace_poi(poi_id);
