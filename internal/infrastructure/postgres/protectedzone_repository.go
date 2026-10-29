package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/protectedzone"
)

const protectedzoneColumns = "id, name, category, description, regulations, ST_AsText(boundary), created_at"

type ProtectedZoneRepository struct {
	pool *pgxpool.Pool
}

func NewProtectedZoneRepository(pool *pgxpool.Pool) *ProtectedZoneRepository {
	return &ProtectedZoneRepository{pool: pool}
}

func (r *ProtectedZoneRepository) GetByID(ctx context.Context, id string) (protectedzone.ProtectedZone, error) {
	row := r.pool.QueryRow(ctx, "SELECT "+protectedzoneColumns+" FROM protected_zone WHERE id = $1", id)
	e, err := scanProtectedZone(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return protectedzone.ProtectedZone{}, protectedzone.ErrNotFound
	}
	return e, err
}

func (r *ProtectedZoneRepository) List(ctx context.Context) ([]protectedzone.ProtectedZone, error) {
	rows, err := r.pool.Query(ctx, "SELECT "+protectedzoneColumns+" FROM protected_zone ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entities []protectedzone.ProtectedZone
	for rows.Next() {
		e, err := scanProtectedZone(rows)
		if err != nil {
			return nil, err
		}
		entities = append(entities, e)
	}
	return entities, rows.Err()
}

func (r *ProtectedZoneRepository) Create(ctx context.Context, e protectedzone.ProtectedZone) (protectedzone.ProtectedZone, error) {
	row := r.pool.QueryRow(ctx,
		"INSERT INTO protected_zone (name, category, description, regulations, boundary) VALUES ($1, $2, $3, $4, ST_GeomFromText($5, 4326)) RETURNING "+protectedzoneColumns,
		e.Name, e.Category, e.Description, e.Regulations, wktArg(e.Boundary))
	return scanProtectedZone(row)
}

func (r *ProtectedZoneRepository) Update(ctx context.Context, e protectedzone.ProtectedZone) (protectedzone.ProtectedZone, error) {
	row := r.pool.QueryRow(ctx,
		"UPDATE protected_zone SET name = $2, category = $3, description = $4, regulations = $5, boundary = ST_GeomFromText($6, 4326) WHERE id = $1 RETURNING "+protectedzoneColumns,
		e.ID, e.Name, e.Category, e.Description, e.Regulations, wktArg(e.Boundary))
	updated, err := scanProtectedZone(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return protectedzone.ProtectedZone{}, protectedzone.ErrNotFound
	}
	return updated, err
}

func (r *ProtectedZoneRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM protected_zone WHERE id = $1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return protectedzone.ErrNotFound
	}
	return nil
}

func scanProtectedZone(row pgx.Row) (protectedzone.ProtectedZone, error) {
	var e protectedzone.ProtectedZone
	var geomBoundary *string
	err := row.Scan(&e.ID, &e.Name, &e.Category, &e.Description, &e.Regulations, &geomBoundary, &e.CreatedAt)
	if err != nil {
		return protectedzone.ProtectedZone{}, err
	}
	e.Boundary = wktBytes(geomBoundary)
	return e, nil
}
