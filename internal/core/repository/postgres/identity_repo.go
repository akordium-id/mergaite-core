package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/akordium-id/mergiate-core/internal/core/domain/identity"
	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
	"github.com/akordium-id/mergiate-core/internal/core/repository/postgres/sqlc"
)

type identityRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

// NewIdentityRepository creates a new PostgreSQL Identity & Access repository.
func NewIdentityRepository(pool *pgxpool.Pool) identity.Repository {
	return &identityRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

// ----------------------------------------------------------------------------
// User Operations
// ----------------------------------------------------------------------------

func (r *identityRepository) CreateUser(ctx context.Context, u *identity.User) error {
	params := sqlc.CreateUserParams{
		ID:           shared.ToPgUUID(u.ID),
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Name:         u.Name,
		Status:       string(u.Status),
		CreatedAt:    pgtype.Timestamptz{Time: u.CreatedAt, Valid: true},
		UpdatedAt:    pgtype.Timestamptz{Time: u.UpdatedAt, Valid: true},
	}

	row, err := r.queries.CreateUser(ctx, params)
	if err != nil {
		return err
	}

	u.ID = shared.FromPgUUID(row.ID)
	u.CreatedAt = row.CreatedAt.Time
	u.UpdatedAt = row.UpdatedAt.Time
	return nil
}

func (r *identityRepository) GetUserByEmail(ctx context.Context, email string) (*identity.User, error) {
	row, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}

	return &identity.User{
		ID:           shared.FromPgUUID(row.ID),
		Email:        row.Email,
		PasswordHash: row.PasswordHash,
		Name:         row.Name,
		Status:       identity.UserStatus(row.Status),
		CreatedAt:    row.CreatedAt.Time,
		UpdatedAt:    row.UpdatedAt.Time,
	}, nil
}

func (r *identityRepository) GetUserByID(ctx context.Context, id shared.ID) (*identity.User, error) {
	row, err := r.queries.GetUserByID(ctx, shared.ToPgUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}

	return &identity.User{
		ID:           shared.FromPgUUID(row.ID),
		Email:        row.Email,
		PasswordHash: row.PasswordHash,
		Name:         row.Name,
		Status:       identity.UserStatus(row.Status),
		CreatedAt:    row.CreatedAt.Time,
		UpdatedAt:    row.UpdatedAt.Time,
	}, nil
}

// ----------------------------------------------------------------------------
// Tenant Membership Operations
// ----------------------------------------------------------------------------

func (r *identityRepository) AddTenantMember(ctx context.Context, member *identity.TenantUser) error {
	params := sqlc.AddTenantMemberParams{
		ID:       shared.ToPgUUID(member.ID),
		TenantID: shared.ToPgUUID(member.TenantID),
		UserID:   shared.ToPgUUID(member.UserID),
		Status:   string(member.Status),
		JoinedAt: pgtype.Timestamptz{Time: member.JoinedAt, Valid: true},
	}

	row, err := r.queries.AddTenantMember(ctx, params)
	if err != nil {
		return err
	}

	member.ID = shared.FromPgUUID(row.ID)
	member.JoinedAt = row.JoinedAt.Time
	return nil
}

func (r *identityRepository) GetTenantMember(ctx context.Context, tenantID, userID shared.ID) (*identity.TenantUser, error) {
	row, err := r.queries.GetTenantMember(ctx, sqlc.GetTenantMemberParams{
		TenantID: shared.ToPgUUID(tenantID),
		UserID:   shared.ToPgUUID(userID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}

	return &identity.TenantUser{
		ID:       shared.FromPgUUID(row.ID),
		TenantID: shared.FromPgUUID(row.TenantID),
		UserID:   shared.FromPgUUID(row.UserID),
		Status:   identity.MembershipStatus(row.Status),
		JoinedAt: row.JoinedAt.Time,
	}, nil
}

func (r *identityRepository) ListUserTenants(ctx context.Context, userID shared.ID) ([]identity.TenantMembership, error) {
	rows, err := r.queries.ListUserTenants(ctx, shared.ToPgUUID(userID))
	if err != nil {
		return nil, err
	}

	memberships := make([]identity.TenantMembership, len(rows))
	for i, row := range rows {
		memberships[i] = identity.TenantMembership{
			TenantID:         shared.FromPgUUID(row.TenantID),
			TenantCode:       row.TenantCode,
			TenantName:       row.TenantName,
			TenantStatus:     row.TenantStatus,
			MembershipStatus: identity.MembershipStatus(row.MembershipStatus),
			JoinedAt:         row.JoinedAt.Time,
		}
	}
	return memberships, nil
}

func (r *identityRepository) ListTenantUsers(ctx context.Context, tenantID shared.ID) ([]identity.TenantUserItem, error) {
	rows, err := r.queries.ListTenantUsers(ctx, shared.ToPgUUID(tenantID))
	if err != nil {
		return nil, err
	}

	users := make([]identity.TenantUserItem, len(rows))
	for i, row := range rows {
		uID := shared.FromPgUUID(row.UserID)
		roles, _ := r.ListUserRolesInTenant(ctx, tenantID, uID)

		users[i] = identity.TenantUserItem{
			UserID:           uID,
			Email:            row.Email,
			Name:             row.Name,
			UserStatus:       identity.UserStatus(row.UserStatus),
			MembershipStatus: identity.MembershipStatus(row.MembershipStatus),
			JoinedAt:         row.JoinedAt.Time,
			Roles:            roles,
		}
	}
	return users, nil
}

// ----------------------------------------------------------------------------
// Role Operations
// ----------------------------------------------------------------------------

func (r *identityRepository) CreateRole(ctx context.Context, role *identity.Role) error {
	var desc *string
	if role.Description != "" {
		desc = &role.Description
	}

	params := sqlc.CreateRoleParams{
		ID:          shared.ToPgUUID(role.ID),
		TenantID:    shared.ToPgUUID(role.TenantID),
		Code:        role.Code,
		Name:        role.Name,
		Description: desc,
		IsSystem:    role.IsSystem,
		CreatedAt:   pgtype.Timestamptz{Time: role.CreatedAt, Valid: true},
		UpdatedAt:   pgtype.Timestamptz{Time: role.UpdatedAt, Valid: true},
	}

	row, err := r.queries.CreateRole(ctx, params)
	if err != nil {
		return err
	}

	role.ID = shared.FromPgUUID(row.ID)
	role.CreatedAt = row.CreatedAt.Time
	role.UpdatedAt = row.UpdatedAt.Time
	return nil
}

func (r *identityRepository) GetRoleByID(ctx context.Context, tenantID, id shared.ID) (*identity.Role, error) {
	row, err := r.queries.GetRoleByID(ctx, sqlc.GetRoleByIDParams{
		TenantID: shared.ToPgUUID(tenantID),
		ID:       shared.ToPgUUID(id),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}

	role := &identity.Role{
		ID:          shared.FromPgUUID(row.ID),
		TenantID:    shared.FromPgUUID(row.TenantID),
		Code:        row.Code,
		Name:        row.Name,
		Description: strFromPtr(row.Description),
		IsSystem:    row.IsSystem,
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
	}

	perms, err := r.ListRolePermissions(ctx, role.ID)
	if err == nil {
		role.Permissions = perms
	}

	return role, nil
}

func (r *identityRepository) GetRoleByCode(ctx context.Context, tenantID shared.ID, code string) (*identity.Role, error) {
	row, err := r.queries.GetRoleByCode(ctx, sqlc.GetRoleByCodeParams{
		TenantID: shared.ToPgUUID(tenantID),
		Code:     code,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}

	role := &identity.Role{
		ID:          shared.FromPgUUID(row.ID),
		TenantID:    shared.FromPgUUID(row.TenantID),
		Code:        row.Code,
		Name:        row.Name,
		Description: strFromPtr(row.Description),
		IsSystem:    row.IsSystem,
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
	}

	perms, err := r.ListRolePermissions(ctx, role.ID)
	if err == nil {
		role.Permissions = perms
	}

	return role, nil
}

func (r *identityRepository) ListRoles(ctx context.Context, tenantID shared.ID) ([]identity.Role, error) {
	rows, err := r.queries.ListRoles(ctx, shared.ToPgUUID(tenantID))
	if err != nil {
		return nil, err
	}

	roles := make([]identity.Role, len(rows))
	for i, row := range rows {
		roleID := shared.FromPgUUID(row.ID)
		perms, _ := r.ListRolePermissions(ctx, roleID)

		roles[i] = identity.Role{
			ID:          roleID,
			TenantID:    shared.FromPgUUID(row.TenantID),
			Code:        row.Code,
			Name:        row.Name,
			Description: strFromPtr(row.Description),
			IsSystem:    row.IsSystem,
			Permissions: perms,
			CreatedAt:   row.CreatedAt.Time,
			UpdatedAt:   row.UpdatedAt.Time,
		}
	}
	return roles, nil
}

// ----------------------------------------------------------------------------
// Permission Operations
// ----------------------------------------------------------------------------

func (r *identityRepository) ListPermissions(ctx context.Context) ([]identity.Permission, error) {
	rows, err := r.queries.ListPermissions(ctx)
	if err != nil {
		return nil, err
	}

	perms := make([]identity.Permission, len(rows))
	for i, row := range rows {
		perms[i] = identity.Permission{
			ID:          shared.FromPgUUID(row.ID),
			Code:        row.Code,
			Name:        row.Name,
			Category:    row.Category,
			Description: strFromPtr(row.Description),
		}
	}
	return perms, nil
}

func (r *identityRepository) AssignPermissionsToRole(ctx context.Context, roleID shared.ID, permissionIDs []shared.ID) error {
	_ = r.queries.ClearRolePermissions(ctx, shared.ToPgUUID(roleID))

	for _, pID := range permissionIDs {
		err := r.queries.AssignPermissionToRole(ctx, sqlc.AssignPermissionToRoleParams{
			RoleID:       shared.ToPgUUID(roleID),
			PermissionID: shared.ToPgUUID(pID),
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *identityRepository) ListRolePermissions(ctx context.Context, roleID shared.ID) ([]identity.Permission, error) {
	rows, err := r.queries.ListRolePermissions(ctx, shared.ToPgUUID(roleID))
	if err != nil {
		return nil, err
	}

	perms := make([]identity.Permission, len(rows))
	for i, row := range rows {
		perms[i] = identity.Permission{
			ID:          shared.FromPgUUID(row.ID),
			Code:        row.Code,
			Name:        row.Name,
			Category:    row.Category,
			Description: strFromPtr(row.Description),
		}
	}
	return perms, nil
}

// ----------------------------------------------------------------------------
// User Role Operations
// ----------------------------------------------------------------------------

func (r *identityRepository) AssignUserRole(ctx context.Context, tenantID, userID, roleID shared.ID) error {
	return r.queries.AssignUserRole(ctx, sqlc.AssignUserRoleParams{
		TenantID: shared.ToPgUUID(tenantID),
		UserID:   shared.ToPgUUID(userID),
		RoleID:   shared.ToPgUUID(roleID),
	})
}

func (r *identityRepository) RemoveUserRole(ctx context.Context, tenantID, userID, roleID shared.ID) error {
	return r.queries.RemoveUserRole(ctx, sqlc.RemoveUserRoleParams{
		TenantID: shared.ToPgUUID(tenantID),
		UserID:   shared.ToPgUUID(userID),
		RoleID:   shared.ToPgUUID(roleID),
	})
}

func (r *identityRepository) ListUserRolesInTenant(ctx context.Context, tenantID, userID shared.ID) ([]identity.Role, error) {
	rows, err := r.queries.ListUserRolesInTenant(ctx, sqlc.ListUserRolesInTenantParams{
		TenantID: shared.ToPgUUID(tenantID),
		UserID:   shared.ToPgUUID(userID),
	})
	if err != nil {
		return nil, err
	}

	roles := make([]identity.Role, len(rows))
	for i, row := range rows {
		roles[i] = identity.Role{
			ID:          shared.FromPgUUID(row.ID),
			TenantID:    tenantID,
			Code:        row.Code,
			Name:        row.Name,
			Description: strFromPtr(row.Description),
			IsSystem:    row.IsSystem,
		}
	}
	return roles, nil
}

func (r *identityRepository) ListUserPermissionsInTenant(ctx context.Context, tenantID, userID shared.ID) ([]string, error) {
	return r.queries.ListUserPermissionsInTenant(ctx, sqlc.ListUserPermissionsInTenantParams{
		TenantID: shared.ToPgUUID(tenantID),
		UserID:   shared.ToPgUUID(userID),
	})
}
