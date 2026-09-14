package identity

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/akordium-id/mergiate-core/internal/core/domain/identity"
	"github.com/akordium-id/mergiate-core/internal/core/domain/repository"
	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
	"github.com/akordium-id/mergiate-core/pkg/auth"
)

// Commands and Results
type RegisterCommand struct {
	Email    string     `json:"email"`
	Password string     `json:"password"`
	Name     string     `json:"name"`
	TenantID *shared.ID `json:"tenant_id,omitempty"`
}

type LoginCommand struct {
	Email    string     `json:"email"`
	Password string     `json:"password"`
	TenantID *shared.ID `json:"tenant_id,omitempty"`
}

type LoginResult struct {
	User                    *identity.User              `json:"user"`
	Token                   string                      `json:"token,omitempty"`
	ExpiresIn               int64                       `json:"expires_in,omitempty"`
	TenantID                *shared.ID                  `json:"tenant_id,omitempty"`
	Roles                   []string                    `json:"roles,omitempty"`
	Permissions             []string                    `json:"permissions,omitempty"`
	RequiresTenantSelection bool                        `json:"requires_tenant_selection,omitempty"`
	AvailableTenants        []identity.TenantMembership `json:"available_tenants,omitempty"`
}

type SwitchTenantCommand struct {
	UserID   shared.ID `json:"user_id"`
	TenantID shared.ID `json:"tenant_id"`
}

type UserProfileResult struct {
	User    *identity.User              `json:"user"`
	Tenants []identity.TenantMembership `json:"tenants"`
}

type AddMemberCommand struct {
	TenantID shared.ID   `json:"tenant_id"`
	UserID   shared.ID   `json:"user_id"`
	RoleIDs  []shared.ID `json:"role_ids,omitempty"`
}

type AssignMemberRolesCommand struct {
	TenantID shared.ID   `json:"tenant_id"`
	UserID   shared.ID   `json:"user_id"`
	RoleIDs  []shared.ID `json:"role_ids"`
}

type CreateRoleCommand struct {
	TenantID      shared.ID   `json:"tenant_id"`
	Code          string      `json:"code"`
	Name          string      `json:"name"`
	Description   string      `json:"description,omitempty"`
	PermissionIDs []shared.ID `json:"permission_ids,omitempty"`
}

type AssignRolePermissionsCommand struct {
	TenantID      shared.ID   `json:"tenant_id"`
	RoleID        shared.ID   `json:"role_id"`
	PermissionIDs []shared.ID `json:"permission_ids"`
}

type Usecase interface {
	Register(ctx context.Context, cmd RegisterCommand) (*identity.User, error)
	Login(ctx context.Context, cmd LoginCommand) (*LoginResult, error)
	SwitchTenant(ctx context.Context, cmd SwitchTenantCommand) (*LoginResult, error)
	GetProfile(ctx context.Context, userID shared.ID) (*UserProfileResult, error)

	// Member Management
	AddMember(ctx context.Context, cmd AddMemberCommand) (*identity.TenantUser, error)
	ListMembers(ctx context.Context, tenantID shared.ID) ([]identity.TenantUserItem, error)
	AssignMemberRoles(ctx context.Context, cmd AssignMemberRolesCommand) error
	RemoveMemberRole(ctx context.Context, tenantID, userID, roleID shared.ID) error

	// Role Management
	CreateRole(ctx context.Context, cmd CreateRoleCommand) (*identity.Role, error)
	GetRole(ctx context.Context, tenantID, roleID shared.ID) (*identity.Role, error)
	ListRoles(ctx context.Context, tenantID shared.ID) ([]identity.Role, error)
	AssignRolePermissions(ctx context.Context, cmd AssignRolePermissionsCommand) error
	ListPermissions(ctx context.Context) ([]identity.Permission, error)
}

type usecase struct {
	identityRepo identity.Repository
	tenantRepo   repository.TenantRepository
	tokenManager auth.TokenManager
	tokenTTL     time.Duration
}

// NewUsecase creates an Identity & RBAC usecase instance.
func NewUsecase(
	identityRepo identity.Repository,
	tenantRepo repository.TenantRepository,
	tokenManager auth.TokenManager,
	tokenTTL time.Duration,
) Usecase {
	if tokenTTL <= 0 {
		tokenTTL = 24 * time.Hour
	}
	return &usecase{
		identityRepo: identityRepo,
		tenantRepo:   tenantRepo,
		tokenManager: tokenManager,
		tokenTTL:     tokenTTL,
	}
}

func (u *usecase) Register(ctx context.Context, cmd RegisterCommand) (*identity.User, error) {
	email := strings.ToLower(strings.TrimSpace(cmd.Email))
	name := strings.TrimSpace(cmd.Name)
	password := cmd.Password

	if email == "" || !strings.Contains(email, "@") {
		return nil, fmt.Errorf("%w: valid email is required", shared.ErrInvalidInput)
	}
	if name == "" {
		return nil, fmt.Errorf("%w: name is required", shared.ErrInvalidInput)
	}
	if len(password) < 8 {
		return nil, fmt.Errorf("%w: password must be at least 8 characters", shared.ErrInvalidInput)
	}

	// Check if user already exists
	existing, err := u.identityRepo.GetUserByEmail(ctx, email)
	if err != nil && err != shared.ErrNotFound {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("%w: user with email '%s' already exists", shared.ErrAlreadyExists, email)
	}

	hashed, err := auth.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	id, err := shared.NewID()
	if err != nil {
		return nil, err
	}

	user := &identity.User{
		ID:           id,
		Email:        email,
		PasswordHash: hashed,
		Name:         name,
		Status:       identity.UserStatusActive,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := u.identityRepo.CreateUser(ctx, user); err != nil {
		return nil, err
	}

	// If tenant ID is provided, verify tenant and add membership
	if cmd.TenantID != nil && *cmd.TenantID != shared.NilID() {
		if u.tenantRepo != nil {
			tenant, err := u.tenantRepo.GetByID(ctx, *cmd.TenantID)
			if err != nil {
				return nil, fmt.Errorf("invalid tenant: %w", err)
			}
			if !tenant.IsActive() {
				return nil, shared.ErrTenantSuspended
			}
		}

		memberID, _ := shared.NewID()
		_ = u.identityRepo.AddTenantMember(ctx, &identity.TenantUser{
			ID:       memberID,
			TenantID: *cmd.TenantID,
			UserID:   user.ID,
			Status:   identity.MembershipStatusActive,
			JoinedAt: time.Now().UTC(),
		})
	}

	return user, nil
}

func (u *usecase) Login(ctx context.Context, cmd LoginCommand) (*LoginResult, error) {
	email := strings.ToLower(strings.TrimSpace(cmd.Email))
	if email == "" || cmd.Password == "" {
		return nil, fmt.Errorf("%w: email and password are required", shared.ErrInvalidInput)
	}

	user, err := u.identityRepo.GetUserByEmail(ctx, email)
	if err != nil {
		if err == shared.ErrNotFound {
			return nil, fmt.Errorf("%w: invalid credentials", shared.ErrUnauthorized)
		}
		return nil, err
	}

	if user.Status != identity.UserStatusActive {
		return nil, fmt.Errorf("%w: user account is %s", shared.ErrUnauthorized, user.Status)
	}

	if !auth.CheckPasswordHash(cmd.Password, user.PasswordHash) {
		return nil, fmt.Errorf("%w: invalid credentials", shared.ErrUnauthorized)
	}

	// If no tenant is specified, retrieve user's tenant memberships
	if cmd.TenantID == nil || *cmd.TenantID == shared.NilID() {
		tenants, err := u.identityRepo.ListUserTenants(ctx, user.ID)
		if err != nil {
			return nil, err
		}

		// If user belongs to exactly one tenant, auto-select it
		if len(tenants) == 1 {
			tenantID := tenants[0].TenantID
			return u.issueLoginResult(ctx, user, tenantID)
		}

		return &LoginResult{
			User:                    user,
			RequiresTenantSelection: true,
			AvailableTenants:        tenants,
		}, nil
	}

	return u.issueLoginResult(ctx, user, *cmd.TenantID)
}

func (u *usecase) issueLoginResult(ctx context.Context, user *identity.User, tenantID shared.ID) (*LoginResult, error) {
	// Verify membership
	member, err := u.identityRepo.GetTenantMember(ctx, tenantID, user.ID)
	if err != nil {
		if err == shared.ErrNotFound {
			return nil, fmt.Errorf("%w: user is not a member of tenant %s", shared.ErrForbidden, tenantID)
		}
		return nil, err
	}

	if member.Status != identity.MembershipStatusActive {
		return nil, fmt.Errorf("%w: tenant membership is %s", shared.ErrForbidden, member.Status)
	}

	// Fetch roles and permissions
	roles, err := u.identityRepo.ListUserRolesInTenant(ctx, tenantID, user.ID)
	if err != nil {
		return nil, err
	}

	roleCodes := make([]string, len(roles))
	for i, r := range roles {
		roleCodes[i] = r.Code
	}

	permissions, err := u.identityRepo.ListUserPermissionsInTenant(ctx, tenantID, user.ID)
	if err != nil {
		return nil, err
	}

	token, err := u.tokenManager.GenerateToken(user.ID, tenantID, user.Email, user.Name, roleCodes, permissions, u.tokenTTL)
	if err != nil {
		return nil, fmt.Errorf("failed to generate auth token: %w", err)
	}

	return &LoginResult{
		User:        user,
		Token:       token,
		ExpiresIn:   int64(u.tokenTTL.Seconds()),
		TenantID:    &tenantID,
		Roles:       roleCodes,
		Permissions: permissions,
	}, nil
}

func (u *usecase) SwitchTenant(ctx context.Context, cmd SwitchTenantCommand) (*LoginResult, error) {
	user, err := u.identityRepo.GetUserByID(ctx, cmd.UserID)
	if err != nil {
		return nil, err
	}
	return u.issueLoginResult(ctx, user, cmd.TenantID)
}

func (u *usecase) GetProfile(ctx context.Context, userID shared.ID) (*UserProfileResult, error) {
	user, err := u.identityRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	tenants, err := u.identityRepo.ListUserTenants(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &UserProfileResult{
		User:    user,
		Tenants: tenants,
	}, nil
}

// ----------------------------------------------------------------------------
// Member Management
// ----------------------------------------------------------------------------

func (u *usecase) AddMember(ctx context.Context, cmd AddMemberCommand) (*identity.TenantUser, error) {
	if cmd.TenantID == shared.NilID() || cmd.UserID == shared.NilID() {
		return nil, fmt.Errorf("%w: tenant_id and user_id are required", shared.ErrInvalidInput)
	}

	// Verify user exists
	if _, err := u.identityRepo.GetUserByID(ctx, cmd.UserID); err != nil {
		return nil, err
	}

	// Check if already a member
	existing, err := u.identityRepo.GetTenantMember(ctx, cmd.TenantID, cmd.UserID)
	if err != nil && err != shared.ErrNotFound {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("%w: user is already a member of this tenant", shared.ErrAlreadyExists)
	}

	memberID, err := shared.NewID()
	if err != nil {
		return nil, err
	}

	member := &identity.TenantUser{
		ID:       memberID,
		TenantID: cmd.TenantID,
		UserID:   cmd.UserID,
		Status:   identity.MembershipStatusActive,
		JoinedAt: time.Now().UTC(),
	}

	if err := u.identityRepo.AddTenantMember(ctx, member); err != nil {
		return nil, err
	}

	// Assign roles if provided
	for _, roleID := range cmd.RoleIDs {
		_ = u.identityRepo.AssignUserRole(ctx, cmd.TenantID, cmd.UserID, roleID)
	}

	return member, nil
}

func (u *usecase) ListMembers(ctx context.Context, tenantID shared.ID) ([]identity.TenantUserItem, error) {
	if tenantID == shared.NilID() {
		return nil, shared.ErrTenantRequired
	}
	return u.identityRepo.ListTenantUsers(ctx, tenantID)
}

func (u *usecase) AssignMemberRoles(ctx context.Context, cmd AssignMemberRolesCommand) error {
	if cmd.TenantID == shared.NilID() || cmd.UserID == shared.NilID() {
		return fmt.Errorf("%w: tenant_id and user_id are required", shared.ErrInvalidInput)
	}

	for _, roleID := range cmd.RoleIDs {
		if err := u.identityRepo.AssignUserRole(ctx, cmd.TenantID, cmd.UserID, roleID); err != nil {
			return err
		}
	}
	return nil
}

func (u *usecase) RemoveMemberRole(ctx context.Context, tenantID, userID, roleID shared.ID) error {
	if tenantID == shared.NilID() || userID == shared.NilID() || roleID == shared.NilID() {
		return fmt.Errorf("%w: tenant_id, user_id, and role_id are required", shared.ErrInvalidInput)
	}
	return u.identityRepo.RemoveUserRole(ctx, tenantID, userID, roleID)
}

// ----------------------------------------------------------------------------
// Role Management
// ----------------------------------------------------------------------------

func (u *usecase) CreateRole(ctx context.Context, cmd CreateRoleCommand) (*identity.Role, error) {
	if cmd.TenantID == shared.NilID() {
		return nil, shared.ErrTenantRequired
	}

	code := strings.ToLower(strings.TrimSpace(cmd.Code))
	name := strings.TrimSpace(cmd.Name)

	if code == "" || name == "" {
		return nil, fmt.Errorf("%w: code and name are required", shared.ErrInvalidInput)
	}

	existing, err := u.identityRepo.GetRoleByCode(ctx, cmd.TenantID, code)
	if err != nil && err != shared.ErrNotFound {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("%w: role code '%s' already exists in this tenant", shared.ErrAlreadyExists, code)
	}

	roleID, err := shared.NewID()
	if err != nil {
		return nil, err
	}

	role := &identity.Role{
		ID:          roleID,
		TenantID:    cmd.TenantID,
		Code:        code,
		Name:        name,
		Description: cmd.Description,
		IsSystem:    false,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := u.identityRepo.CreateRole(ctx, role); err != nil {
		return nil, err
	}

	if len(cmd.PermissionIDs) > 0 {
		if err := u.identityRepo.AssignPermissionsToRole(ctx, role.ID, cmd.PermissionIDs); err != nil {
			return nil, err
		}
		perms, _ := u.identityRepo.ListRolePermissions(ctx, role.ID)
		role.Permissions = perms
	}

	return role, nil
}

func (u *usecase) GetRole(ctx context.Context, tenantID, roleID shared.ID) (*identity.Role, error) {
	if tenantID == shared.NilID() {
		return nil, shared.ErrTenantRequired
	}
	return u.identityRepo.GetRoleByID(ctx, tenantID, roleID)
}

func (u *usecase) ListRoles(ctx context.Context, tenantID shared.ID) ([]identity.Role, error) {
	if tenantID == shared.NilID() {
		return nil, shared.ErrTenantRequired
	}
	return u.identityRepo.ListRoles(ctx, tenantID)
}

func (u *usecase) AssignRolePermissions(ctx context.Context, cmd AssignRolePermissionsCommand) error {
	if cmd.TenantID == shared.NilID() || cmd.RoleID == shared.NilID() {
		return fmt.Errorf("%w: tenant_id and role_id are required", shared.ErrInvalidInput)
	}

	// Verify role belongs to tenant
	if _, err := u.identityRepo.GetRoleByID(ctx, cmd.TenantID, cmd.RoleID); err != nil {
		return err
	}

	return u.identityRepo.AssignPermissionsToRole(ctx, cmd.RoleID, cmd.PermissionIDs)
}

func (u *usecase) ListPermissions(ctx context.Context) ([]identity.Permission, error) {
	return u.identityRepo.ListPermissions(ctx)
}
