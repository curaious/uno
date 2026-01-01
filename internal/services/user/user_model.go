package user

import "time"

type UserRole string

const (
	RoleSuperAdmin    UserRole = "super-admin"
	RoleProjectAdmin  UserRole = "project-admin"
	RoleProjectMember UserRole = "project-member"
	RoleProjectViewer UserRole = "project-viewer"
)

type User struct {
	ID                  string    `db:"id" json:"id"`
	Name                string    `db:"name" json:"name"`
	Email               string    `db:"email" json:"email"`
	PasswordHash        string    `db:"password_hash" json:"-"`
	PasswordAuthEnabled bool      `db:"password_auth_enabled" json:"password_auth_enabled"`
	Role                UserRole  `db:"role" json:"role"`
	CreatedAt           time.Time `db:"created_at" json:"created_at"`
	UpdatedAt           time.Time `db:"updated_at" json:"updated_at"`
}
