package postgres

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/traily-org/server/internal/domain/plantspecies"
)

type PlantSpeciesRepository struct{ pool *pgxpool.Pool }

func NewPlantSpeciesRepository(pool *pgxpool.Pool) *PlantSpeciesRepository {
	return &PlantSpeciesRepository{pool: pool}
}
func (r *PlantSpeciesRepository) GetByID(ctx context.Context, id string) (plantspecies.PlantSpecies, error) {
	row := r.pool.QueryRow(ctx, "SELECT id, common_name, scientific_name, description FROM plant_species WHERE id = $1", id)
	var p plantspecies.PlantSpecies
	err := row.Scan(&p.ID, &p.CommonName, &p.ScientificName, &p.Description)
	if errors.Is(err, pgx.ErrNoRows) {
		return plantspecies.PlantSpecies{}, plantspecies.ErrNotFound
	}
	return p, err
}
func (r *PlantSpeciesRepository) List(ctx context.Context) ([]plantspecies.PlantSpecies, error) {
	rows, err := r.pool.Query(ctx, "SELECT id, common_name, scientific_name, description FROM plant_species ORDER BY common_name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var species []plantspecies.PlantSpecies
	for rows.Next() {
		var p plantspecies.PlantSpecies
		if err := rows.Scan(&p.ID, &p.CommonName, &p.ScientificName, &p.Description); err != nil {
			return nil, err
		}
		species = append(species, p)
	}
	return species, rows.Err()
}
func (r *PlantSpeciesRepository) Create(ctx context.Context, p plantspecies.PlantSpecies) (plantspecies.PlantSpecies, error) {
	row := r.pool.QueryRow(ctx, "INSERT INTO plant_species (common_name, scientific_name, description) VALUES ($1, $2, $3) RETURNING id, common_name, scientific_name, description", p.CommonName, p.ScientificName, p.Description)
	err := row.Scan(&p.ID, &p.CommonName, &p.ScientificName, &p.Description)
	return p, err
}
func (r *PlantSpeciesRepository) Update(ctx context.Context, p plantspecies.PlantSpecies) (plantspecies.PlantSpecies, error) {
	row := r.pool.QueryRow(ctx, "UPDATE plant_species SET common_name=$2, scientific_name=$3, description=$4 WHERE id=$1 RETURNING id, common_name, scientific_name, description", p.ID, p.CommonName, p.ScientificName, p.Description)
	err := row.Scan(&p.ID, &p.CommonName, &p.ScientificName, &p.Description)
	if errors.Is(err, pgx.ErrNoRows) {
		return plantspecies.PlantSpecies{}, plantspecies.ErrNotFound
	}
	return p, err
}
func (r *PlantSpeciesRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM plant_species WHERE id=$1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return plantspecies.ErrNotFound
	}
	return nil
}
