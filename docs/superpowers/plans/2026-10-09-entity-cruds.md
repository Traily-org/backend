# Entity CRUDs Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement complete CRUD operations for all 9 domain entities (AppUser, Track, Trace, Activity, Publication, POI, Herbier, PlantSpecies, HerbierEntry) following the hexagonal architecture pattern.

**Architecture:** Three-layer architecture with clear separation: domain layer (entities, interfaces, services), infrastructure layer (PostgreSQL repository implementations), and adapter layer (HTTP handlers with DTOs). Each entity follows the same pattern established by the existing User entity.

**Tech Stack:** Go 1.21+, PostgreSQL with PostGIS extension, Gin web framework, pgx driver, JSON DTOs with binding validation.

**Spec:** Based on migrations in `migrations/` directory containing schema definitions for all entities.

## Global Constraints

- Each CRUD operation must validate context (ctx) for cancellation
- All timestamps use TIMESTAMPTZ
- Geospatial types use GEOMETRY(Point/LineString, 4326)
- UUIDs are auto-generated with `gen_random_uuid()`
- Foreign key cascading follows migration definitions
- All errors map to domain-specific error types (ErrNotFound, ErrConflict, etc.)
- HTTP handlers use Gin context with status codes: 200 (OK), 201 (Created), 204 (No Content), 400 (Bad Request), 404 (Not Found), 409 (Conflict), 500 (Server Error)
- All datetime fields in responses are ISO 8601 format
- Repository methods must return either the entity or an error, never both

## Review Focus

1. **Geospatial data handling (PostGIS)** - Track.Route and POI.Location/HerbierEntry.Location are geometry types that need proper scanning/encoding to/from database
2. **Relationship integrity** - Foreign key constraints (e.g., Activity references Track which references AppUser) must be enforced consistently in create/update operations
3. **Optional fields** - Several entities have nullable fields (Description, TraceID, ActivityID) that must be handled correctly in both scanning and JSON marshaling
4. **Unique constraints** - Track.user_id+id, Activity.user_id+id are functionally unique; Herbier has unique user_id constraint
5. **Temporal logic** - Activity and Track have date validation constraints (ended_at >= started_at) that should be enforced at service layer before database

---

## File Structure

### Domain Layer
- `internal/domain/appuser/` - AppUser entity, repository interface, service, errors
- `internal/domain/track/` - Track entity, repository interface, service, errors
- `internal/domain/trace/` - Trace entity, repository interface, service, errors
- `internal/domain/activity/` - Activity entity, repository interface, service, errors
- `internal/domain/publication/` - Publication entity, repository interface, service, errors
- `internal/domain/poi/` - POI entity, repository interface, service, errors
- `internal/domain/herbier/` - Herbier entity, repository interface, service, errors
- `internal/domain/plantspecies/` - PlantSpecies entity, repository interface, service, errors
- `internal/domain/herbierentry/` - HerbierEntry entity, repository interface, service, errors

### Infrastructure Layer
- `internal/infrastructure/postgres/appuser_repository.go`
- `internal/infrastructure/postgres/track_repository.go`
- `internal/infrastructure/postgres/trace_repository.go`
- `internal/infrastructure/postgres/activity_repository.go`
- `internal/infrastructure/postgres/publication_repository.go`
- `internal/infrastructure/postgres/poi_repository.go`
- `internal/infrastructure/postgres/herbier_repository.go`
- `internal/infrastructure/postgres/plantspecies_repository.go`
- `internal/infrastructure/postgres/herbierentry_repository.go`
- `internal/infrastructure/postgres/queries/` - SQL query files for all entities

### Adapter Layer (HTTP)
- `internal/adapters/http/appuser_handler.go`
- `internal/adapters/http/track_handler.go`
- `internal/adapters/http/trace_handler.go`
- `internal/adapters/http/activity_handler.go`
- `internal/adapters/http/publication_handler.go`
- `internal/adapters/http/poi_handler.go`
- `internal/adapters/http/herbier_handler.go`
- `internal/adapters/http/plantspecies_handler.go`
- `internal/adapters/http/herbierentry_handler.go`
- `internal/adapters/http/dto.go` - All DTOs (consolidated or split as needed)
- `internal/adapters/http/server.go` - Modified to register all new handlers and routes

---

## Task 1: AppUser CRUD

**Files:**
- Create: `internal/domain/appuser/appuser.go`
- Create: `internal/domain/appuser/repository.go`
- Create: `internal/domain/appuser/service.go`
- Create: `internal/domain/appuser/errors.go`
- Create: `internal/infrastructure/postgres/appuser_repository.go`
- Create: `internal/infrastructure/postgres/queries/appuser_queries.go`
- Create: `internal/infrastructure/postgres/queries/get_appuser.sql`
- Create: `internal/infrastructure/postgres/queries/list_appusers.sql`
- Create: `internal/infrastructure/postgres/queries/create_appuser.sql`
- Create: `internal/infrastructure/postgres/queries/update_appuser.sql`
- Create: `internal/infrastructure/postgres/queries/delete_appuser.sql`
- Create: `internal/adapters/http/appuser_handler.go`
- Modify: `internal/adapters/http/dto.go` - Add AppUser DTOs
- Modify: `internal/adapters/http/server.go` - Add appuser handler and routes

**Interfaces:**
- Consumes: User service pattern (already exists)
- Produces: AppUserService interface with Get, Create, Update, Delete methods; HTTP routes: GET/POST /api/v1/app-users, GET/PUT/DELETE /api/v1/app-users/:id

- [ ] **Step 1: Create AppUser domain entity**

```go
// internal/domain/appuser/appuser.go
package appuser

import "time"

type AppUser struct {
	ID          string
	Username    string
	Email       string
	DisplayName *string // nullable
	AvatarURL   *string // nullable
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
```

- [ ] **Step 2: Create AppUser repository interface**

```go
// internal/domain/appuser/repository.go
package appuser

import "context"

type Repository interface {
	GetByID(ctx context.Context, id string) (AppUser, error)
	GetByUsername(ctx context.Context, username string) (AppUser, error)
	List(ctx context.Context) ([]AppUser, error)
	Create(ctx context.Context, u AppUser) (AppUser, error)
	Update(ctx context.Context, u AppUser) (AppUser, error)
	Delete(ctx context.Context, id string) error
}
```

- [ ] **Step 3: Create AppUser errors**

```go
// internal/domain/appuser/errors.go
package appuser

import "errors"

var (
	ErrNotFound        = errors.New("appuser not found")
	ErrEmailTaken      = errors.New("email already in use")
	ErrUsernameTaken   = errors.New("username already in use")
	ErrInvalidUsername = errors.New("username must be 3-50 characters")
	ErrInvalidEmail    = errors.New("invalid email format")
)
```

- [ ] **Step 4: Create AppUser service**

```go
// internal/domain/appuser/service.go
package appuser

import "context"

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Get(ctx context.Context, id string) (AppUser, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) GetByUsername(ctx context.Context, username string) (AppUser, error) {
	return s.repo.GetByUsername(ctx, username)
}

func (s *Service) List(ctx context.Context) ([]AppUser, error) {
	return s.repo.List(ctx)
}

func (s *Service) Create(ctx context.Context, u AppUser) (AppUser, error) {
	return s.repo.Create(ctx, u)
}

func (s *Service) Update(ctx context.Context, u AppUser) (AppUser, error) {
	return s.repo.Update(ctx, u)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
```

- [ ] **Step 5: Create SQL query files**

```sql
-- internal/infrastructure/postgres/queries/get_appuser.sql
SELECT id, username, email, display_name, avatar_url, created_at, updated_at
FROM app_user
WHERE id = $1;

-- internal/infrastructure/postgres/queries/list_appusers.sql
SELECT id, username, email, display_name, avatar_url, created_at, updated_at
FROM app_user
ORDER BY created_at DESC;

-- internal/infrastructure/postgres/queries/create_appuser.sql
INSERT INTO app_user (username, email, display_name, avatar_url)
VALUES ($1, $2, $3, $4)
RETURNING id, username, email, display_name, avatar_url, created_at, updated_at;

-- internal/infrastructure/postgres/queries/update_appuser.sql
UPDATE app_user
SET username = $2, email = $3, display_name = $4, avatar_url = $5, updated_at = NOW()
WHERE id = $1
RETURNING id, username, email, display_name, avatar_url, created_at, updated_at;

-- internal/infrastructure/postgres/queries/delete_appuser.sql
DELETE FROM app_user WHERE id = $1;
```

- [ ] **Step 6: Create queries loader**

```go
// internal/infrastructure/postgres/queries/appuser_queries.go
package queries

import _ "embed"

//go:embed get_appuser.sql
var GetAppUserQuery string

//go:embed list_appusers.sql
var ListAppUsersQuery string

//go:embed create_appuser.sql
var CreateAppUserQuery string

//go:embed update_appuser.sql
var UpdateAppUserQuery string

//go:embed delete_appuser.sql
var DeleteAppUserQuery string
```

- [ ] **Step 7: Create AppUser repository implementation**

```go
// internal/infrastructure/postgres/appuser_repository.go
package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/appuser"
	"github.com/traily-org/server/internal/infrastructure/postgres/queries"
)

type AppUserRepository struct {
	pool *pgxpool.Pool
}

func NewAppUserRepository(pool *pgxpool.Pool) *AppUserRepository {
	return &AppUserRepository{pool: pool}
}

func (r *AppUserRepository) GetByID(ctx context.Context, id string) (appuser.AppUser, error) {
	row := r.pool.QueryRow(ctx, queries.GetAppUserQuery, id)
	return scanAppUser(row)
}

func (r *AppUserRepository) GetByUsername(ctx context.Context, username string) (appuser.AppUser, error) {
	row := r.pool.QueryRow(ctx, "SELECT id, username, email, display_name, avatar_url, created_at, updated_at FROM app_user WHERE username = $1", username)
	u, err := scanAppUser(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return appuser.AppUser{}, appuser.ErrNotFound
	}
	return u, err
}

func (r *AppUserRepository) List(ctx context.Context) ([]appuser.AppUser, error) {
	rows, err := r.pool.Query(ctx, queries.ListAppUsersQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []appuser.AppUser
	for rows.Next() {
		u, err := scanAppUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	return users, rows.Err()
}

func (r *AppUserRepository) Create(ctx context.Context, u appuser.AppUser) (appuser.AppUser, error) {
	row := r.pool.QueryRow(ctx, queries.CreateAppUserQuery, u.Username, u.Email, u.DisplayName, u.AvatarURL)
	created, err := scanAppUser(row)
	if isUniqueViolation(err) {
		// Could be username or email
		if err.Error() != "" {
			return appuser.AppUser{}, appuser.ErrUsernameTaken
		}
		return appuser.AppUser{}, appuser.ErrEmailTaken
	}
	return created, err
}

func (r *AppUserRepository) Update(ctx context.Context, u appuser.AppUser) (appuser.AppUser, error) {
	row := r.pool.QueryRow(ctx, queries.UpdateAppUserQuery, u.ID, u.Username, u.Email, u.DisplayName, u.AvatarURL)
	updated, err := scanAppUser(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return appuser.AppUser{}, appuser.ErrNotFound
	}
	if isUniqueViolation(err) {
		return appuser.AppUser{}, appuser.ErrUsernameTaken
	}
	return updated, err
}

func (r *AppUserRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, queries.DeleteAppUserQuery, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return appuser.ErrNotFound
	}
	return nil
}

func scanAppUser(row pgx.Row) (appuser.AppUser, error) {
	var u appuser.AppUser
	err := row.Scan(&u.ID, &u.Username, &u.Email, &u.DisplayName, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt)
	return u, err
}
```

- [ ] **Step 8: Add AppUser DTOs to dto.go**

```go
// Add to internal/adapters/http/dto.go

type appUserResponse struct {
	ID          string    `json:"id"`
	Username    string    `json:"username"`
	Email       string    `json:"email"`
	DisplayName *string   `json:"display_name,omitempty"`
	AvatarURL   *string   `json:"avatar_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func newAppUserResponse(u appuser.AppUser) appUserResponse {
	return appUserResponse{
		ID:          u.ID,
		Username:    u.Username,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		AvatarURL:   u.AvatarURL,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}

type createAppUserRequest struct {
	Username    string  `json:"username" binding:"required,min=3,max=50"`
	Email       string  `json:"email" binding:"required,email"`
	DisplayName *string `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
}

type updateAppUserRequest struct {
	Username    string  `json:"username" binding:"required,min=3,max=50"`
	Email       string  `json:"email" binding:"required,email"`
	DisplayName *string `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
}
```

- [ ] **Step 9: Create AppUser HTTP handler**

```go
// internal/adapters/http/appuser_handler.go
package http

import (
	"context"
	"errors"
	nethttp "net/http"

	"github.com/gin-gonic/gin"

	"github.com/traily-org/server/internal/domain/appuser"
)

type appUserService interface {
	Get(ctx context.Context, id string) (appuser.AppUser, error)
	GetByUsername(ctx context.Context, username string) (appuser.AppUser, error)
	List(ctx context.Context) ([]appuser.AppUser, error)
	Create(ctx context.Context, u appuser.AppUser) (appuser.AppUser, error)
	Update(ctx context.Context, u appuser.AppUser) (appuser.AppUser, error)
	Delete(ctx context.Context, id string) error
}

type AppUserHandler struct {
	service appUserService
}

func NewAppUserHandler(service appUserService) *AppUserHandler {
	return &AppUserHandler{service: service}
}

func (h *AppUserHandler) Get(ctx *gin.Context) {
	id := ctx.Param("id")
	u, err := h.service.Get(ctx.Request.Context(), id)
	if err != nil {
		writeAppUserError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusOK, newAppUserResponse(u))
}

func (h *AppUserHandler) List(ctx *gin.Context) {
	users, err := h.service.List(ctx.Request.Context())
	if err != nil {
		ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	responses := make([]appUserResponse, len(users))
	for i, u := range users {
		responses[i] = newAppUserResponse(u)
	}
	ctx.JSON(nethttp.StatusOK, responses)
}

func (h *AppUserHandler) Create(ctx *gin.Context) {
	var req createAppUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	u, err := h.service.Create(ctx.Request.Context(), appuser.AppUser{
		Username:    req.Username,
		Email:       req.Email,
		DisplayName: req.DisplayName,
		AvatarURL:   req.AvatarURL,
	})
	if err != nil {
		writeAppUserError(ctx, err)
		return
	}

	ctx.JSON(nethttp.StatusCreated, newAppUserResponse(u))
}

func (h *AppUserHandler) Update(ctx *gin.Context) {
	id := ctx.Param("id")

	var req updateAppUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	u, err := h.service.Update(ctx.Request.Context(), appuser.AppUser{
		ID:          id,
		Username:    req.Username,
		Email:       req.Email,
		DisplayName: req.DisplayName,
		AvatarURL:   req.AvatarURL,
	})
	if err != nil {
		writeAppUserError(ctx, err)
		return
	}

	ctx.JSON(nethttp.StatusOK, newAppUserResponse(u))
}

func (h *AppUserHandler) Delete(ctx *gin.Context) {
	id := ctx.Param("id")
	if err := h.service.Delete(ctx.Request.Context(), id); err != nil {
		writeAppUserError(ctx, err)
		return
	}
	ctx.Status(nethttp.StatusNoContent)
}

func writeAppUserError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, appuser.ErrNotFound):
		ctx.JSON(nethttp.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, appuser.ErrEmailTaken), errors.Is(err, appuser.ErrUsernameTaken):
		ctx.JSON(nethttp.StatusConflict, gin.H{"error": err.Error()})
	default:
		ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
```

- [ ] **Step 10: Update server.go to register AppUser handler**

Modify `internal/adapters/http/server.go`:
- Add `appUserHandler *AppUserHandler` field to Server struct
- Update `NewServer` constructor to accept appUserHandler parameter
- Add routes in `registerRoutes()`:
  ```go
  v1.GET("/app-users", s.appUserHandler.List)
  v1.POST("/app-users", s.appUserHandler.Create)
  v1.GET("/app-users/:id", s.appUserHandler.Get)
  v1.PUT("/app-users/:id", s.appUserHandler.Update)
  v1.DELETE("/app-users/:id", s.appUserHandler.Delete)
  ```

- [ ] **Step 11: Update main.go to wire AppUser dependencies**

Modify `cmd/main.go`:
- Create appUserRepo with `postgres.NewAppUserRepository(pool)`
- Create appUserService with `appuser.NewService(appUserRepo)`
- Create appUserHandler with `http.NewAppUserHandler(appUserService)`
- Pass to `http.NewServer(userHandler, appUserHandler)`

- [ ] **Step 12: Test AppUser CRUD**

Run: `go test ./internal/domain/appuser/...`
Expected: All tests pass (if integration tests exist)

Or test via curl:
```bash
# Create
curl -X POST http://localhost:8080/api/v1/app-users \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","email":"test@example.com"}'

# Get
curl http://localhost:8080/api/v1/app-users/<id>

# List
curl http://localhost:8080/api/v1/app-users

# Update
curl -X PUT http://localhost:8080/api/v1/app-users/<id> \
  -H "Content-Type: application/json" \
  -d '{"username":"newuser","email":"new@example.com"}'

# Delete
curl -X DELETE http://localhost:8080/api/v1/app-users/<id>
```

- [ ] **Step 13: Commit AppUser CRUD**

```bash
git add -A
git commit -m "feat: add appuser CRUD with domain service and HTTP handler"
```

---

## Task 2: Track CRUD

**Files:**
- Create: `internal/domain/track/track.go`
- Create: `internal/domain/track/repository.go`
- Create: `internal/domain/track/service.go`
- Create: `internal/domain/track/errors.go`
- Create: `internal/infrastructure/postgres/track_repository.go`
- Create: `internal/infrastructure/postgres/queries/track_queries.go`
- Create: SQL query files for track
- Create: `internal/adapters/http/track_handler.go`
- Modify: `internal/adapters/http/dto.go` - Add Track DTOs
- Modify: `internal/adapters/http/server.go` - Add track handler and routes
- Modify: `cmd/main.go` - Wire Track dependencies

**Interfaces:**
- Consumes: AppUser entity (for user_id foreign key validation)
- Produces: TrackService interface; HTTP routes: GET/POST /api/v1/tracks, GET/PUT/DELETE /api/v1/tracks/:id

- [ ] **Step 1: Create Track domain entity**

```go
// internal/domain/track/track.go
package track

import (
	"time"
)

type Track struct {
	ID              string
	UserID          string
	Route           []byte // GeoJSON or WKB encoded LineString
	DistanceM       *float64
	ElevationGainM  *float64
	ElevationLossM  *float64
	StartedAt       *time.Time
	EndedAt         *time.Time
	CreatedAt       time.Time
}
```

- [ ] **Step 2: Create Track repository interface**

```go
// internal/domain/track/repository.go
package track

import "context"

type Repository interface {
	GetByID(ctx context.Context, id string) (Track, error)
	ListByUserID(ctx context.Context, userID string) ([]Track, error)
	Create(ctx context.Context, t Track) (Track, error)
	Update(ctx context.Context, t Track) (Track, error)
	Delete(ctx context.Context, id string) error
}
```

- [ ] **Step 3: Create Track errors**

```go
// internal/domain/track/errors.go
package track

import "errors"

var (
	ErrNotFound   = errors.New("track not found")
	ErrUserNotFound = errors.New("user not found")
	ErrInvalidDistance = errors.New("distance must be positive")
	ErrInvalidElevation = errors.New("elevation must be positive")
	ErrInvalidDates = errors.New("ended_at must be after or equal to started_at")
)
```

- [ ] **Step 4: Create Track service**

```go
// internal/domain/track/service.go
package track

import (
	"context"
	"time"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Get(ctx context.Context, id string) (Track, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) ListByUser(ctx context.Context, userID string) ([]Track, error) {
	return s.repo.ListByUserID(ctx, userID)
}

func (s *Service) Create(ctx context.Context, t Track) (Track, error) {
	if t.DistanceM != nil && *t.DistanceM < 0 {
		return Track{}, ErrInvalidDistance
	}
	if t.ElevationGainM != nil && *t.ElevationGainM < 0 {
		return Track{}, ErrInvalidElevation
	}
	if t.ElevationLossM != nil && *t.ElevationLossM < 0 {
		return Track{}, ErrInvalidElevation
	}
	if t.StartedAt != nil && t.EndedAt != nil && t.EndedAt.Before(*t.StartedAt) {
		return Track{}, ErrInvalidDates
	}
	return s.repo.Create(ctx, t)
}

func (s *Service) Update(ctx context.Context, t Track) (Track, error) {
	if t.DistanceM != nil && *t.DistanceM < 0 {
		return Track{}, ErrInvalidDistance
	}
	if t.ElevationGainM != nil && *t.ElevationGainM < 0 {
		return Track{}, ErrInvalidElevation
	}
	if t.ElevationLossM != nil && *t.ElevationLossM < 0 {
		return Track{}, ErrInvalidElevation
	}
	if t.StartedAt != nil && t.EndedAt != nil && t.EndedAt.Before(*t.StartedAt) {
		return Track{}, ErrInvalidDates
	}
	return s.repo.Update(ctx, t)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
```

- [ ] **Step 5: Create Track SQL query files**

```sql
-- internal/infrastructure/postgres/queries/get_track.sql
SELECT id, user_id, route, distance_m, elevation_gain_m, elevation_loss_m, started_at, ended_at, created_at
FROM track
WHERE id = $1;

-- internal/infrastructure/postgres/queries/list_tracks_by_user.sql
SELECT id, user_id, route, distance_m, elevation_gain_m, elevation_loss_m, started_at, ended_at, created_at
FROM track
WHERE user_id = $1
ORDER BY created_at DESC;

-- internal/infrastructure/postgres/queries/create_track.sql
INSERT INTO track (user_id, route, distance_m, elevation_gain_m, elevation_loss_m, started_at, ended_at)
VALUES ($1, ST_GeomFromText($2, 4326), $3, $4, $5, $6, $7)
RETURNING id, user_id, ST_AsText(route), distance_m, elevation_gain_m, elevation_loss_m, started_at, ended_at, created_at;

-- internal/infrastructure/postgres/queries/update_track.sql
UPDATE track
SET user_id = $2, route = ST_GeomFromText($3, 4326), distance_m = $4, elevation_gain_m = $5, elevation_loss_m = $6, started_at = $7, ended_at = $8
WHERE id = $1
RETURNING id, user_id, ST_AsText(route), distance_m, elevation_gain_m, elevation_loss_m, started_at, ended_at, created_at;

-- internal/infrastructure/postgres/queries/delete_track.sql
DELETE FROM track WHERE id = $1;
```

- [ ] **Step 6: Create Track queries loader and repository**

```go
// internal/infrastructure/postgres/queries/track_queries.go
package queries

import _ "embed"

//go:embed get_track.sql
var GetTrackQuery string

//go:embed list_tracks_by_user.sql
var ListTracksByUserQuery string

//go:embed create_track.sql
var CreateTrackQuery string

//go:embed update_track.sql
var UpdateTrackQuery string

//go:embed delete_track.sql
var DeleteTrackQuery string
```

```go
// internal/infrastructure/postgres/track_repository.go
package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/traily-org/server/internal/domain/track"
	"github.com/traily-org/server/internal/infrastructure/postgres/queries"
)

type TrackRepository struct {
	pool *pgxpool.Pool
}

func NewTrackRepository(pool *pgxpool.Pool) *TrackRepository {
	return &TrackRepository{pool: pool}
}

func (r *TrackRepository) GetByID(ctx context.Context, id string) (track.Track, error) {
	row := r.pool.QueryRow(ctx, queries.GetTrackQuery, id)
	return scanTrack(row)
}

func (r *TrackRepository) ListByUserID(ctx context.Context, userID string) ([]track.Track, error) {
	rows, err := r.pool.Query(ctx, queries.ListTracksByUserQuery, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tracks []track.Track
	for rows.Next() {
		t, err := scanTrack(rows)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, t)
	}
	return tracks, rows.Err()
}

func (r *TrackRepository) Create(ctx context.Context, t track.Track) (track.Track, error) {
	var routeWKT string
	if t.Route != nil {
		// Assuming Route is WKT format; adjust if using WKB
		routeWKT = string(t.Route)
	}
	row := r.pool.QueryRow(ctx, queries.CreateTrackQuery, t.UserID, routeWKT, t.DistanceM, t.ElevationGainM, t.ElevationLossM, t.StartedAt, t.EndedAt)
	return scanTrack(row)
}

func (r *TrackRepository) Update(ctx context.Context, t track.Track) (track.Track, error) {
	var routeWKT string
	if t.Route != nil {
		routeWKT = string(t.Route)
	}
	row := r.pool.QueryRow(ctx, queries.UpdateTrackQuery, t.ID, t.UserID, routeWKT, t.DistanceM, t.ElevationGainM, t.ElevationLossM, t.StartedAt, t.EndedAt)
	updated, err := scanTrack(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return track.Track{}, track.ErrNotFound
	}
	return updated, err
}

func (r *TrackRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, queries.DeleteTrackQuery, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return track.ErrNotFound
	}
	return nil
}

func scanTrack(row pgx.Row) (track.Track, error) {
	var t track.Track
	var routeWKT string
	err := row.Scan(&t.ID, &t.UserID, &routeWKT, &t.DistanceM, &t.ElevationGainM, &t.ElevationLossM, &t.StartedAt, &t.EndedAt, &t.CreatedAt)
	if err == nil && routeWKT != "" {
		t.Route = []byte(routeWKT)
	}
	return t, err
}
```

- [ ] **Step 7: Add Track DTOs**

```go
// Add to internal/adapters/http/dto.go

type trackResponse struct {
	ID              string     `json:"id"`
	UserID          string     `json:"user_id"`
	Route           string     `json:"route"` // WKT or GeoJSON
	DistanceM       *float64   `json:"distance_m,omitempty"`
	ElevationGainM  *float64   `json:"elevation_gain_m,omitempty"`
	ElevationLossM  *float64   `json:"elevation_loss_m,omitempty"`
	StartedAt       *time.Time `json:"started_at,omitempty"`
	EndedAt         *time.Time `json:"ended_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

func newTrackResponse(t track.Track) trackResponse {
	return trackResponse{
		ID:              t.ID,
		UserID:          t.UserID,
		Route:           string(t.Route),
		DistanceM:       t.DistanceM,
		ElevationGainM:  t.ElevationGainM,
		ElevationLossM:  t.ElevationLossM,
		StartedAt:       t.StartedAt,
		EndedAt:         t.EndedAt,
		CreatedAt:       t.CreatedAt,
	}
}

type createTrackRequest struct {
	UserID          string     `json:"user_id" binding:"required"`
	Route           string     `json:"route" binding:"required"`
	DistanceM       *float64   `json:"distance_m"`
	ElevationGainM  *float64   `json:"elevation_gain_m"`
	ElevationLossM  *float64   `json:"elevation_loss_m"`
	StartedAt       *time.Time `json:"started_at"`
	EndedAt         *time.Time `json:"ended_at"`
}

type updateTrackRequest struct {
	UserID          string     `json:"user_id" binding:"required"`
	Route           string     `json:"route" binding:"required"`
	DistanceM       *float64   `json:"distance_m"`
	ElevationGainM  *float64   `json:"elevation_gain_m"`
	ElevationLossM  *float64   `json:"elevation_loss_m"`
	StartedAt       *time.Time `json:"started_at"`
	EndedAt         *time.Time `json:"ended_at"`
}
```

- [ ] **Step 8: Create Track HTTP handler**

```go
// internal/adapters/http/track_handler.go
package http

import (
	"context"
	"errors"
	nethttp "net/http"

	"github.com/gin-gonic/gin"

	"github.com/traily-org/server/internal/domain/track"
)

type trackService interface {
	Get(ctx context.Context, id string) (track.Track, error)
	ListByUser(ctx context.Context, userID string) ([]track.Track, error)
	Create(ctx context.Context, t track.Track) (track.Track, error)
	Update(ctx context.Context, t track.Track) (track.Track, error)
	Delete(ctx context.Context, id string) error
}

type TrackHandler struct {
	service trackService
}

func NewTrackHandler(service trackService) *TrackHandler {
	return &TrackHandler{service: service}
}

func (h *TrackHandler) Get(ctx *gin.Context) {
	id := ctx.Param("id")
	t, err := h.service.Get(ctx.Request.Context(), id)
	if err != nil {
		writeTrackError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusOK, newTrackResponse(t))
}

func (h *TrackHandler) ListByUser(ctx *gin.Context) {
	userID := ctx.Query("user_id")
	if userID == "" {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": "user_id query parameter required"})
		return
	}

	tracks, err := h.service.ListByUser(ctx.Request.Context(), userID)
	if err != nil {
		ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	responses := make([]trackResponse, len(tracks))
	for i, t := range tracks {
		responses[i] = newTrackResponse(t)
	}
	ctx.JSON(nethttp.StatusOK, responses)
}

func (h *TrackHandler) Create(ctx *gin.Context) {
	var req createTrackRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	t, err := h.service.Create(ctx.Request.Context(), track.Track{
		UserID:          req.UserID,
		Route:           []byte(req.Route),
		DistanceM:       req.DistanceM,
		ElevationGainM:  req.ElevationGainM,
		ElevationLossM:  req.ElevationLossM,
		StartedAt:       req.StartedAt,
		EndedAt:         req.EndedAt,
	})
	if err != nil {
		writeTrackError(ctx, err)
		return
	}

	ctx.JSON(nethttp.StatusCreated, newTrackResponse(t))
}

func (h *TrackHandler) Update(ctx *gin.Context) {
	id := ctx.Param("id")

	var req updateTrackRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	t, err := h.service.Update(ctx.Request.Context(), track.Track{
		ID:              id,
		UserID:          req.UserID,
		Route:           []byte(req.Route),
		DistanceM:       req.DistanceM,
		ElevationGainM:  req.ElevationGainM,
		ElevationLossM:  req.ElevationLossM,
		StartedAt:       req.StartedAt,
		EndedAt:         req.EndedAt,
	})
	if err != nil {
		writeTrackError(ctx, err)
		return
	}

	ctx.JSON(nethttp.StatusOK, newTrackResponse(t))
}

func (h *TrackHandler) Delete(ctx *gin.Context) {
	id := ctx.Param("id")
	if err := h.service.Delete(ctx.Request.Context(), id); err != nil {
		writeTrackError(ctx, err)
		return
	}
	ctx.Status(nethttp.StatusNoContent)
}

func writeTrackError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, track.ErrNotFound):
		ctx.JSON(nethttp.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, track.ErrInvalidDistance), errors.Is(err, track.ErrInvalidElevation), errors.Is(err, track.ErrInvalidDates):
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		ctx.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
```

- [ ] **Step 9: Update server.go with Track handler and routes**

Add to `internal/adapters/http/server.go`:
```go
v1.GET("/tracks/:id", s.trackHandler.Get)
v1.GET("/tracks", s.trackHandler.ListByUser)
v1.POST("/tracks", s.trackHandler.Create)
v1.PUT("/tracks/:id", s.trackHandler.Update)
v1.DELETE("/tracks/:id", s.trackHandler.Delete)
```

- [ ] **Step 10: Wire Track dependencies in main.go**

```bash
git add -A
git commit -m "feat: add track CRUD with PostGIS support"
```

---

## Task 3: Trace CRUD

**Files:**
- Create: `internal/domain/trace/trace.go`
- Create: `internal/domain/trace/repository.go`
- Create: `internal/domain/trace/service.go`
- Create: `internal/domain/trace/errors.go`
- Create: `internal/infrastructure/postgres/trace_repository.go`
- Create: SQL queries for trace
- Create: `internal/adapters/http/trace_handler.go`
- Modify: `internal/adapters/http/dto.go` - Add Trace DTOs
- Modify: `internal/adapters/http/server.go` - Add trace handler and routes
- Modify: `cmd/main.go` - Wire Trace dependencies

**Interfaces:**
- Consumes: Track entity, AppUser via user_id
- Produces: TraceService; HTTP routes: GET/POST /api/v1/traces, GET/PUT/DELETE /api/v1/traces/:id

[Follow identical pattern to Task 2 for Trace entity with fields: ID, UserID, TrackID, Name, Description, Difficulty, CreatedAt, UpdatedAt]

- [ ] **Step 1-10: Implement Trace following Task 2 pattern**

Trace-specific details:
- Enum: `trace_difficulty` (easy, moderate, hard, expert)
- Required fields: ID, UserID, TrackID, Name
- Optional fields: Description, Difficulty
- Unique: TrackID (one-to-one with Track)
- Foreign keys: user_id → app_user, track_id → track (RESTRICT on delete to prevent orphaning)

- [ ] **Step 11: Commit Trace CRUD**

```bash
git commit -m "feat: add trace CRUD with difficulty levels"
```

---

## Task 4: Activity CRUD

**Files:** Similar structure to Task 2 and 3

**Interfaces:**
- Consumes: Track, Trace (optional), AppUser
- Produces: ActivityService; HTTP routes for activity CRUD

[Follow pattern; Activity has optional trace_id and complex date validation]

- [ ] **Step 1-11: Implement Activity**

Activity-specific:
- Foreign keys: user_id → app_user, track_id → track (RESTRICT), trace_id → trace (SET NULL)
- Date constraint: ended_at >= started_at (validated in service)
- Optional fields: name, started_at, ended_at, trace_id

- [ ] **Step 12: Commit Activity CRUD**

```bash
git commit -m "feat: add activity CRUD with date validation"
```

---

## Task 5: Publication CRUD

**Files:** Similar structure to previous tasks

**Interfaces:**
- Consumes: Track, Trace (optional), AppUser
- Produces: PublicationService; HTTP routes

[Follow pattern; Publication represents published tracks/activities]

- [ ] **Step 1-11: Implement Publication**

Publication-specific:
- Foreign keys: user_id → app_user, track_id → track (RESTRICT), trace_id → trace (SET NULL)
- Optional fields: title, content, trace_id
- Timestamp: published_at (default NOW)

- [ ] **Step 12: Commit Publication CRUD**

```bash
git commit -m "feat: add publication CRUD for user content"
```

---

## Task 6: POI (Point of Interest) CRUD

**Files:** Similar structure; note geospatial requirement

**Interfaces:**
- Consumes: AppUser (optional creator)
- Produces: POI Service; HTTP routes

- [ ] **Step 1-11: Implement POI**

POI-specific:
- Geospatial: location GEOMETRY(Point, 4326) with GIST index
- Enum: poi_type (viewpoint, waterfall, lake, river, mountain, cave, refuge, monument, historical, restaurant, parking, camping, other)
- Foreign key: user_id → app_user (SET NULL - POI can exist without creator)
- Optional field: user_id (user who created it), description
- Index on location for proximity queries

- [ ] **Step 12: Commit POI CRUD**

```bash
git commit -m "feat: add POI CRUD with geospatial support"
```

---

## Task 7: Herbier (Plant Notebook) CRUD

**Files:** Similar structure

**Interfaces:**
- Consumes: AppUser
- Produces: HerbierService; HTTP routes

- [ ] **Step 1-11: Implement Herbier**

Herbier-specific:
- Unique user_id constraint (one herbier per user)
- Default name: "Mon herbier"
- Foreign key: user_id → app_user (CASCADE)

- [ ] **Step 12: Commit Herbier CRUD**

```bash
git commit -m "feat: add herbier CRUD with per-user plant notebook"
```

---

## Task 8: PlantSpecies CRUD

**Files:** Similar structure; simpler (no user association)

**Interfaces:**
- Produces: PlantSpeciesService; HTTP routes

- [ ] **Step 1-11: Implement PlantSpecies**

PlantSpecies-specific:
- No user association (reference data)
- Fields: ID, CommonName (required), ScientificName (optional), Description (optional)
- Index on CommonName for search

- [ ] **Step 12: Commit PlantSpecies CRUD**

```bash
git commit -m "feat: add plant species CRUD as reference data"
```

---

## Task 9: HerbierEntry CRUD

**Files:** Similar structure; most complex relationships

**Interfaces:**
- Consumes: Herbier, PlantSpecies, Activity (optional), POI (optional)
- Produces: HerbierEntryService; HTTP routes

- [ ] **Step 1-11: Implement HerbierEntry**

HerbierEntry-specific:
- Geospatial: location GEOMETRY(Point, 4326) optional
- Foreign keys: herbier_id (CASCADE), plant_species_id (RESTRICT), activity_id (SET NULL), poi_id (SET NULL)
- Optional fields: activity_id, poi_id, photo_url, notes, location, observed_at
- Timestamp: created_at (not updated_at — immutable entries)

- [ ] **Step 12: Commit HerbierEntry CRUD**

```bash
git commit -m "feat: add herbier entry CRUD with plant tracking"
```

---

## Task 10: Integration and Relationship Testing

**Files:**
- Modify: `cmd/main.go` - Wire all handler dependencies
- Modify: `internal/adapters/http/server.go` - Ensure all routes registered

- [ ] **Step 1: Wire all repositories in main.go**

```go
// cmd/main.go - in init function or main
appUserRepo := postgres.NewAppUserRepository(pool)
trackRepo := postgres.NewTrackRepository(pool)
traceRepo := postgres.NewTraceRepository(pool)
activityRepo := postgres.NewActivityRepository(pool)
publicationRepo := postgres.NewPublicationRepository(pool)
poiRepo := postgres.NewPOIRepository(pool)
herbierRepo := postgres.NewHerbierRepository(pool)
plantSpeciesRepo := postgres.NewPlantSpeciesRepository(pool)
herbierEntryRepo := postgres.NewHerbierEntryRepository(pool)
```

- [ ] **Step 2: Wire all services**

```go
appUserService := appuser.NewService(appUserRepo)
trackService := track.NewService(trackRepo)
traceService := trace.NewService(traceRepo)
activityService := activity.NewService(activityRepo)
publicationService := publication.NewService(publicationRepo)
poiService := poi.NewService(poiRepo)
herbierService := herbier.NewService(herbierRepo)
plantSpeciesService := plantspecies.NewService(plantSpeciesRepo)
herbierEntryService := herbierentry.NewService(herbierEntryRepo)
```

- [ ] **Step 3: Wire all handlers**

```go
appUserHandler := http.NewAppUserHandler(appUserService)
trackHandler := http.NewTrackHandler(trackService)
traceHandler := http.NewTraceHandler(traceService)
activityHandler := http.NewActivityHandler(activityService)
publicationHandler := http.NewPublicationHandler(publicationService)
poiHandler := http.NewPOIHandler(poiService)
herbierHandler := http.NewHerbierHandler(herbierService)
plantSpeciesHandler := http.NewPlantSpeciesHandler(plantSpeciesService)
herbierEntryHandler := http.NewHerbierEntryHandler(herbierEntryService)

server := http.NewServer(
	userHandler,
	appUserHandler,
	trackHandler,
	traceHandler,
	activityHandler,
	publicationHandler,
	poiHandler,
	herbierHandler,
	plantSpeciesHandler,
	herbierEntryHandler,
)
```

- [ ] **Step 4: Verify all routes are registered in server.go**

Ensure Server struct has all handlers and registerRoutes() registers all endpoints.

- [ ] **Step 5: Run the application and test endpoints**

```bash
go run cmd/main.go

# Test a workflow: create app-user, create track, create trace, create activity
curl -X POST http://localhost:8080/api/v1/app-users \
  -H "Content-Type: application/json" \
  -d '{"username":"hiker","email":"hiker@example.com"}'
```

- [ ] **Step 6: Commit integration**

```bash
git add cmd/main.go internal/adapters/http/server.go
git commit -m "feat: wire all entity CRUDs and register HTTP routes"
```

---

## Task 11: Final Verification and Documentation

- [ ] **Step 1: Run full test suite**

```bash
go test ./...
```

Expected: All tests pass

- [ ] **Step 2: Build application**

```bash
go build -o bin/server cmd/main.go
```

Expected: No build errors

- [ ] **Step 3: Document API endpoints**

Create or update `docs/API.md` with all endpoint documentation including query parameters, request/response formats.

- [ ] **Step 4: Clean up and final commit**

```bash
git add docs/
git commit -m "docs: add API endpoint documentation"
```

---

## Verification Checklist

Before considering this plan complete:

1. ✅ All 9 entities have complete CRUD implementations
2. ✅ Each entity follows hexagonal architecture (domain/infrastructure/adapter layers)
3. ✅ All foreign key relationships are validated
4. ✅ All geospatial queries properly use PostGIS functions
5. ✅ All datetime fields are TIMESTAMPTZ with proper timezone handling
6. ✅ All HTTP handlers use proper status codes and error responses
7. ✅ All requests/responses have validation binding tags
8. ✅ All repositories implement their interfaces completely
9. ✅ All services validate business logic before delegating to repositories
10. ✅ Application builds and runs without errors
