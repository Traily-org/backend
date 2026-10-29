package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/species"
)

const speciesColumns = "id, category, common_name, scientific_name, family, description, protection_status, rarity, ethical_advice, observation_months, thumbnail_url, created_at, updated_at"

type SpeciesRepository struct {
	pool *pgxpool.Pool
}

func NewSpeciesRepository(pool *pgxpool.Pool) *SpeciesRepository {
	return &SpeciesRepository{pool: pool}
}

func (r *SpeciesRepository) GetByID(ctx context.Context, id string) (species.Species, error) {
	row := r.pool.QueryRow(ctx, "SELECT "+speciesColumns+" FROM species WHERE id = $1", id)
	e, err := scanSpecies(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return species.Species{}, species.ErrNotFound
	}
	return e, err
}

func (r *SpeciesRepository) List(ctx context.Context) ([]species.Species, error) {
	rows, err := r.pool.Query(ctx, "SELECT "+speciesColumns+" FROM species ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entities []species.Species
	for rows.Next() {
		e, err := scanSpecies(rows)
		if err != nil {
			return nil, err
		}
		entities = append(entities, e)
	}
	return entities, rows.Err()
}

func (r *SpeciesRepository) Create(ctx context.Context, e species.Species) (species.Species, error) {
	row := r.pool.QueryRow(ctx,
		"INSERT INTO species (category, common_name, scientific_name, family, description, protection_status, rarity, ethical_advice, observation_months, thumbnail_url) VALUES ($1, $2, $3, $4, $5, COALESCE(NULLIF($6, '')::protection_level, 'none'), COALESCE(NULLIF($7, '')::rarity_level, 'common'), $8, COALESCE($9::int[], '{}'), $10) RETURNING "+speciesColumns,
		e.Category, e.CommonName, e.ScientificName, e.Family, e.Description, e.ProtectionStatus, e.Rarity, e.EthicalAdvice, e.ObservationMonths, e.ThumbnailURL)
	return scanSpecies(row)
}

func (r *SpeciesRepository) Update(ctx context.Context, e species.Species) (species.Species, error) {
	row := r.pool.QueryRow(ctx,
		"UPDATE species SET category = COALESCE(NULLIF($2, '')::species_category, category), common_name = $3, scientific_name = $4, family = $5, description = $6, protection_status = COALESCE(NULLIF($7, '')::protection_level, protection_status), rarity = COALESCE(NULLIF($8, '')::rarity_level, rarity), ethical_advice = $9, observation_months = COALESCE($10::int[], '{}'), thumbnail_url = $11, updated_at = now() WHERE id = $1 RETURNING "+speciesColumns,
		e.ID, e.Category, e.CommonName, e.ScientificName, e.Family, e.Description, e.ProtectionStatus, e.Rarity, e.EthicalAdvice, e.ObservationMonths, e.ThumbnailURL)
	updated, err := scanSpecies(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return species.Species{}, species.ErrNotFound
	}
	return updated, err
}

func (r *SpeciesRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM species WHERE id = $1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return species.ErrNotFound
	}
	return nil
}

func scanSpecies(row pgx.Row) (species.Species, error) {
	var e species.Species
	err := row.Scan(&e.ID, &e.Category, &e.CommonName, &e.ScientificName, &e.Family, &e.Description, &e.ProtectionStatus, &e.Rarity, &e.EthicalAdvice, &e.ObservationMonths, &e.ThumbnailURL, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return species.Species{}, err
	}
	return e, nil
}
