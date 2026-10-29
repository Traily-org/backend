package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/team"
)

const teamColumns = "id, name, description, avatar_url, created_by, created_at, updated_at"

type TeamRepository struct {
	pool *pgxpool.Pool
}

func NewTeamRepository(pool *pgxpool.Pool) *TeamRepository {
	return &TeamRepository{pool: pool}
}

func (r *TeamRepository) GetByID(ctx context.Context, id string) (team.Team, error) {
	row := r.pool.QueryRow(ctx, "SELECT "+teamColumns+" FROM team WHERE id = $1", id)
	e, err := scanTeam(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return team.Team{}, team.ErrNotFound
	}
	return e, err
}

func (r *TeamRepository) List(ctx context.Context) ([]team.Team, error) {
	rows, err := r.pool.Query(ctx, "SELECT "+teamColumns+" FROM team ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entities []team.Team
	for rows.Next() {
		e, err := scanTeam(rows)
		if err != nil {
			return nil, err
		}
		entities = append(entities, e)
	}
	return entities, rows.Err()
}

func (r *TeamRepository) Create(ctx context.Context, e team.Team) (team.Team, error) {
	row := r.pool.QueryRow(ctx,
		"INSERT INTO team (name, description, avatar_url, created_by) VALUES ($1, $2, $3, $4) RETURNING "+teamColumns,
		e.Name, e.Description, e.AvatarURL, e.CreatedBy)
	return scanTeam(row)
}

func (r *TeamRepository) Update(ctx context.Context, e team.Team) (team.Team, error) {
	row := r.pool.QueryRow(ctx,
		"UPDATE team SET name = $2, description = $3, avatar_url = $4, created_by = $5, updated_at = now() WHERE id = $1 RETURNING "+teamColumns,
		e.ID, e.Name, e.Description, e.AvatarURL, e.CreatedBy)
	updated, err := scanTeam(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return team.Team{}, team.ErrNotFound
	}
	return updated, err
}

func (r *TeamRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM team WHERE id = $1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return team.ErrNotFound
	}
	return nil
}

func scanTeam(row pgx.Row) (team.Team, error) {
	var e team.Team
	err := row.Scan(&e.ID, &e.Name, &e.Description, &e.AvatarURL, &e.CreatedBy, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return team.Team{}, err
	}
	return e, nil
}
