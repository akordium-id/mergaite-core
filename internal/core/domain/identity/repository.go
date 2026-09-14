package identity

import (
	"context"

	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
)

// Repository defines data access operations for Identity & RBAC.
type Repository interface {
	// User operations
	CreateUser(ctx context.Context, u *User) error
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id shared.ID) (*User, error)

	// Tenant membership operations
	AddTenantMember(ctx context.Context, member *TenantUser) error
	GetTenantMember(ctx context.Context, tenantID, userID shared.ID) (*TenantUser, error)
	ListUserTenants(ctx context.Context, userID shared.ID) ([]TenantMembership, error)
	ListTenantUsers(ctx context.Context, tenantID shared.ID) ([]TenantUserItem, error)

	// Role operations
	CreateRole(ctx context.Context, r *Role) error
	GetRoleByID(ctx context.Context, tenantID, id shared.ID) (*Role, error)
	GetRoleByCode(ctx context.Context, tenantID shared.ID, code string) (*Role, error)
	ListRoles(ctx context.Context, tenantID shared.ID) ([]Role, error)

	// Permission operations
	ListPermissions(ctx context.Context) ([]Permission, error)
	AssignPermissionsToRole(ctx context.Context, roleID shared.ID, permissionIDs []shared.ID) error
	ListRolePermissions(ctx context.Context, roleID shared.ID) ([]Permission, error)

	// User Role Assignment
	AssignUserRole(ctx context.Context, tenantID, userID, roleID shared.ID) error
	RemoveUserRole(ctx context.Context, tenantID, userID, roleID shared.ID) error
	ListUserRolesInTenant(ctx context.Context, tenantID, userID shared.ID) ([]Role, error)
	ListUserPermissionsInTenant(ctx context.Context, tenantID, userID shared.ID) ([]string, error)
}
