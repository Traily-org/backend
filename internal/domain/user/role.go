package user

// Role is the permission level of a user, stored in users.role.
type Role string

// Roles allowed by the users_role_valid constraint.
const (
	RoleUser      Role = "user"
	RoleModerator Role = "moderator"
	RoleAdmin     Role = "admin"
)
