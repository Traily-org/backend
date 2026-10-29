package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/teammember"
)

const teammemberColumns = "id, team_id, user_id, role, joined_at"

type TeamMemberRepository struct {
	pool *pgxpool.Pool
}

func NewTeamMemberRepository(pool *pgxpool.Pool) *TeamMemberRepository {
	return &TeamMemberRepository{pool: pool}
}

func (r *TeamMemberRepository) GetByID(ctx context.Context, id string) (teammember.TeamMember, error) {
	row := r.pool.QueryRow(ctx, "SELECT "+teammemberColumns+" FROM team_member WHERE id = $1", id)
	e, err := scanTeamMember(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return teammember.TeamMember{}, teammember.ErrNotFound
	}
	return e, err
}

func (r *TeamMemberRepository) List(ctx context.Context) ([]teammember.TeamMember, error) {
	rows, err := r.pool.Query(ctx, "SELECT "+teammemberColumns+" FROM team_member ORDER BY joined_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entities []teammember.TeamMember
	for rows.Next() {
		e, err := scanTeamMember(rows)
		if err != nil {
			return nil, err
		}
		entities = append(entities, e)
	}
	return entities, rows.Err()
}

func (r *TeamMemberRepository) Create(ctx context.Context, e teammember.TeamMember) (teammember.TeamMember, error) {
	row := r.pool.QueryRow(ctx,
		"INSERT INTO team_member (team_id, user_id, role) VALUES ($1, $2, COALESCE(NULLIF($3, '')::team_role, 'member')) RETURNING "+teammemberColumns,
		e.TeamID, e.UserID, e.Role)
	return scanTeamMember(row)
}

func (r *TeamMemberRepository) Update(ctx context.Context, e teammember.TeamMember) (teammember.TeamMember, error) {
	row := r.pool.QueryRow(ctx,
		"UPDATE team_member SET team_id = $2, user_id = $3, role = COALESCE(NULLIF($4, '')::team_role, role) WHERE id = $1 RETURNING "+teammemberColumns,
		e.ID, e.TeamID, e.UserID, e.Role)
	updated, err := scanTeamMember(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return teammember.TeamMember{}, teammember.ErrNotFound
	}
	return updated, err
}

func (r *TeamMemberRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM team_member WHERE id = $1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return teammember.ErrNotFound
	}
	return nil
}

func scanTeamMember(row pgx.Row) (teammember.TeamMember, error) {
	var e teammember.TeamMember
	err := row.Scan(&e.ID, &e.TeamID, &e.UserID, &e.Role, &e.JoinedAt)
	if err != nil {
		return teammember.TeamMember{}, err
	}
	return e, nil
}
