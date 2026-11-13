package report

import (
	"context"
	"strings"
	"unicode/utf8"
)

// Service creates and lists reports.
type Service struct {
	repo Repository
}

// NewService returns a Service backed by repo.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Create records a report from in.ReporterID about the target. The target
// must exist and be visible to the reporter, must not be the reporter or
// authored by them, and the reporter must not already have a pending report
// on it. The report is stored synchronously, so it shows up in List as soon
// as Create returns.
func (s *Service) Create(ctx context.Context, in Input) (Report, error) {
	reason := strings.TrimSpace(in.Reason)
	if reason == "" {
		return Report{}, ErrReasonRequired
	}
	if utf8.RuneCountInString(reason) > MaxReasonLength {
		return Report{}, ErrReasonTooLong
	}
	if !in.TargetType.Valid() {
		return Report{}, ErrInvalidTargetType
	}

	author, err := s.repo.TargetAuthor(ctx, in.TargetType, in.TargetID, in.ReporterID)
	if err != nil {
		return Report{}, err
	}
	if author != nil && *author == in.ReporterID {
		return Report{}, ErrCannotReportSelf
	}

	return s.repo.Create(ctx, Report{
		ReporterID: in.ReporterID,
		TargetType: in.TargetType,
		TargetID:   in.TargetID,
		Reason:     reason,
		Status:     StatusPending,
	})
}

// List returns a page of reports, newest first. An out-of-range limit is
// replaced by DefaultLimit or MaxLimit, a negative offset by 0.
func (s *Service) List(ctx context.Context, f Filter) (Page, error) {
	if f.Status != nil && !f.Status.Valid() {
		return Page{}, ErrInvalidStatus
	}
	if f.TargetType != nil && !f.TargetType.Valid() {
		return Page{}, ErrInvalidTargetType
	}
	switch {
	case f.Limit <= 0:
		f.Limit = DefaultLimit
	case f.Limit > MaxLimit:
		f.Limit = MaxLimit
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	return s.repo.List(ctx, f)
}
