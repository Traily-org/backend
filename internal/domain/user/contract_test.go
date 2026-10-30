package user_test

import (
	"github.com/traily-org/server/internal/auth"
	"github.com/traily-org/server/internal/domain/user"
)

// Service must satisfy the user store auth relies on.
var _ auth.UserStore = (*user.Service)(nil)
