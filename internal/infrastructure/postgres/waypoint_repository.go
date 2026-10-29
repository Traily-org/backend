package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/waypoint"
)

const waypointColumns = "id, trail_id, label, description, order_index, ST_AsText(location), photo_url, created_at"

type WaypointRepository struct {
	pool *pgxpool.Pool
}

func NewWaypointRepository(pool *pgxpool.Pool) *WaypointRepository {
	return &WaypointRepository{pool: pool}
}

func (r *WaypointRepository) GetByID(ctx context.Context, id string) (waypoint.Waypoint, error) {
	row := r.pool.QueryRow(ctx, "SELECT "+waypointColumns+" FROM waypoint WHERE id = $1", id)
	e, err := scanWaypoint(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return waypoint.Waypoint{}, waypoint.ErrNotFound
	}
	return e, err
}

func (r *WaypointRepository) List(ctx context.Context) ([]waypoint.Waypoint, error) {
	rows, err := r.pool.Query(ctx, "SELECT "+waypointColumns+" FROM waypoint ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entities []waypoint.Waypoint
	for rows.Next() {
		e, err := scanWaypoint(rows)
		if err != nil {
			return nil, err
		}
		entities = append(entities, e)
	}
	return entities, rows.Err()
}

func (r *WaypointRepository) Create(ctx context.Context, e waypoint.Waypoint) (waypoint.Waypoint, error) {
	row := r.pool.QueryRow(ctx,
		"INSERT INTO waypoint (trail_id, label, description, order_index, location, photo_url) VALUES ($1, $2, $3, $4, ST_GeomFromText($5, 4326), $6) RETURNING "+waypointColumns,
		e.TrailID, e.Label, e.Description, e.OrderIndex, wktArg(e.Location), e.PhotoURL)
	return scanWaypoint(row)
}

func (r *WaypointRepository) Update(ctx context.Context, e waypoint.Waypoint) (waypoint.Waypoint, error) {
	row := r.pool.QueryRow(ctx,
		"UPDATE waypoint SET trail_id = $2, label = $3, description = $4, order_index = $5, location = ST_GeomFromText($6, 4326), photo_url = $7 WHERE id = $1 RETURNING "+waypointColumns,
		e.ID, e.TrailID, e.Label, e.Description, e.OrderIndex, wktArg(e.Location), e.PhotoURL)
	updated, err := scanWaypoint(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return waypoint.Waypoint{}, waypoint.ErrNotFound
	}
	return updated, err
}

func (r *WaypointRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM waypoint WHERE id = $1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return waypoint.ErrNotFound
	}
	return nil
}

func scanWaypoint(row pgx.Row) (waypoint.Waypoint, error) {
	var e waypoint.Waypoint
	var geomLocation *string
	err := row.Scan(&e.ID, &e.TrailID, &e.Label, &e.Description, &e.OrderIndex, &geomLocation, &e.PhotoURL, &e.CreatedAt)
	if err != nil {
		return waypoint.Waypoint{}, err
	}
	e.Location = wktBytes(geomLocation)
	return e, nil
}
