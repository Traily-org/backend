//go:build integration

// Package testutil fournit l'infrastructure partagée des tests d'intégration :
// un container PostGIS unique par binaire de test, migré une seule fois, et des
// helpers pour obtenir une base propre et créer des fixtures.
package testutil

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/traily-org/server/migrations"
)

// PostGISImage est l'image Docker utilisée pour les tests d'intégration ; elle
// doit rester alignée sur celle du docker-compose.
const PostGISImage = "postgis/postgis:16-3.4-alpine"

const (
	dbName     = "traily"
	dbUser     = "traily"
	dbPassword = "traily"
)

// Tables exclues du nettoyage entre deux tests.
var preservedTables = map[string]bool{
	"schema_migrations": true,
	"spatial_ref_sys":   true,
}

var (
	setupOnce sync.Once
	setupErr  error
	baseDSN   string
)

// setup démarre le container et applique les migrations sur la base principale.
// Le container n'est pas arrêté explicitement : Ryuk (testcontainers) le
// supprime à la fin du binaire de test.
func setup() {
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx, PostGISImage,
		tcpostgres.WithDatabase(dbName),
		tcpostgres.WithUsername(dbUser),
		tcpostgres.WithPassword(dbPassword),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		setupErr = fmt.Errorf("start postgis container: %w", err)
		return
	}

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		setupErr = fmt.Errorf("get connection string: %w", err)
		return
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		setupErr = fmt.Errorf("connect to database: %w", err)
		return
	}
	defer pool.Close()

	if err := migrations.Run(ctx, pool); err != nil {
		setupErr = fmt.Errorf("run migrations: %w", err)
		return
	}

	baseDSN = dsn
}

// DSN retourne la chaîne de connexion de la base principale (déjà migrée),
// en démarrant le container partagé au premier appel.
func DSN(t *testing.T) string {
	t.Helper()

	setupOnce.Do(setup)
	if setupErr != nil {
		t.Fatalf("testutil: %v", setupErr)
	}
	return baseDSN
}

// NewPool retourne un pool connecté à la base principale migrée, après avoir
// vidé toutes les tables du schéma public (hors schema_migrations et
// spatial_ref_sys). Le pool est fermé automatiquement en fin de test.
//
// Le TRUNCATE porte sur toute la base partagée : les tests qui l'utilisent ne
// doivent pas appeler t.Parallel().
func NewPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	pool := connect(t, DSN(t))
	truncateAll(t, pool)
	return pool
}

// NewEmptyDatabase crée une base vierge (sans aucune migration) dans le
// container partagé et retourne un pool connecté dessus. La base est supprimée
// en fin de test.
func NewEmptyDatabase(t *testing.T) *pgxpool.Pool {
	t.Helper()

	ctx := context.Background()
	dsn := DSN(t)

	admin := connect(t, dsn)

	name := fmt.Sprintf("test_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize()); err != nil {
		t.Fatalf("testutil: create database %s: %v", name, err)
	}

	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("testutil: parse dsn: %v", err)
	}
	u.Path = "/" + name

	pool, err := pgxpool.New(ctx, u.String())
	if err != nil {
		t.Fatalf("testutil: connect to database %s: %v", name, err)
	}

	// Enregistré avant la fermeture du pool admin (LIFO) : le pool de test est
	// fermé, puis la base supprimée, puis le pool admin fermé.
	t.Cleanup(func() {
		pool.Close()
		if _, err := admin.Exec(context.Background(), "DROP DATABASE IF EXISTS "+pgx.Identifier{name}.Sanitize()+" WITH (FORCE)"); err != nil {
			t.Logf("testutil: drop database %s: %v", name, err)
		}
	})

	return pool
}

// CreateUser insère un utilisateur avec un email unique et retourne son ID.
func CreateUser(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()

	var id string
	err := pool.QueryRow(context.Background(), `
		INSERT INTO users (email, password, name)
		VALUES ('user-' || gen_random_uuid() || '@example.com', 'password-hash', 'Test User')
		RETURNING id::text`).Scan(&id)
	if err != nil {
		t.Fatalf("testutil: create user: %v", err)
	}
	return id
}

func connect(t *testing.T, dsn string) *pgxpool.Pool {
	t.Helper()

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("testutil: connect to database: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func truncateAll(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	ctx := context.Background()

	rows, err := pool.Query(ctx, `SELECT tablename FROM pg_tables WHERE schemaname = 'public'`)
	if err != nil {
		t.Fatalf("testutil: list tables: %v", err)
	}
	names, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatalf("testutil: scan tables: %v", err)
	}

	tables := make([]string, 0, len(names))
	for _, name := range names {
		if preservedTables[name] {
			continue
		}
		tables = append(tables, pgx.Identifier{"public", name}.Sanitize())
	}
	if len(tables) == 0 {
		return
	}

	if _, err := pool.Exec(ctx, "TRUNCATE TABLE "+strings.Join(tables, ", ")+" RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("testutil: truncate tables: %v", err)
	}
}
