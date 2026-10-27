package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/species"
)

const speciesColumns = "id, category, common_name, scientific_name, family, description, protection_status, rarity, ethical_advice, observation_months, thumbnail_url, created_at, updated_at"

type SpeciesRepository struct{ pool *pgxpool.Pool }

func NewSpeciesRepository(pool *pgxpool.Pool) *SpeciesRepository {
	return &SpeciesRepository{pool: pool}
}

func scanSpecies(row pgx.Row) (species.Species, error) {
	var s species.Species
	err := row.Scan(&s.ID, &s.Category, &s.CommonName, &s.ScientificName, &s.Family, &s.Description, &s.ProtectionStatus, &s.Rarity, &s.EthicalAdvice, &s.ObservationMonths, &s.ThumbnailURL, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return species.Species{}, species.ErrNotFound
	}
	return s, err
}

func (r *SpeciesRepository) GetByID(ctx context.Context, id string) (species.Species, error) {
	return scanSpecies(r.pool.QueryRow(ctx, "SELECT "+speciesColumns+" FROM species WHERE id = $1", id))
}

func (r *SpeciesRepository) List(ctx context.Context) ([]species.Species, error) {
	rows, err := r.pool.Query(ctx, "SELECT "+speciesColumns+" FROM species ORDER BY common_name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []species.Species
	for rows.Next() {
		s, err := scanSpecies(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

func (r *SpeciesRepository) Create(ctx context.Context, s species.Species) (species.Species, error) {
	return scanSpecies(r.pool.QueryRow(ctx, "INSERT INTO species (category, common_name, scientific_name, family, description, protection_status, rarity, ethical_advice, observation_months, thumbnail_url) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, COALESCE($9, '{}'::integer[]), $10) RETURNING "+speciesColumns,
		s.Category, s.CommonName, s.ScientificName, s.Family, s.Description, s.ProtectionStatus, s.Rarity, s.EthicalAdvice, s.ObservationMonths, s.ThumbnailURL))
}

func (r *SpeciesRepository) Update(ctx context.Context, s species.Species) (species.Species, error) {
	return scanSpecies(r.pool.QueryRow(ctx, "UPDATE species SET category=$2, common_name=$3, scientific_name=$4, family=$5, description=$6, protection_status=$7, rarity=$8, ethical_advice=$9, observation_months=COALESCE($10, '{}'::integer[]), thumbnail_url=$11, updated_at=NOW() WHERE id=$1 RETURNING "+speciesColumns,
		s.ID, s.Category, s.CommonName, s.ScientificName, s.Family, s.Description, s.ProtectionStatus, s.Rarity, s.EthicalAdvice, s.ObservationMonths, s.ThumbnailURL))
}

func (r *SpeciesRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM species WHERE id=$1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return species.ErrNotFound
	}
	return nil
}
