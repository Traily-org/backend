package ban

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/traily-org/server/internal/domain/user"
)

// UserGetter is the subset of the user service the ban service relies on.
type UserGetter interface {
	Get(ctx context.Context, id string) (user.User, error)
}

// Moderator is the authenticated user performing a moderation action.
type Moderator struct {
	ID   string
	Role user.Role
}

// Input describes a ban to create.
type Input struct {
	UserID string
	// ReportID is the report that justifies the ban. It must concern the
	// user, directly or through content they authored.
	ReportID string
	Reason   string
	// ExpiresAt is the end of the ban; nil makes it permanent.
	ExpiresAt *time.Time
	By        Moderator
}

// Service bans accounts and tells whether an account or an email address is
// banned.
type Service struct {
	repo     Repository
	users    UserGetter
	sessions user.SessionRevoker
	now      func() time.Time
}

// NewService returns a Service backed by repo.
func NewService(repo Repository, users UserGetter) *Service {
	return &Service{repo: repo, users: users, now: time.Now}
}

// SetSessionRevoker registers the revoker called after a ban, so that the
// banned user cannot refresh their tokens. It is set after construction
// because the auth service itself depends on the ban service.
func (s *Service) SetSessionRevoker(r user.SessionRevoker) {
	s.sessions = r
}

// Ban bans in.UserID until in.ExpiresAt, or permanently, and bans their
// email address for the same duration. The ban must come from a report that
// concerns the user and was not dismissed; the report is marked as reviewed.
// Every session of the user is revoked. A moderator can only ban users with
// the user role; an admin can ban anyone but themselves. A user already
// banned gets one more ban; the one ending last wins.
func (s *Service) Ban(ctx context.Context, in Input) (Ban, error) {
	reason := strings.TrimSpace(in.Reason)
	if reason == "" {
		return Ban{}, ErrReasonRequired
	}
	if in.ReportID == "" {
		return Ban{}, ErrReportRequired
	}
	if in.UserID == in.By.ID {
		return Ban{}, ErrCannotBanSelf
	}
	now := s.now()
	if in.ExpiresAt != nil && !in.ExpiresAt.After(now) {
		return Ban{}, ErrExpiryNotInFuture
	}

	target, err := s.users.Get(ctx, in.UserID)
	if err != nil {
		return Ban{}, err
	}
	if target.Role != user.RoleUser && in.By.Role != user.RoleAdmin {
		return Ban{}, ErrCannotBanStaff
	}

	report, err := s.repo.Report(ctx, in.ReportID)
	if err != nil {
		return Ban{}, err
	}
	if report.TargetUserID == nil || *report.TargetUserID != in.UserID {
		return Ban{}, ErrReportNotAboutUser
	}
	if report.Dismissed {
		return Ban{}, ErrReportDismissed
	}

	by := in.By.ID
	reportID := report.ID
	created, err := s.repo.Create(ctx, Ban{
		UserID:    in.UserID,
		ReportID:  &reportID,
		Reason:    reason,
		BannedBy:  &by,
		ExpiresAt: in.ExpiresAt,
	}, []Identifier{{
		Type:      IdentifierEmail,
		ValueHash: HashEmail(target.Email),
		ExpiresAt: in.ExpiresAt,
	}})
	if err != nil {
		return Ban{}, err
	}

	if s.sessions != nil {
		if err := s.sessions.RevokeAllSessions(ctx, in.UserID); err != nil {
			return Ban{}, fmt.Errorf("revoke sessions after ban: %w", err)
		}
	}
	return created, nil
}

// Lift ends every active ban of userID and unbans their email address. It
// returns ErrNotBanned when the user has no active ban.
func (s *Service) Lift(ctx context.Context, userID string, by Moderator) error {
	n, err := s.repo.Lift(ctx, userID, by.ID, s.now())
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotBanned
	}
	return nil
}

// Active returns the active ban of userID that ends last, or ErrNotBanned.
func (s *Service) Active(ctx context.Context, userID string) (Ban, error) {
	return s.repo.Active(ctx, userID, s.now())
}

// CheckUser returns an ACCOUNT_BANNED error, wrapping ErrBanned, when userID
// has an active ban, and nil otherwise.
func (s *Service) CheckUser(ctx context.Context, userID string) error {
	b, err := s.repo.Active(ctx, userID, s.now())
	if errors.Is(err, ErrNotBanned) {
		return nil
	}
	if err != nil {
		return err
	}
	return bannedAccountError(b)
}

// CheckEmail returns an ACCOUNT_BANNED error, wrapping ErrBanned, when email
// or another spelling of it (see NormalizeEmail) is banned, and nil otherwise.
func (s *Service) CheckEmail(ctx context.Context, email string) error {
	banned, err := s.repo.IdentifierBanned(ctx, IdentifierEmail, HashEmail(email), s.now())
	if err != nil {
		return err
	}
	if banned {
		return errBannedEmail
	}
	return nil
}
