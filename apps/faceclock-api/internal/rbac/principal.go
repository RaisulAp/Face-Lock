package rbac

import (
	"context"

	"github.com/google/uuid"
)

type contextKey struct{ name string }

var principalContextKey = &contextKey{name: "faceclock_principal"}

// Principal represents the authenticated actor making the current HTTP request.
type Principal struct {
	UserID       uuid.UUID
	EmployeeID   *uuid.UUID
	Email        string
	TokenVersion int
	Roles        []string
	Permissions  map[string]struct{}
}

// HasRole checks whether the principal has the specified role name.
func (p *Principal) HasRole(role string) bool {
	for _, r := range p.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// IsSuperAdmin returns true if the principal has the super_admin role.
func (p *Principal) IsSuperAdmin() bool {
	return p.HasRole("super_admin")
}

// HasPermission returns true if the principal has the given permission (or is super_admin).
func (p *Principal) HasPermission(permission string) bool {
	if p.IsSuperAdmin() {
		return true
	}
	_, ok := p.Permissions[permission]
	return ok
}

// HasAnyPermission returns true if principal possesses at least one of the listed permissions.
func (p *Principal) HasAnyPermission(permissions ...string) bool {
	if p.IsSuperAdmin() {
		return true
	}
	for _, perm := range permissions {
		if _, ok := p.Permissions[perm]; ok {
			return true
		}
	}
	return false
}

// HasAllPermissions returns true if principal possesses every listed permission.
func (p *Principal) HasAllPermissions(permissions ...string) bool {
	if p.IsSuperAdmin() {
		return true
	}
	for _, perm := range permissions {
		if _, ok := p.Permissions[perm]; !ok {
			return false
		}
	}
	return true
}

// WithPrincipal adds the Principal to the context.
func WithPrincipal(ctx context.Context, p *Principal) context.Context {
	return context.WithValue(ctx, principalContextKey, p)
}

// GetPrincipal extracts the Principal from context if present.
func GetPrincipal(ctx context.Context) (*Principal, bool) {
	p, ok := ctx.Value(principalContextKey).(*Principal)
	return p, ok && p != nil
}
