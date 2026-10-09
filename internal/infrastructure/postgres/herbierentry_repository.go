package postgres

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/traily-org/server/internal/domain/herbierentry"
)

type HerbierEntryRepository struct{ pool *pgxpool.Pool }

func NewHerbierEntryRepository(pool *pgxpool.Pool) *HerbierEntryRepository {
	return &HerbierEntryRepository{pool: pool}
}
func (r *HerbierEntryRepository) GetByID(ctx context.Context, id string) (herbierentry.HerbierEntry, error) {
	row := r.pool.QueryRow(ctx, "SELECT id, herbier_id, activity_id, poi_id, plant_species_id, photo_url, notes, ST_AsText(location), observed_at, created_at FROM herbier_entry WHERE id = $1", id)
	var h herbierentry.HerbierEntry
	var loc string
	err := row.Scan(&h.ID, &h.HerbierID, &h.ActivityID, &h.POIID, &h.PlantSpeciesID, &h.PhotoURL, &h.Notes, &loc, &h.ObservedAt, &h.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return herbierentry.HerbierEntry{}, herbierentry.ErrNotFound
	}
	if err == nil && loc != "" {
		h.Location = []byte(loc)
	}
	return h, err
}
func (r *HerbierEntryRepository) List(ctx context.Context) ([]herbierentry.HerbierEntry, error) {
	rows, err := r.pool.Query(ctx, "SELECT id, herbier_id, activity_id, poi_id, plant_species_id, photo_url, notes, ST_AsText(location), observed_at, created_at FROM herbier_entry ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []herbierentry.HerbierEntry
	for rows.Next() {
		var h herbierentry.HerbierEntry
		var loc string
		if err := rows.Scan(&h.ID, &h.HerbierID, &h.ActivityID, &h.POIID, &h.PlantSpeciesID, &h.PhotoURL, &h.Notes, &loc, &h.ObservedAt, &h.CreatedAt); err != nil {
			return nil, err
		}
		if loc != "" {
			h.Location = []byte(loc)
		}
		entries = append(entries, h)
	}
	return entries, rows.Err()
}
func (r *HerbierEntryRepository) Create(ctx context.Context, h herbierentry.HerbierEntry) (herbierentry.HerbierEntry, error) {
	var loc string
	if h.Location != nil {
		loc = string(h.Location)
	}
	row := r.pool.QueryRow(ctx, "INSERT INTO herbier_entry (herbier_id, activity_id, poi_id, plant_species_id, photo_url, notes, location, observed_at) VALUES ($1, $2, $3, $4, $5, $6, CASE WHEN $7::text IS NOT NULL THEN ST_GeomFromText($7, 4326) END, $8) RETURNING id, herbier_id, activity_id, poi_id, plant_species_id, photo_url, notes, ST_AsText(location), observed_at, created_at", h.HerbierID, h.ActivityID, h.POIID, h.PlantSpeciesID, h.PhotoURL, h.Notes, loc, h.ObservedAt)
	var nloc string
	err := row.Scan(&h.ID, &h.HerbierID, &h.ActivityID, &h.POIID, &h.PlantSpeciesID, &h.PhotoURL, &h.Notes, &nloc, &h.ObservedAt, &h.CreatedAt)
	if err == nil && nloc != "" {
		h.Location = []byte(nloc)
	}
	return h, err
}
func (r *HerbierEntryRepository) Update(ctx context.Context, h herbierentry.HerbierEntry) (herbierentry.HerbierEntry, error) {
	var loc string
	if h.Location != nil {
		loc = string(h.Location)
	}
	row := r.pool.QueryRow(ctx, "UPDATE herbier_entry SET herbier_id=$2, activity_id=$3, poi_id=$4, plant_species_id=$5, photo_url=$6, notes=$7, location=CASE WHEN $8::text IS NOT NULL THEN ST_GeomFromText($8, 4326) END, observed_at=$9 WHERE id=$1 RETURNING id, herbier_id, activity_id, poi_id, plant_species_id, photo_url, notes, ST_AsText(location), observed_at, created_at", h.ID, h.HerbierID, h.ActivityID, h.POIID, h.PlantSpeciesID, h.PhotoURL, h.Notes, loc, h.ObservedAt)
	var nloc string
	err := row.Scan(&h.ID, &h.HerbierID, &h.ActivityID, &h.POIID, &h.PlantSpeciesID, &h.PhotoURL, &h.Notes, &nloc, &h.ObservedAt, &h.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return herbierentry.HerbierEntry{}, herbierentry.ErrNotFound
	}
	if err == nil && nloc != "" {
		h.Location = []byte(nloc)
	}
	return h, err
}
func (r *HerbierEntryRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM herbier_entry WHERE id=$1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return herbierentry.ErrNotFound
	}
	return nil
}
