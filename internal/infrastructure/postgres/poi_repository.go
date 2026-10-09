package postgres
import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/traily-org/server/internal/domain/poi"
)
type POIRepository struct{ pool *pgxpool.Pool }
func NewPOIRepository(pool *pgxpool.Pool) *POIRepository { return &POIRepository{pool: pool} }
func (r *POIRepository) GetByID(ctx context.Context, id string) (poi.POI, error) {
	row := r.pool.QueryRow(ctx, "SELECT id, user_id, name, description, type, ST_AsText(location), created_at, updated_at FROM poi WHERE id = $1", id)
	var p poi.POI; var loc string; err := row.Scan(&p.ID, &p.UserID, &p.Name, &p.Description, &p.Type, &loc, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) { return poi.POI{}, poi.ErrNotFound }
	if err == nil && loc != "" { p.Location = []byte(loc) }
	return p, err
}
func (r *POIRepository) List(ctx context.Context) ([]poi.POI, error) {
	rows, err := r.pool.Query(ctx, "SELECT id, user_id, name, description, type, ST_AsText(location), created_at, updated_at FROM poi ORDER BY created_at DESC")
	if err != nil { return nil, err }; defer rows.Close()
	var pois []poi.POI
	for rows.Next() {
		var p poi.POI; var loc string
		if err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.Description, &p.Type, &loc, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		if loc != "" { p.Location = []byte(loc) }
		pois = append(pois, p)
	}
	return pois, rows.Err()
}
func (r *POIRepository) Create(ctx context.Context, p poi.POI) (poi.POI, error) {
	var loc string; if p.Location != nil { loc = string(p.Location) }
	row := r.pool.QueryRow(ctx, "INSERT INTO poi (user_id, name, description, type, location) VALUES ($1, $2, $3, $4, ST_GeomFromText($5, 4326)) RETURNING id, user_id, name, description, type, ST_AsText(location), created_at, updated_at", p.UserID, p.Name, p.Description, p.Type, loc)
	var nloc string; err := row.Scan(&p.ID, &p.UserID, &p.Name, &p.Description, &p.Type, &nloc, &p.CreatedAt, &p.UpdatedAt)
	if err == nil && nloc != "" { p.Location = []byte(nloc) }
	return p, err
}
func (r *POIRepository) Update(ctx context.Context, p poi.POI) (poi.POI, error) {
	var loc string; if p.Location != nil { loc = string(p.Location) }
	row := r.pool.QueryRow(ctx, "UPDATE poi SET user_id=$2, name=$3, description=$4, type=$5, location=ST_GeomFromText($6, 4326), updated_at=NOW() WHERE id=$1 RETURNING id, user_id, name, description, type, ST_AsText(location), created_at, updated_at", p.ID, p.UserID, p.Name, p.Description, p.Type, loc)
	var nloc string; err := row.Scan(&p.ID, &p.UserID, &p.Name, &p.Description, &p.Type, &nloc, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) { return poi.POI{}, poi.ErrNotFound }
	if err == nil && nloc != "" { p.Location = []byte(nloc) }
	return p, err
}
func (r *POIRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM poi WHERE id=$1", id)
	if err != nil { return err }
	if tag.RowsAffected() == 0 { return poi.ErrNotFound }
	return nil
}
