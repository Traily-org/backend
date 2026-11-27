-- A user cannot flood the moderation queue: at most one pending report per
-- reporter and target. Once a moderator has reviewed or dismissed it, the
-- same user may report the target again.
CREATE UNIQUE INDEX uq_reports_pending_per_reporter
    ON reports(reporter_id, target_type, target_id)
    WHERE status = 'pending';

-- The dashboard lists reports newest first, optionally by status.
CREATE INDEX idx_reports_created_at
    ON reports(created_at DESC);
