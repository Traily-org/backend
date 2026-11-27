package ban

import "time"

// IdentifierType is the kind of a banned identifier.
type IdentifierType string

// IdentifierEmail is a banned email address, hashed with HashEmail.
const IdentifierEmail IdentifierType = "email"

// Identifier is a hashed value that cannot be used to sign up while it is
// banned. It outlives the account it was taken from.
type Identifier struct {
	Type      IdentifierType
	ValueHash string
	// ExpiresAt is copied from the ban; nil means permanent.
	ExpiresAt *time.Time
}
