package trail

import (
	"context"
	"errors"
	"math"
	"testing"
)

// fakeRepository records the trails it receives and echoes them back.
type fakeRepository struct {
	created     *Trail
	createCalls int
	listedFor   string
	stored      []Trail
	err         error
}

func (f *fakeRepository) GetByID(_ context.Context, _ string) (Trail, error) {
	if f.err != nil {
		return Trail{}, f.err
	}
	return Trail{ID: "stored"}, nil
}

func (f *fakeRepository) ListByUserID(_ context.Context, userID string) ([]Trail, error) {
	f.listedFor = userID
	if f.err != nil {
		return nil, f.err
	}
	return f.stored, nil
}

func (f *fakeRepository) Create(_ context.Context, t Trail) (Trail, error) {
	f.createCalls++
	if f.err != nil {
		return Trail{}, f.err
	}
	received := t
	f.created = &received
	t.ID = "generated-id"
	return t, nil
}

func validTrail() Trail {
	owner := "user-1"
	return Trail{
		UserID:              &owner,
		Name:                "Tour du lac",
		Difficulty:          DifficultyHard,
		DistanceMeters:      4200,
		ElevationGainMeters: 120,
		ElevationLossMeters: 118,
		Path: []Coordinate{
			{Longitude: 6.1294, Latitude: 45.8992},
			{Longitude: 6.1301, Latitude: 45.9004},
		},
		IsPublished: true,
	}
}

func TestService_Create(t *testing.T) {
	repo := &fakeRepository{}
	svc := NewService(repo)

	created, err := svc.Create(context.Background(), validTrail())
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if created.ID != "generated-id" {
		t.Fatalf("Create() ID = %q, want the repository one", created.ID)
	}
	if repo.created == nil || repo.created.Name != "Tour du lac" || repo.created.Difficulty != DifficultyHard {
		t.Fatalf("repository received %+v", repo.created)
	}
}

func TestService_Create_Defaults(t *testing.T) {
	repo := &fakeRepository{}
	svc := NewService(repo)

	in := validTrail()
	in.Name = "  Tour du lac \n"
	in.Difficulty = ""
	in.IsOfficial = true

	if _, err := svc.Create(context.Background(), in); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if repo.created.Name != "Tour du lac" {
		t.Fatalf("name = %q, want it trimmed", repo.created.Name)
	}
	if repo.created.Difficulty != DifficultyModerate {
		t.Fatalf("difficulty = %q, want %q by default", repo.created.Difficulty, DifficultyModerate)
	}
	if repo.created.IsOfficial {
		t.Fatal("a trail created through the service must never be official")
	}
}

func TestService_Create_Validation(t *testing.T) {
	negative := -1
	empty := ""

	tests := []struct {
		name   string
		mutate func(*Trail)
		want   error
	}{
		{"no owner", func(t *Trail) { t.UserID = nil }, ErrUserRequired},
		{"blank owner", func(t *Trail) { t.UserID = &empty }, ErrUserRequired},
		{"no name", func(t *Trail) { t.Name = "" }, ErrNameRequired},
		{"blank name", func(t *Trail) { t.Name = "   " }, ErrNameRequired},
		{"unknown difficulty", func(t *Trail) { t.Difficulty = "legendary" }, ErrInvalidDifficulty},
		{"negative distance", func(t *Trail) { t.DistanceMeters = -0.1 }, ErrInvalidDistance},
		{"NaN distance", func(t *Trail) { t.DistanceMeters = math.NaN() }, ErrInvalidDistance},
		{"infinite distance", func(t *Trail) { t.DistanceMeters = math.Inf(1) }, ErrInvalidDistance},
		{"negative gain", func(t *Trail) { t.ElevationGainMeters = -1 }, ErrInvalidElevation},
		{"negative loss", func(t *Trail) { t.ElevationLossMeters = -1 }, ErrInvalidElevation},
		{"negative duration", func(t *Trail) { t.EstimatedDurationSeconds = &negative }, ErrInvalidDuration},
		{"no path", func(t *Trail) { t.Path = nil }, ErrPathTooShort},
		{"one point", func(t *Trail) { t.Path = t.Path[:1] }, ErrPathTooShort},
		{"longitude too high", func(t *Trail) { t.Path[1].Longitude = 180.01 }, ErrInvalidCoordinate},
		{"longitude too low", func(t *Trail) { t.Path[0].Longitude = -181 }, ErrInvalidCoordinate},
		{"latitude too high", func(t *Trail) { t.Path[1].Latitude = 90.5 }, ErrInvalidCoordinate},
		{"latitude too low", func(t *Trail) { t.Path[0].Latitude = -91 }, ErrInvalidCoordinate},
		{"NaN coordinate", func(t *Trail) { t.Path[0].Latitude = math.NaN() }, ErrInvalidCoordinate},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepository{}
			in := validTrail()
			tt.mutate(&in)

			_, err := NewService(repo).Create(context.Background(), in)

			if !errors.Is(err, tt.want) {
				t.Fatalf("Create() error = %v, want %v", err, tt.want)
			}
			if repo.createCalls != 0 {
				t.Fatal("an invalid trail must not reach the repository")
			}
		})
	}
}

func TestService_Create_AcceptsBoundaryValues(t *testing.T) {
	in := validTrail()
	in.DistanceMeters, in.ElevationGainMeters, in.ElevationLossMeters = 0, 0, 0
	in.Path = []Coordinate{{Longitude: -180, Latitude: -90}, {Longitude: 180, Latitude: 90}}
	zero := 0
	in.EstimatedDurationSeconds = &zero

	if _, err := NewService(&fakeRepository{}).Create(context.Background(), in); err != nil {
		t.Fatalf("Create() error = %v, want boundary values accepted", err)
	}
}

func TestService_Create_RepositoryError(t *testing.T) {
	boom := errors.New("boom")

	_, err := NewService(&fakeRepository{err: boom}).Create(context.Background(), validTrail())

	if !errors.Is(err, boom) {
		t.Fatalf("Create() error = %v, want the repository error", err)
	}
}

func TestService_ListByUser(t *testing.T) {
	repo := &fakeRepository{stored: []Trail{{ID: "a"}, {ID: "b"}}}

	trails, err := NewService(repo).ListByUser(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListByUser() error = %v", err)
	}

	if repo.listedFor != "user-1" || len(trails) != 2 {
		t.Fatalf("ListByUser() queried %q and returned %d trails", repo.listedFor, len(trails))
	}
}

func TestService_ListByUser_RequiresUser(t *testing.T) {
	repo := &fakeRepository{}

	if _, err := NewService(repo).ListByUser(context.Background(), ""); !errors.Is(err, ErrUserRequired) {
		t.Fatalf("ListByUser() error = %v, want %v", err, ErrUserRequired)
	}
	if repo.listedFor != "" {
		t.Fatal("the repository must not be queried without a user")
	}
}

func TestService_Get(t *testing.T) {
	got, err := NewService(&fakeRepository{}).Get(context.Background(), "stored")
	if err != nil || got.ID != "stored" {
		t.Fatalf("Get() = %+v, %v", got, err)
	}

	if _, err := NewService(&fakeRepository{err: ErrNotFound}).Get(context.Background(), "x"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get() error = %v, want %v", err, ErrNotFound)
	}
}
