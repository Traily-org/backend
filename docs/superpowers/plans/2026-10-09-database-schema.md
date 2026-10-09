# Database Schema Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the complete PostgreSQL database schema for Traily hiking app, including user management, geospatial tracks, activities, POI references, and a plant observation herbarium system.

**Architecture:** The schema uses PostGIS for geographic data (LineString routes and Point locations) and defines entity relationships across three main domains: hiking activities (tracks, traces, activities), geographic features (POIs), and plant observations (herbier). Migrations are embedded via `go:embed` and applied sequentially using a `schema_migrations` tracking table. Domain models follow clean architecture patterns with separate packages per domain entity.

**Tech Stack:** 
- PostgreSQL 14+ with PostGIS extension
- pgcrypto for UUID generation
- Go 1.21+ with pgx/v5 for database access
- Embedded SQL migrations via go:embed

**Spec:** Provided SQL schema with 13 tables, 2 ENUMs, and PostGIS/pgcrypto extensions

## Global Constraints

- All timestamps use TIMESTAMPTZ (timezone-aware)
- All primary keys are UUID with `gen_random_uuid()` default
- Foreign keys cascade on delete except where explicitly RESTRICT
- Geographic data uses SRID 4326 (WGS84/lat-long coordinates)
- Migrations are numbered sequentially (002_, 003_, etc.) and applied in alphabetical order
- Go domain models use the same field names as database columns for seamless mapping

## Review Focus

1. **PostGIS extension initialization** — `geo/location queries will silently fail if PostGIS is not installed; verify EXTENSION creation happens first`
2. **Constraint enforcement on temporal data** — Activity/Track dates can be NULL but if both set must satisfy `ended_at >= started_at`
3. **Cascade vs RESTRICT trade-offs** — Track/Publication/Activity must RESTRICT to prevent orphaning associated traces; verify the right policies are applied
4. **UUID generation consistency** — All tables default to `gen_random_uuid()`; verify no table is missing this
5. **Index coverage for foreign keys** — Tables with frequent lookups (trace_poi, activity_poi) need explicit indexes on the POI/Activity side

---

## File Structure

**Migration Files (sequential SQL):**
- `migrations/002_create_extensions.sql` — PostGIS and pgcrypto extensions
- `migrations/003_create_enums.sql` — trace_difficulty, poi_type enums
- `migrations/004_create_app_user.sql` — User table (replaces old `users` table)
- `migrations/005_create_track.sql` — Track table with geospatial route
- `migrations/006_create_trace.sql` — Reference trace with track reference
- `migrations/007_create_activity.sql` — Completed hike activity
- `migrations/008_create_publication.sql` — Track publication/post
- `migrations/009_create_poi.sql` — Points of Interest
- `migrations/010_create_trace_poi.sql` — Many-to-many trace→POI
- `migrations/011_create_activity_poi.sql` — Many-to-many activity→POI
- `migrations/012_create_herbier.sql` — Plant observation collection
- `migrations/013_create_plant_species.sql` — Plant species catalog
- `migrations/014_create_herbier_entry.sql` — Individual plant observations

**Go Domain Models (clean architecture):**
- `internal/domain/track/track.go` — Track domain model
- `internal/domain/trace/trace.go` — Trace domain model
- `internal/domain/activity/activity.go` — Activity domain model
- `internal/domain/publication/publication.go` — Publication domain model
- `internal/domain/poi/poi.go` — Point of Interest domain model
- `internal/domain/herbier/herbier.go` — Herbarium collection model
- `internal/domain/herbier/plant_species.go` — Plant species model
- `internal/domain/herbier/herbier_entry.go` — Plant observation model

---

## Task Breakdown

### Task 1: Create Extensions Migration

**Files:**
- Create: `migrations/002_create_extensions.sql`

**Interfaces:**
- Produces: PostgreSQL `postgis` and `pgcrypto` extensions available for all subsequent migrations

- [ ] **Step 1: Write the migration file with both extensions**

```sql
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS pgcrypto;
```

- [ ] **Step 2: Verify migration syntax is valid**

Run: `cat migrations/002_create_extensions.sql`
Expected: File contains two CREATE EXTENSION statements

- [ ] **Step 3: Commit**

```bash
git add migrations/002_create_extensions.sql
git commit -m "feat: add postgis and pgcrypto extensions"
```

---

### Task 2: Create ENUMs Migration

**Files:**
- Create: `migrations/003_create_enums.sql`

**Interfaces:**
- Consumes: postgis, pgcrypto extensions from Task 1
- Produces: `trace_difficulty` enum (easy, moderate, hard, expert) and `poi_type` enum (13 values)

- [ ] **Step 1: Write the migration file with both ENUMs**

```sql
CREATE TYPE trace_difficulty AS ENUM (
    'easy',
    'moderate',
    'hard',
    'expert'
);

CREATE TYPE poi_type AS ENUM (
    'viewpoint',
    'waterfall',
    'lake',
    'river',
    'mountain',
    'cave',
    'refuge',
    'monument',
    'historical',
    'restaurant',
    'parking',
    'camping',
    'other'
);
```

- [ ] **Step 2: Verify enum syntax**

Run: `cat migrations/003_create_enums.sql | grep -E "CREATE TYPE|'[a-z]+'" | head -5`
Expected: File shows both TYPE definitions with proper enum values

- [ ] **Step 3: Commit**

```bash
git add migrations/003_create_enums.sql
git commit -m "feat: create trace_difficulty and poi_type enums"
```

---

### Task 3: Create app_user Table Migration

**Files:**
- Create: `migrations/004_create_app_user.sql`

**Interfaces:**
- Consumes: pgcrypto extension from Task 1
- Produces: `app_user` table with UUID primary key, username/email unique constraints, timestamps

- [ ] **Step 1: Write the migration file**

```sql
CREATE TABLE app_user (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    display_name VARCHAR(100),
    avatar_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

- [ ] **Step 2: Verify schema structure**

Run: `cat migrations/004_create_app_user.sql | grep -E "CREATE TABLE|id UUID|username|email"`
Expected: File contains table definition with all required columns

- [ ] **Step 3: Commit**

```bash
git add migrations/004_create_app_user.sql
git commit -m "feat: create app_user table"
```

---

### Task 4: Create Track Table Migration

**Files:**
- Create: `migrations/005_create_track.sql`

**Interfaces:**
- Consumes: pgcrypto extension, app_user table from Tasks 1 and 3
- Produces: `track` table with PostGIS LineString route geometry, distance/elevation stats, user foreign key, and indexes

- [ ] **Step 1: Write the migration file with GIST index for geometry**

```sql
CREATE TABLE track (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL
        REFERENCES app_user(id)
        ON DELETE CASCADE,
    route GEOMETRY(LineString, 4326) NOT NULL,
    distance_m DOUBLE PRECISION,
    elevation_gain_m DOUBLE PRECISION,
    elevation_loss_m DOUBLE PRECISION,
    started_at TIMESTAMPTZ,
    ended_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT track_distance_positive
        CHECK (distance_m IS NULL OR distance_m >= 0),
    CONSTRAINT track_elevation_gain_positive
        CHECK (elevation_gain_m IS NULL OR elevation_gain_m >= 0),
    CONSTRAINT track_elevation_loss_positive
        CHECK (elevation_loss_m IS NULL OR elevation_loss_m >= 0),
    CONSTRAINT track_dates_valid
        CHECK (
            ended_at IS NULL
            OR started_at IS NULL
            OR ended_at >= started_at
        )
);

CREATE INDEX idx_track_user
    ON track(user_id);

CREATE INDEX idx_track_route
    ON track USING GIST(route);
```

- [ ] **Step 2: Verify constraints and indexes**

Run: `cat migrations/005_create_track.sql | grep -E "CONSTRAINT|CREATE INDEX"`
Expected: Shows 5 total items: 3 constraints + 2 indexes

- [ ] **Step 3: Commit**

```bash
git add migrations/005_create_track.sql
git commit -m "feat: create track table with geospatial route"
```

---

### Task 5: Create Trace Table Migration

**Files:**
- Create: `migrations/006_create_trace.sql`

**Interfaces:**
- Consumes: pgcrypto extension, app_user table, track table, trace_difficulty enum from Tasks 1, 3, 4, and 2
- Produces: `trace` table with unique track_id reference (1:1), difficulty level, and user ownership

- [ ] **Step 1: Write the migration file**

```sql
CREATE TABLE trace (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL
        REFERENCES app_user(id)
        ON DELETE CASCADE,
    track_id UUID NOT NULL UNIQUE
        REFERENCES track(id)
        ON DELETE RESTRICT,
    name VARCHAR(150) NOT NULL,
    description TEXT,
    difficulty trace_difficulty,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_trace_user
    ON trace(user_id);
```

- [ ] **Step 2: Verify UNIQUE constraint on track_id**

Run: `cat migrations/006_create_trace.sql | grep -A 2 "track_id UUID"`
Expected: Shows `UNIQUE` keyword before REFERENCES

- [ ] **Step 3: Commit**

```bash
git add migrations/006_create_trace.sql
git commit -m "feat: create trace reference table"
```

---

### Task 6: Create Activity Table Migration

**Files:**
- Create: `migrations/007_create_activity.sql`

**Interfaces:**
- Consumes: pgcrypto extension, app_user table, track table, trace table from Tasks 1, 3, 4, and 6
- Produces: `activity` table with UNIQUE track_id (1:1), optional trace reference, temporal constraints

- [ ] **Step 1: Write the migration file**

```sql
CREATE TABLE activity (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL
        REFERENCES app_user(id)
        ON DELETE CASCADE,
    track_id UUID NOT NULL UNIQUE
        REFERENCES track(id)
        ON DELETE RESTRICT,
    trace_id UUID
        REFERENCES trace(id)
        ON DELETE SET NULL,
    name VARCHAR(150),
    started_at TIMESTAMPTZ,
    ended_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT activity_dates_valid
        CHECK (
            ended_at IS NULL
            OR started_at IS NULL
            OR ended_at >= started_at
        )
);

CREATE INDEX idx_activity_user
    ON activity(user_id);

CREATE INDEX idx_activity_trace
    ON activity(trace_id);
```

- [ ] **Step 2: Verify indexes exist for foreign key lookups**

Run: `cat migrations/007_create_activity.sql | grep "CREATE INDEX"`
Expected: Shows 2 indexes (user and trace)

- [ ] **Step 3: Commit**

```bash
git add migrations/007_create_activity.sql
git commit -m "feat: create activity table for completed hikes"
```

---

### Task 7: Create Publication Table Migration

**Files:**
- Create: `migrations/008_create_publication.sql`

**Interfaces:**
- Consumes: pgcrypto extension, app_user table, track table, trace table from Tasks 1, 3, 4, and 6
- Produces: `publication` table with required track_id (RESTRICT), optional trace_id (SET NULL), temporal tracking

- [ ] **Step 1: Write the migration file**

```sql
CREATE TABLE publication (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL
        REFERENCES app_user(id)
        ON DELETE CASCADE,
    track_id UUID NOT NULL
        REFERENCES track(id)
        ON DELETE RESTRICT,
    trace_id UUID
        REFERENCES trace(id)
        ON DELETE SET NULL,
    title VARCHAR(200),
    content TEXT,
    published_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_publication_user
    ON publication(user_id);

CREATE INDEX idx_publication_track
    ON publication(track_id);

CREATE INDEX idx_publication_trace
    ON publication(trace_id);

CREATE INDEX idx_publication_date
    ON publication(published_at DESC);
```

- [ ] **Step 2: Verify published_at index is DESC for reverse-chronological queries**

Run: `cat migrations/008_create_publication.sql | grep "idx_publication_date"`
Expected: Shows `DESC` keyword in the index definition

- [ ] **Step 3: Commit**

```bash
git add migrations/008_create_publication.sql
git commit -m "feat: create publication table for track sharing"
```

---

### Task 8: Create POI Table Migration

**Files:**
- Create: `migrations/009_create_poi.sql`

**Interfaces:**
- Consumes: pgcrypto extension, PostGIS from Task 1, app_user table from Task 3, poi_type enum from Task 2
- Produces: `poi` table with Point geometry, optional user owner, GIST index for spatial queries

- [ ] **Step 1: Write the migration file**

```sql
CREATE TABLE poi (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID
        REFERENCES app_user(id)
        ON DELETE SET NULL,
    name VARCHAR(150) NOT NULL,
    description TEXT,
    type poi_type NOT NULL DEFAULT 'other',
    location GEOMETRY(Point, 4326) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_poi_location
    ON poi USING GIST(location);
```

- [ ] **Step 2: Verify GIST index for spatial queries**

Run: `cat migrations/009_create_poi.sql | grep "GIST"`
Expected: Shows GIST index on location column

- [ ] **Step 3: Commit**

```bash
git add migrations/009_create_poi.sql
git commit -m "feat: create poi table for points of interest"
```

---

### Task 9: Create Trace-POI Junction Table Migration

**Files:**
- Create: `migrations/010_create_trace_poi.sql`

**Interfaces:**
- Consumes: trace table from Task 6, poi table from Task 8
- Produces: `trace_poi` junction table with composite primary key, order tracking, and distance metadata

- [ ] **Step 1: Write the migration file**

```sql
CREATE TABLE trace_poi (
    trace_id UUID NOT NULL
        REFERENCES trace(id)
        ON DELETE CASCADE,
    poi_id UUID NOT NULL
        REFERENCES poi(id)
        ON DELETE CASCADE,
    order_index INTEGER,
    distance_from_start_m DOUBLE PRECISION,
    PRIMARY KEY (trace_id, poi_id)
);

CREATE INDEX idx_trace_poi_poi
    ON trace_poi(poi_id);
```

- [ ] **Step 2: Verify composite primary key**

Run: `cat migrations/010_create_trace_poi.sql | grep "PRIMARY KEY"`
Expected: Shows PRIMARY KEY with both trace_id and poi_id

- [ ] **Step 3: Commit**

```bash
git add migrations/010_create_trace_poi.sql
git commit -m "feat: create trace_poi junction table"
```

---

### Task 10: Create Activity-POI Junction Table Migration

**Files:**
- Create: `migrations/011_create_activity_poi.sql`

**Interfaces:**
- Consumes: activity table from Task 7, poi table from Task 8
- Produces: `activity_poi` junction table with visited_at timestamp tracking

- [ ] **Step 1: Write the migration file**

```sql
CREATE TABLE activity_poi (
    activity_id UUID NOT NULL
        REFERENCES activity(id)
        ON DELETE CASCADE,
    poi_id UUID NOT NULL
        REFERENCES poi(id)
        ON DELETE CASCADE,
    visited_at TIMESTAMPTZ,
    PRIMARY KEY (activity_id, poi_id)
);

CREATE INDEX idx_activity_poi_poi
    ON activity_poi(poi_id);
```

- [ ] **Step 2: Verify composite primary key and visited_at timestamp**

Run: `cat migrations/011_create_activity_poi.sql | grep -E "PRIMARY KEY|visited_at"`
Expected: Shows both items in the output

- [ ] **Step 3: Commit**

```bash
git add migrations/011_create_activity_poi.sql
git commit -m "feat: create activity_poi junction table"
```

---

### Task 11: Create Herbier Table Migration

**Files:**
- Create: `migrations/012_create_herbier.sql`

**Interfaces:**
- Consumes: pgcrypto extension, app_user table from Tasks 1 and 3
- Produces: `herbier` table with UNIQUE user_id (1:1) — each user has one collection

- [ ] **Step 1: Write the migration file**

```sql
CREATE TABLE herbier (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE
        REFERENCES app_user(id)
        ON DELETE CASCADE,
    name VARCHAR(150) NOT NULL DEFAULT 'Mon herbier',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

- [ ] **Step 2: Verify UNIQUE constraint on user_id**

Run: `cat migrations/012_create_herbier.sql | grep -A 1 "user_id UUID"`
Expected: Shows UNIQUE keyword on the same line or next

- [ ] **Step 3: Commit**

```bash
git add migrations/012_create_herbier.sql
git commit -m "feat: create herbier plant collection table"
```

---

### Task 12: Create Plant Species Table Migration

**Files:**
- Create: `migrations/013_create_plant_species.sql`

**Interfaces:**
- Consumes: pgcrypto extension from Task 1
- Produces: `plant_species` reference table (shared catalog, no user ownership)

- [ ] **Step 1: Write the migration file**

```sql
CREATE TABLE plant_species (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    common_name VARCHAR(150) NOT NULL,
    scientific_name VARCHAR(200),
    description TEXT
);

CREATE INDEX idx_plant_species_common_name
    ON plant_species(common_name);
```

- [ ] **Step 2: Verify index on common_name for search queries**

Run: `cat migrations/013_create_plant_species.sql | grep "idx_plant_species_common_name"`
Expected: Shows index creation statement

- [ ] **Step 3: Commit**

```bash
git add migrations/013_create_plant_species.sql
git commit -m "feat: create plant_species reference table"
```

---

### Task 13: Create Herbier Entry Table Migration

**Files:**
- Create: `migrations/014_create_herbier_entry.sql`

**Interfaces:**
- Consumes: herbier table from Task 12, activity table from Task 7, poi table from Task 8, plant_species table from Task 12, PostGIS from Task 1
- Produces: `herbier_entry` table with optional activity/poi references, Point geometry for observation location

- [ ] **Step 1: Write the migration file**

```sql
CREATE TABLE herbier_entry (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    herbier_id UUID NOT NULL
        REFERENCES herbier(id)
        ON DELETE CASCADE,
    activity_id UUID
        REFERENCES activity(id)
        ON DELETE SET NULL,
    poi_id UUID
        REFERENCES poi(id)
        ON DELETE SET NULL,
    plant_species_id UUID NOT NULL
        REFERENCES plant_species(id)
        ON DELETE RESTRICT,
    photo_url TEXT,
    notes TEXT,
    location GEOMETRY(Point, 4326),
    observed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_herbier_entry_herbier
    ON herbier_entry(herbier_id);

CREATE INDEX idx_herbier_entry_activity
    ON herbier_entry(activity_id);

CREATE INDEX idx_herbier_entry_species
    ON herbier_entry(plant_species_id);

CREATE INDEX idx_herbier_entry_location
    ON herbier_entry USING GIST(location);
```

- [ ] **Step 2: Verify all required indexes exist**

Run: `cat migrations/014_create_herbier_entry.sql | grep "CREATE INDEX" | wc -l`
Expected: Shows 4 (herbier, activity, species, location)

- [ ] **Step 3: Commit**

```bash
git add migrations/014_create_herbier_entry.sql
git commit -m "feat: create herbier_entry table for plant observations"
```

---

### Task 14: Create Track Domain Model

**Files:**
- Create: `internal/domain/track/track.go`

**Interfaces:**
- Consumes: Nothing (base model)
- Produces: `Track` struct with UUID, user_id, geometry, distance/elevation, temporal bounds

- [ ] **Step 1: Create the domain model file**

```go
package track

import (
	"time"
)

type Track struct {
	ID               string
	UserID           string
	Route            string // PostGIS LineString WKT or serialized geometry
	DistanceM        *float64
	ElevationGainM   *float64
	ElevationLossM   *float64
	StartedAt        *time.Time
	EndedAt          *time.Time
	CreatedAt        time.Time
}
```

- [ ] **Step 2: Verify struct field names match database columns**

Run: `grep -E "type Track struct|^	[A-Z]" internal/domain/track/track.go`
Expected: Shows Track struct with 10 fields matching database schema

- [ ] **Step 3: Commit**

```bash
git add internal/domain/track/track.go
git commit -m "feat: create track domain model"
```

---

### Task 15: Create Trace Domain Model

**Files:**
- Create: `internal/domain/trace/trace.go`

**Interfaces:**
- Consumes: Nothing (base model)
- Produces: `Trace` struct with UUID, user_id, track_id, name, description, difficulty

- [ ] **Step 1: Create the domain model file**

```go
package trace

import (
	"database/sql"
	"time"
)

type Difficulty string

const (
	DifficultyEasy     Difficulty = "easy"
	DifficultyModerate Difficulty = "moderate"
	DifficultyHard     Difficulty = "hard"
	DifficultyExpert   Difficulty = "expert"
)

type Trace struct {
	ID          string
	UserID      string
	TrackID     string
	Name        string
	Description *string
	Difficulty  sql.NullString
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
```

- [ ] **Step 2: Verify Difficulty constants match enum values**

Run: `grep "Difficulty.*=" internal/domain/trace/trace.go | head -4`
Expected: Shows 4 difficulty constants (easy, moderate, hard, expert)

- [ ] **Step 3: Commit**

```bash
git add internal/domain/trace/trace.go
git commit -m "feat: create trace domain model with difficulty levels"
```

---

### Task 16: Create Activity Domain Model

**Files:**
- Create: `internal/domain/activity/activity.go`

**Interfaces:**
- Consumes: Nothing (base model)
- Produces: `Activity` struct with UUID, user_id, track_id, optional trace_id, name, temporal bounds

- [ ] **Step 1: Create the domain model file**

```go
package activity

import (
	"time"
)

type Activity struct {
	ID        string
	UserID    string
	TrackID   string
	TraceID   *string
	Name      *string
	StartedAt *time.Time
	EndedAt   *time.Time
	CreatedAt time.Time
}
```

- [ ] **Step 2: Verify pointers for optional fields**

Run: `grep -E "ID|Name|StartedAt|EndedAt|CreatedAt" internal/domain/activity/activity.go`
Expected: Optional fields (Name, StartedAt, EndedAt, TraceID) use pointers; required fields do not

- [ ] **Step 3: Commit**

```bash
git add internal/domain/activity/activity.go
git commit -m "feat: create activity domain model"
```

---

### Task 17: Create Publication Domain Model

**Files:**
- Create: `internal/domain/publication/publication.go`

**Interfaces:**
- Consumes: Nothing (base model)
- Produces: `Publication` struct with UUID, user_id, track_id (required), optional trace_id, title, content

- [ ] **Step 1: Create the domain model file**

```go
package publication

import (
	"time"
)

type Publication struct {
	ID          string
	UserID      string
	TrackID     string
	TraceID     *string
	Title       *string
	Content     *string
	PublishedAt time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
```

- [ ] **Step 2: Verify required vs optional fields**

Run: `grep -E "^	[A-Za-z].*\*" internal/domain/publication/publication.go | wc -l`
Expected: Shows 3 optional fields (TraceID, Title, Content use pointers)

- [ ] **Step 3: Commit**

```bash
git add internal/domain/publication/publication.go
git commit -m "feat: create publication domain model"
```

---

### Task 18: Create POI Domain Model

**Files:**
- Create: `internal/domain/poi/poi.go`

**Interfaces:**
- Consumes: Nothing (base model)
- Produces: `POI` struct with UUID, optional user_id, name, type, Point geometry

- [ ] **Step 1: Create the domain model file with POI type enum**

```go
package poi

import (
	"time"
)

type POIType string

const (
	POITypeViewpoint   POIType = "viewpoint"
	POITypeWaterfall   POIType = "waterfall"
	POITypeLake        POIType = "lake"
	POITypeRiver       POIType = "river"
	POITypeMountain    POIType = "mountain"
	POITypeCave        POIType = "cave"
	POITypeRefuge      POIType = "refuge"
	POITypeMonument    POIType = "monument"
	POITypeHistorical  POIType = "historical"
	POITypeRestaurant  POIType = "restaurant"
	POITypeParking     POIType = "parking"
	POITypeCamping     POIType = "camping"
	POITypeOther       POIType = "other"
)

type POI struct {
	ID          string
	UserID      *string
	Name        string
	Description *string
	Type        POIType
	Location    string // PostGIS Point WKT or serialized geometry
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
```

- [ ] **Step 2: Verify all 13 POI type constants exist**

Run: `grep "POIType.*=" internal/domain/poi/poi.go | wc -l`
Expected: Shows 13 constants

- [ ] **Step 3: Commit**

```bash
git add internal/domain/poi/poi.go
git commit -m "feat: create poi domain model with 13 location types"
```

---

### Task 19: Create Herbier Domain Models

**Files:**
- Create: `internal/domain/herbier/herbier.go`
- Create: `internal/domain/herbier/plant_species.go`
- Create: `internal/domain/herbier/herbier_entry.go`

**Interfaces:**
- Consumes: Nothing (base models)
- Produces: Three structs for herbarium collection, plant catalog, and observations

- [ ] **Step 1: Create herbier.go for the collection**

```go
package herbier

import (
	"time"
)

type Herbier struct {
	ID        string
	UserID    string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}
```

- [ ] **Step 2: Create plant_species.go for the shared catalog**

```go
package herbier

type PlantSpecies struct {
	ID             string
	CommonName     string
	ScientificName *string
	Description    *string
}
```

- [ ] **Step 3: Create herbier_entry.go for observations**

```go
package herbier

import (
	"time"
)

type HerbierEntry struct {
	ID              string
	HerbierID       string
	ActivityID      *string
	POIID           *string
	PlantSpeciesID  string
	PhotoURL        *string
	Notes           *string
	Location        *string // PostGIS Point WKT or serialized geometry
	ObservedAt      *time.Time
	CreatedAt       time.Time
}
```

- [ ] **Step 4: Verify all three files exist**

Run: `ls -1 internal/domain/herbier/`
Expected: Shows herbier.go, plant_species.go, herbier_entry.go

- [ ] **Step 5: Commit all herbier models**

```bash
git add internal/domain/herbier/
git commit -m "feat: create herbier domain models (collection, species, entries)"
```

---

### Task 20: Verify All Migrations Apply Successfully

**Files:**
- Test: All migration files from Tasks 1-13

**Interfaces:**
- Consumes: All 13 migration files (002_-014_)
- Produces: Confirmation that migrations apply without errors

- [ ] **Step 1: Run migrations against test database**

Run: `cd /Users/abroudoux/dev/epitech/traily/src/backend && docker-compose up -d postgres && sleep 5`
Expected: Postgres container starts successfully

- [ ] **Step 2: Build and run migration command**

Run: `cd /Users/abroudoux/dev/epitech/traily/src/backend && go build -o bin/migrate ./cmd/migrate && ./bin/migrate`
Expected: All migrations apply without errors; output shows migration names as applied

- [ ] **Step 3: Verify schema in database**

Run: `docker-compose exec postgres psql -U postgres -d traily -c "\dt" 2>/dev/null | grep -E "app_user|track|trace|activity|publication|poi|herbier|plant_species"`
Expected: Lists all 13 tables created

- [ ] **Step 4: Verify PostGIS extension is active**

Run: `docker-compose exec postgres psql -U postgres -d traily -c "SELECT extname FROM pg_extension WHERE extname = 'postgis';" 2>/dev/null`
Expected: Shows `postgis` installed

- [ ] **Step 5: Verify ENUM types exist**

Run: `docker-compose exec postgres psql -U postgres -d traily -c "\dT trace_difficulty poi_type" 2>/dev/null`
Expected: Shows both ENUM types in output

- [ ] **Step 6: Commit verification (no file changes)**

Run: `git status`
Expected: Clean working tree

---

### Task 21: Update User Domain Model to Match app_user Table

**Files:**
- Modify: `internal/domain/user/user.go`

**Interfaces:**
- Consumes: app_user migration from Task 4
- Produces: Updated User struct matching new schema (username, email, display_name, avatar_url)

- [ ] **Step 1: Update the User struct**

Old structure (from start):
```go
type User struct {
	ID        string
	Email     string
	Password  string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}
```

New structure:
```go
type User struct {
	ID          string
	Username    string
	Email       string
	DisplayName *string
	AvatarURL   *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
```

- [ ] **Step 2: Verify password field is removed**

Run: `grep -i password internal/domain/user/user.go`
Expected: No match (password field removed)

- [ ] **Step 3: Check if repositories need updating**

Run: `grep -r "Password" internal/domain/user/ internal/infrastructure/postgres/`
Expected: Lists any references to Password field that need removal

- [ ] **Step 4: Commit**

```bash
git add internal/domain/user/user.go
git commit -m "feat: update user model to match new app_user schema"
```

---

## Verification Checklist

Before marking this plan complete:

1. **All 14 migration files created** (002_ through 014_)
2. **All 8 domain models created** (Track, Trace, Activity, Publication, POI, Herbier, PlantSpecies, HerbierEntry)
3. **Migrations apply cleanly** without errors
4. **All tables exist in database** after migration
5. **PostGIS extension is installed** and working
6. **ENUM types are created** and queryable
7. **All indexes are created** as specified
8. **User domain model updated** to match new schema
9. **Foreign key constraints enforced** (RESTRICT vs CASCADE verified)
10. **All timestamps use TIMESTAMPTZ**

---
