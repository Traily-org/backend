CREATE TABLE protected_zones (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT,
    protection_level protection_level NOT NULL,
    area GEOMETRY(MultiPolygon, 4326) NOT NULL,
    source TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_protected_zones_area
    ON protected_zones USING GIST(area);

CREATE TABLE reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reporter_id UUID NOT NULL
        REFERENCES users(id)
        ON DELETE CASCADE,
    target_type report_target NOT NULL,
    target_id UUID NOT NULL,
    reason TEXT NOT NULL,
    status report_status NOT NULL DEFAULT 'pending',
    reviewed_by UUID
        REFERENCES users(id)
        ON DELETE SET NULL,
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_reports_reporter
    ON reports(reporter_id);

CREATE INDEX idx_reports_reviewed_by
    ON reports(reviewed_by);

CREATE INDEX idx_reports_target
    ON reports(target_type, target_id);

CREATE INDEX idx_reports_status
    ON reports(status);

CREATE TABLE moderation_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    moderator_id UUID NOT NULL
        REFERENCES users(id)
        ON DELETE CASCADE,
    action mod_action NOT NULL,
    target_type report_target NOT NULL,
    target_id UUID NOT NULL,
    report_id UUID
        REFERENCES reports(id)
        ON DELETE SET NULL,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_moderation_logs_moderator
    ON moderation_logs(moderator_id);

CREATE INDEX idx_moderation_logs_target
    ON moderation_logs(target_type, target_id);
