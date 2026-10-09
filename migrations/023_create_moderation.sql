CREATE TABLE IF NOT EXISTS reports (
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
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_reports_status
    ON reports(status);

CREATE TABLE IF NOT EXISTS moderation_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    moderator_id UUID NOT NULL
        REFERENCES users(id)
        ON DELETE CASCADE,
    action mod_action NOT NULL,
    target_type report_target NOT NULL,
    target_id UUID NOT NULL,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_moderation_logs_target
    ON moderation_logs(target_type, target_id);
