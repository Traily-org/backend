package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/report"
)

const reportColumns = "id, reporter_id, target_type, target_id, reason, status, reviewed_by, reviewed_at, created_at"

type ReportRepository struct {
	pool *pgxpool.Pool
}

func NewReportRepository(pool *pgxpool.Pool) *ReportRepository {
	return &ReportRepository{pool: pool}
}

func (r *ReportRepository) GetByID(ctx context.Context, id string) (report.Report, error) {
	row := r.pool.QueryRow(ctx, "SELECT "+reportColumns+" FROM report WHERE id = $1", id)
	e, err := scanReport(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return report.Report{}, report.ErrNotFound
	}
	return e, err
}

func (r *ReportRepository) List(ctx context.Context) ([]report.Report, error) {
	rows, err := r.pool.Query(ctx, "SELECT "+reportColumns+" FROM report ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entities []report.Report
	for rows.Next() {
		e, err := scanReport(rows)
		if err != nil {
			return nil, err
		}
		entities = append(entities, e)
	}
	return entities, rows.Err()
}

func (r *ReportRepository) Create(ctx context.Context, e report.Report) (report.Report, error) {
	row := r.pool.QueryRow(ctx,
		"INSERT INTO report (reporter_id, target_type, target_id, reason, status, reviewed_by, reviewed_at) VALUES ($1, $2, $3, $4, COALESCE(NULLIF($5, '')::report_status, 'pending'), $6, $7) RETURNING "+reportColumns,
		e.ReporterID, e.TargetType, e.TargetID, e.Reason, e.Status, e.ReviewedBy, e.ReviewedAt)
	return scanReport(row)
}

func (r *ReportRepository) Update(ctx context.Context, e report.Report) (report.Report, error) {
	row := r.pool.QueryRow(ctx,
		"UPDATE report SET reporter_id = $2, target_type = COALESCE(NULLIF($3, '')::report_target, target_type), target_id = $4, reason = $5, status = COALESCE(NULLIF($6, '')::report_status, status), reviewed_by = $7, reviewed_at = $8 WHERE id = $1 RETURNING "+reportColumns,
		e.ID, e.ReporterID, e.TargetType, e.TargetID, e.Reason, e.Status, e.ReviewedBy, e.ReviewedAt)
	updated, err := scanReport(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return report.Report{}, report.ErrNotFound
	}
	return updated, err
}

func (r *ReportRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM report WHERE id = $1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return report.ErrNotFound
	}
	return nil
}

func scanReport(row pgx.Row) (report.Report, error) {
	var e report.Report
	err := row.Scan(&e.ID, &e.ReporterID, &e.TargetType, &e.TargetID, &e.Reason, &e.Status, &e.ReviewedBy, &e.ReviewedAt, &e.CreatedAt)
	if err != nil {
		return report.Report{}, err
	}
	return e, nil
}
