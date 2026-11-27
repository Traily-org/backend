package report

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// fakeRepo is an in-memory Repository. authors maps "type:id" to the author of
// the target; a missing key means the target is not found.
type fakeRepo struct {
	authors map[string]*string
	reports []Report
}

func strPtr(s string) *string { return &s }

func newFakeRepo() *fakeRepo {
	return &fakeRepo{authors: map[string]*string{
		"photo:photo-1":    strPtr("author-1"),
		"photo:own-photo":  strPtr("reporter-1"),
		"trail:trail-anon": nil,
		"user:user-2":      strPtr("user-2"),
		"user:reporter-1":  strPtr("reporter-1"),
	}}
}

func (f *fakeRepo) TargetAuthor(_ context.Context, kind TargetType, id, _ string) (*string, error) {
	author, ok := f.authors[string(kind)+":"+id]
	if !ok {
		return nil, ErrTargetNotFound
	}
	return author, nil
}

func (f *fakeRepo) Create(_ context.Context, r Report) (Report, error) {
	for _, existing := range f.reports {
		if existing.ReporterID == r.ReporterID && existing.TargetType == r.TargetType &&
			existing.TargetID == r.TargetID && existing.Status == StatusPending {
			return Report{}, ErrAlreadyReported
		}
	}
	r.ID = "report-" + string(rune('a'+len(f.reports)))
	r.CreatedAt = time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	f.reports = append(f.reports, r)
	return r, nil
}

func (f *fakeRepo) List(_ context.Context, flt Filter) (Page, error) {
	var items []Report
	for _, r := range f.reports {
		if flt.Status != nil && r.Status != *flt.Status {
			continue
		}
		if flt.TargetType != nil && r.TargetType != *flt.TargetType {
			continue
		}
		items = append(items, r)
	}
	return Page{Items: items, Total: len(items), Limit: flt.Limit, Offset: flt.Offset}, nil
}

func TestService_Create_StoresTraceableReport(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)

	got, err := svc.Create(context.Background(), Input{
		ReporterID: "reporter-1",
		TargetType: TargetPhoto,
		TargetID:   "photo-1",
		Reason:     "  inappropriate content  ",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if got.ID == "" || got.CreatedAt.IsZero() {
		t.Errorf("Create() = %+v, want ID and CreatedAt set", got)
	}
	if got.ReporterID != "reporter-1" || got.TargetType != TargetPhoto || got.TargetID != "photo-1" {
		t.Errorf("Create() = %+v, want reporter, target type and target id kept", got)
	}
	if got.Reason != "inappropriate content" {
		t.Errorf("Reason = %q, want trimmed", got.Reason)
	}
	if got.Status != StatusPending {
		t.Errorf("Status = %q, want pending", got.Status)
	}
}

func TestService_Create_Errors(t *testing.T) {
	tests := []struct {
		name string
		in   Input
		want error
	}{
		{"blank reason", Input{ReporterID: "reporter-1", TargetType: TargetPhoto, TargetID: "photo-1", Reason: "   "}, ErrReasonRequired},
		{"reason too long", Input{ReporterID: "reporter-1", TargetType: TargetPhoto, TargetID: "photo-1", Reason: strings.Repeat("é", MaxReasonLength+1)}, ErrReasonTooLong},
		{"unknown target type", Input{ReporterID: "reporter-1", TargetType: "comment", TargetID: "photo-1", Reason: "spam"}, ErrInvalidTargetType},
		{"target not found", Input{ReporterID: "reporter-1", TargetType: TargetPhoto, TargetID: "missing", Reason: "spam"}, ErrTargetNotFound},
		{"own content", Input{ReporterID: "reporter-1", TargetType: TargetPhoto, TargetID: "own-photo", Reason: "spam"}, ErrCannotReportSelf},
		{"self", Input{ReporterID: "reporter-1", TargetType: TargetUser, TargetID: "reporter-1", Reason: "spam"}, ErrCannotReportSelf},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			_, err := NewService(repo).Create(context.Background(), tt.in)
			if !errors.Is(err, tt.want) {
				t.Errorf("Create() error = %v, want %v", err, tt.want)
			}
			if len(repo.reports) != 0 {
				t.Errorf("stored %d reports, want none", len(repo.reports))
			}
		})
	}
}

func TestService_Create_ReasonAtLimitIsAccepted(t *testing.T) {
	_, err := NewService(newFakeRepo()).Create(context.Background(), Input{
		ReporterID: "reporter-1", TargetType: TargetPhoto, TargetID: "photo-1",
		Reason: strings.Repeat("é", MaxReasonLength),
	})
	if err != nil {
		t.Errorf("Create() error = %v, want nil", err)
	}
}

func TestService_Create_TargetWithoutAuthorCanBeReported(t *testing.T) {
	_, err := NewService(newFakeRepo()).Create(context.Background(), Input{
		ReporterID: "reporter-1", TargetType: TargetTrail, TargetID: "trail-anon", Reason: "wrong path",
	})
	if err != nil {
		t.Errorf("Create() error = %v, want nil", err)
	}
}

func TestService_Create_DuplicatePendingReport(t *testing.T) {
	svc := NewService(newFakeRepo())
	in := Input{ReporterID: "reporter-1", TargetType: TargetUser, TargetID: "user-2", Reason: "harassment"}

	if _, err := svc.Create(context.Background(), in); err != nil {
		t.Fatalf("first Create() error = %v", err)
	}
	if _, err := svc.Create(context.Background(), in); !errors.Is(err, ErrAlreadyReported) {
		t.Errorf("second Create() error = %v, want ErrAlreadyReported", err)
	}
	in.ReporterID = "reporter-2"
	if _, err := svc.Create(context.Background(), in); err != nil {
		t.Errorf("Create() by another reporter error = %v, want nil", err)
	}
}

func TestService_List_AppliesDefaultsAndBounds(t *testing.T) {
	tests := []struct {
		name       string
		in         Filter
		wantLimit  int
		wantOffset int
	}{
		{"defaults", Filter{}, DefaultLimit, 0},
		{"too large", Filter{Limit: 5000, Offset: 10}, MaxLimit, 10},
		{"negative", Filter{Limit: -3, Offset: -4}, DefaultLimit, 0},
		{"kept", Filter{Limit: 30, Offset: 60}, 30, 60},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page, err := NewService(newFakeRepo()).List(context.Background(), tt.in)
			if err != nil {
				t.Fatalf("List() error = %v", err)
			}
			if page.Limit != tt.wantLimit || page.Offset != tt.wantOffset {
				t.Errorf("List() limit/offset = %d/%d, want %d/%d", page.Limit, page.Offset, tt.wantLimit, tt.wantOffset)
			}
		})
	}
}

func TestService_List_RejectsUnknownFilters(t *testing.T) {
	svc := NewService(newFakeRepo())
	badStatus := Status("open")
	badType := TargetType("comment")

	if _, err := svc.List(context.Background(), Filter{Status: &badStatus}); !errors.Is(err, ErrInvalidStatus) {
		t.Errorf("List() error = %v, want ErrInvalidStatus", err)
	}
	if _, err := svc.List(context.Background(), Filter{TargetType: &badType}); !errors.Is(err, ErrInvalidTargetType) {
		t.Errorf("List() error = %v, want ErrInvalidTargetType", err)
	}
}

func TestService_List_ShowsNewReportImmediately(t *testing.T) {
	svc := NewService(newFakeRepo())
	if _, err := svc.Create(context.Background(), Input{ReporterID: "reporter-1", TargetType: TargetPhoto, TargetID: "photo-1", Reason: "nsfw"}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	pending := StatusPending
	page, err := svc.List(context.Background(), Filter{Status: &pending})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].TargetID != "photo-1" {
		t.Errorf("List() = %+v, want the new report", page)
	}
}
