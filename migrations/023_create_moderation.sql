CREATE TABLE IF NOT EXISTS report (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reporter_id UUID NOT NULL
        REFERENCES "user"(id)
        ON DELETE CASCADE,
    target_type report_target NOT NULL,
    target_id UUID NOT NULL,
    reason TEXT NOT NULL,
    status report_status NOT NULL DEFAULT 'pending',
    reviewed_by UUID
        REFERENCES "user"(id)
        ON DELETE SET NULL,
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_report_status
    ON report(status);

CREATE TABLE IF NOT EXISTS moderation_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    moderator_id UUID NOT NULL
        REFERENCES "user"(id)
        ON DELETE CASCADE,
    action mod_action NOT NULL,
    target_type report_target NOT NULL,
    target_id UUID NOT NULL,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_moderation_log_target
    ON moderation_log(target_type, target_id);

CREATE INDEX idx_report_reporter
    ON report(reporter_id);

CREATE INDEX idx_report_reviewed_by
    ON report(reviewed_by);

CREATE INDEX idx_report_target
    ON report(target_type, target_id);

CREATE INDEX idx_moderation_log_moderator
    ON moderation_log(moderator_id);
