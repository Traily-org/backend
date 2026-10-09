package postgres
import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/traily-org/server/internal/domain/herbier"
)
type HerbierRepository struct{ pool *pgxpool.Pool }
func NewHerbierRepository(pool *pgxpool.Pool) *HerbierRepository { return &HerbierRepository{pool: pool} }
func (r *HerbierRepository) GetByID(ctx context.Context, id string) (herbier.Herbier, error) {
	row := r.pool.QueryRow(ctx, "SELECT id, user_id, name, created_at, updated_at FROM herbier WHERE id = $1", id)
	var h herbier.Herbier; err := row.Scan(&h.ID, &h.UserID, &h.Name, &h.CreatedAt, &h.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) { return herbier.Herbier{}, herbier.ErrNotFound }
	return h, err
}
func (r *HerbierRepository) List(ctx context.Context) ([]herbier.Herbier, error) {
	rows, err := r.pool.Query(ctx, "SELECT id, user_id, name, created_at, updated_at FROM herbier ORDER BY created_at DESC")
	if err != nil { return nil, err }; defer rows.Close()
	var herbiers []herbier.Herbier
	for rows.Next() {
		var h herbier.Herbier
		if err := rows.Scan(&h.ID, &h.UserID, &h.Name, &h.CreatedAt, &h.UpdatedAt); err != nil {
			return nil, err
		}
		herbiers = append(herbiers, h)
	}
	return herbiers, rows.Err()
}
func (r *HerbierRepository) Create(ctx context.Context, h herbier.Herbier) (herbier.Herbier, error) {
	row := r.pool.QueryRow(ctx, "INSERT INTO herbier (user_id, name) VALUES ($1, $2) RETURNING id, user_id, name, created_at, updated_at", h.UserID, h.Name)
	err := row.Scan(&h.ID, &h.UserID, &h.Name, &h.CreatedAt, &h.UpdatedAt)
	return h, err
}
func (r *HerbierRepository) Update(ctx context.Context, h herbier.Herbier) (herbier.Herbier, error) {
	row := r.pool.QueryRow(ctx, "UPDATE herbier SET user_id=$2, name=$3, updated_at=NOW() WHERE id=$1 RETURNING id, user_id, name, created_at, updated_at", h.ID, h.UserID, h.Name)
	err := row.Scan(&h.ID, &h.UserID, &h.Name, &h.CreatedAt, &h.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) { return herbier.Herbier{}, herbier.ErrNotFound }
	return h, err
}
func (r *HerbierRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM herbier WHERE id=$1", id)
	if err != nil { return err }
	if tag.RowsAffected() == 0 { return herbier.ErrNotFound }
	return nil
}
