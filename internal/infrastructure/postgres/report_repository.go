package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/report"
)

// uniquePendingReportIndex is the unique index allowing one pending report
// per reporter and target.
const uniquePendingReportIndex = "uq_reports_pending_per_reporter"

const (
	reportColumns = `id::text, reporter_id::text, target_type::text, target_id::text, reason, status::text, reviewed_by::text, reviewed_at, created_at`

	createReportQuery = `
		INSERT INTO reports (reporter_id, target_type, target_id, reason)
		VALUES ($1, $2::text::report_target, $3, $4)
		RETURNING ` + reportColumns

	reportFilter = `
		($1::text IS NULL OR status = $1::text::report_status)
		AND ($2::text IS NULL OR target_type = $2::text::report_target)`

	listReportsQuery = `
		SELECT ` + reportColumns + `
		FROM reports
		WHERE ` + reportFilter + `
		ORDER BY created_at DESC, id
		LIMIT $3 OFFSET $4`

	countReportsQuery = `SELECT count(*) FROM reports WHERE ` + reportFilter
)

// targetAuthorQueries returns, per target type, the author of the target
// $1 as seen by the reporter $2. No row means the target does not exist or is
// not visible to the reporter: only public photos and observations and
// published trails can be reported by someone else. Content shared with
// friends is not reportable until the friendship model exists.
var targetAuthorQueries = map[report.TargetType]string{
	report.TargetUser: `SELECT id::text FROM users WHERE id = $1`,
	report.TargetPhoto: `
		SELECT user_id::text FROM photos
		WHERE id = $1 AND (visibility = 'public' OR user_id = $2)`,
	report.TargetObservation: `
		SELECT user_id::text FROM observations
		WHERE id = $1 AND (visibility = 'public' OR user_id = $2)`,
	report.TargetTrail: `
		SELECT user_id::text FROM trails
		WHERE id = $1 AND (is_published OR user_id = $2)`,
}

// ReportRepository is the PostgreSQL implementation of report.Repository.
type ReportRepository struct {
	pool *pgxpool.Pool
}

// NewReportRepository returns a ReportRepository backed by the given pool.
func NewReportRepository(pool *pgxpool.Pool) *ReportRepository {
	return &ReportRepository{pool: pool}
}

// TargetAuthor returns the author of the target as seen by reporterID, or
// report.ErrTargetNotFound.
func (r *ReportRepository) TargetAuthor(ctx context.Context, kind report.TargetType, id, reporterID string) (*string, error) {
	query, ok := targetAuthorQueries[kind]
	if !ok {
		return nil, report.ErrInvalidTargetType
	}

	// The users query takes a single argument; the others take the reporter
	// too. Passing both is not allowed by PostgreSQL for the former.
	args := []any{id, reporterID}
	if kind == report.TargetUser {
		args = args[:1]
	}

	var author *string
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&author); err != nil {
		return nil, mapError(err, report.ErrTargetNotFound)
	}
	return author, nil
}

// Create stores r as a pending report.
func (r *ReportRepository) Create(ctx context.Context, rep report.Report) (report.Report, error) {
	created, err := scanReport(r.pool.QueryRow(ctx, createReportQuery,
		rep.ReporterID, string(rep.TargetType), rep.TargetID, rep.Reason))
	if err != nil {
		if isUniqueViolation(err, uniquePendingReportIndex) {
			return report.Report{}, report.ErrAlreadyReported
		}
		return report.Report{}, mapError(err, report.ErrTargetNotFound)
	}
	return created, nil
}

// List returns the reports matching f, newest first.
func (r *ReportRepository) List(ctx context.Context, f report.Filter) (report.Page, error) {
	var status, kind *string
	if f.Status != nil {
		s := string(*f.Status)
		status = &s
	}
	if f.TargetType != nil {
		k := string(*f.TargetType)
		kind = &k
	}

	page := report.Page{Limit: f.Limit, Offset: f.Offset}
	if err := r.pool.QueryRow(ctx, countReportsQuery, status, kind).Scan(&page.Total); err != nil {
		return report.Page{}, mapError(err, report.ErrTargetNotFound)
	}

	rows, err := r.pool.Query(ctx, listReportsQuery, status, kind, f.Limit, f.Offset)
	if err != nil {
		return report.Page{}, mapError(err, report.ErrTargetNotFound)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (report.Report, error) {
		return scanReport(row)
	})
	if err != nil {
		return report.Page{}, mapError(err, report.ErrTargetNotFound)
	}
	page.Items = items
	return page, nil
}

// scanReport reads a row selected with reportColumns.
func scanReport(row pgx.Row) (report.Report, error) {
	var rep report.Report
	var kind, status string
	err := row.Scan(&rep.ID, &rep.ReporterID, &kind, &rep.TargetID, &rep.Reason, &status, &rep.ReviewedBy, &rep.ReviewedAt, &rep.CreatedAt)
	rep.TargetType = report.TargetType(kind)
	rep.Status = report.Status(status)
	return rep, err
}
