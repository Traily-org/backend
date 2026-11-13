package report

import "context"

// Repository persists reports.
type Repository interface {
	// TargetAuthor returns the author of the target as seen by reporterID: a
	// nil author means the target exists but has none. It returns
	// ErrTargetNotFound when the target does not exist or is not visible to
	// reporterID (private or unpublished content of someone else).
	TargetAuthor(ctx context.Context, kind TargetType, id, reporterID string) (*string, error)
	// Create stores r with status pending and returns it, or
	// ErrAlreadyReported when reporterID already has a pending report on the
	// same target.
	Create(ctx context.Context, r Report) (Report, error)
	// List returns the reports matching f, newest first.
	List(ctx context.Context, f Filter) (Page, error)
}
