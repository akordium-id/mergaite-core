package identity_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/akordium-id/mergiate-core/internal/core/domain/identity"
	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
	usecase "github.com/akordium-id/mergiate-core/internal/core/usecase/identity"
	"github.com/akordium-id/mergiate-core/pkg/auth"
)

type mockIdentityRepo struct {
	users           map[shared.ID]*identity.User
	usersByEmail    map[string]*identity.User
	tenantMembers   map[string]*identity.TenantUser
	userTenants     map[shared.ID][]identity.TenantMembership
	roles           map[shared.ID]*identity.Role
	rolePermissions map[shared.ID][]shared.ID
	userRoles       map[string][]shared.ID
	permissions     []identity.Permission
}

func newMockIdentityRepo() *mockIdentityRepo {
	return &mockIdentityRepo{
		users:           make(map[shared.ID]*identity.User),
		usersByEmail:    make(map[string]*identity.User),
		tenantMembers:   make(map[string]*identity.TenantUser),
		userTenants:     make(map[shared.ID][]identity.TenantMembership),
		roles:           make(map[shared.ID]*identity.Role),
		rolePermissions: make(map[shared.ID][]shared.ID),
		userRoles:       make(map[string][]shared.ID),
		permissions: []identity.Permission{
			{ID: shared.MustNewID(), Code: "document:create", Name: "Create Document", Category: "document"},
			{ID: shared.MustNewID(), Code: "document:approve", Name: "Approve Document", Category: "document"},
			{ID: shared.MustNewID(), Code: "audit:read", Name: "Read Audit Log", Category: "audit"},
		},
	}
}

func (m *mockIdentityRepo) CreateUser(ctx context.Context, u *identity.User) error {
	m.users[u.ID] = u
	m.usersByEmail[u.Email] = u
	return nil
}

func (m *mockIdentityRepo) GetUserByEmail(ctx context.Context, email string) (*identity.User, error) {
	u, ok := m.usersByEmail[email]
	if !ok {
		return nil, shared.ErrNotFound
	}
	return u, nil
}

func (m *mockIdentityRepo) GetUserByID(ctx context.Context, id shared.ID) (*identity.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, shared.ErrNotFound
	}
	return u, nil
}

func (m *mockIdentityRepo) AddTenantMember(ctx context.Context, member *identity.TenantUser) error {
	key := member.TenantID.String() + ":" + member.UserID.String()
	m.tenantMembers[key] = member
	m.userTenants[member.UserID] = append(m.userTenants[member.UserID], identity.TenantMembership{
		TenantID:         member.TenantID,
		TenantCode:       "tenant-" + member.TenantID.String()[:8],
		TenantName:       "Tenant " + member.TenantID.String()[:8],
		TenantStatus:     "active",
		MembershipStatus: member.Status,
		JoinedAt:         member.JoinedAt,
	})
	return nil
}

func (m *mockIdentityRepo) GetTenantMember(ctx context.Context, tenantID, userID shared.ID) (*identity.TenantUser, error) {
	key := tenantID.String() + ":" + userID.String()
	member, ok := m.tenantMembers[key]
	if !ok {
		return nil, shared.ErrNotFound
	}
	return member, nil
}

func (m *mockIdentityRepo) ListUserTenants(ctx context.Context, userID shared.ID) ([]identity.TenantMembership, error) {
	return m.userTenants[userID], nil
}

func (m *mockIdentityRepo) ListTenantUsers(ctx context.Context, tenantID shared.ID) ([]identity.TenantUserItem, error) {
	var items []identity.TenantUserItem
	for _, member := range m.tenantMembers {
		if member.TenantID == tenantID {
			user := m.users[member.UserID]
			if user != nil {
				items = append(items, identity.TenantUserItem{
					UserID:           user.ID,
					Email:            user.Email,
					Name:             user.Name,
					UserStatus:       user.Status,
					MembershipStatus: member.Status,
					JoinedAt:         member.JoinedAt,
				})
			}
		}
	}
	return items, nil
}

func (m *mockIdentityRepo) CreateRole(ctx context.Context, r *identity.Role) error {
	m.roles[r.ID] = r
	return nil
}

func (m *mockIdentityRepo) GetRoleByID(ctx context.Context, tenantID, id shared.ID) (*identity.Role, error) {
	r, ok := m.roles[id]
	if !ok || r.TenantID != tenantID {
		return nil, shared.ErrNotFound
	}
	return r, nil
}

func (m *mockIdentityRepo) GetRoleByCode(ctx context.Context, tenantID shared.ID, code string) (*identity.Role, error) {
	for _, r := range m.roles {
		if r.TenantID == tenantID && r.Code == code {
			return r, nil
		}
	}
	return nil, shared.ErrNotFound
}

func (m *mockIdentityRepo) ListRoles(ctx context.Context, tenantID shared.ID) ([]identity.Role, error) {
	var res []identity.Role
	for _, r := range m.roles {
		if r.TenantID == tenantID {
			res = append(res, *r)
		}
	}
	return res, nil
}

func (m *mockIdentityRepo) ListPermissions(ctx context.Context) ([]identity.Permission, error) {
	return m.permissions, nil
}

func (m *mockIdentityRepo) AssignPermissionsToRole(ctx context.Context, roleID shared.ID, permissionIDs []shared.ID) error {
	m.rolePermissions[roleID] = permissionIDs
	return nil
}

func (m *mockIdentityRepo) ListRolePermissions(ctx context.Context, roleID shared.ID) ([]identity.Permission, error) {
	pIDs := m.rolePermissions[roleID]
	var res []identity.Permission
	for _, pID := range pIDs {
		for _, p := range m.permissions {
			if p.ID == pID {
				res = append(res, p)
			}
		}
	}
	return res, nil
}

func (m *mockIdentityRepo) AssignUserRole(ctx context.Context, tenantID, userID, roleID shared.ID) error {
	key := tenantID.String() + ":" + userID.String()
	m.userRoles[key] = append(m.userRoles[key], roleID)
	return nil
}

func (m *mockIdentityRepo) RemoveUserRole(ctx context.Context, tenantID, userID, roleID shared.ID) error {
	key := tenantID.String() + ":" + userID.String()
	roles := m.userRoles[key]
	var updated []shared.ID
	for _, rID := range roles {
		if rID != roleID {
			updated = append(updated, rID)
		}
	}
	m.userRoles[key] = updated
	return nil
}

func (m *mockIdentityRepo) ListUserRolesInTenant(ctx context.Context, tenantID, userID shared.ID) ([]identity.Role, error) {
	key := tenantID.String() + ":" + userID.String()
	rIDs := m.userRoles[key]
	var res []identity.Role
	for _, rID := range rIDs {
		if r, ok := m.roles[rID]; ok {
			res = append(res, *r)
		}
	}
	return res, nil
}

func (m *mockIdentityRepo) ListUserPermissionsInTenant(ctx context.Context, tenantID, userID shared.ID) ([]string, error) {
	key := tenantID.String() + ":" + userID.String()
	rIDs := m.userRoles[key]
	permSet := make(map[string]struct{})
	for _, rID := range rIDs {
		for _, pID := range m.rolePermissions[rID] {
			for _, p := range m.permissions {
				if p.ID == pID {
					permSet[p.Code] = struct{}{}
				}
			}
		}
	}
	res := make([]string, 0, len(permSet))
	for code := range permSet {
		res = append(res, code)
	}
	return res, nil
}

func TestIdentityUsecase_RegisterAndLogin(t *testing.T) {
	ctx := context.Background()
	repo := newMockIdentityRepo()
	tokenMgr := auth.NewTokenManager("test-secret-key-12345", "test-issuer")
	uc := usecase.NewUsecase(repo, nil, tokenMgr, 1*time.Hour)

	tenantID := shared.MustNewID()

	// 1. Register User
	user, err := uc.Register(ctx, usecase.RegisterCommand{
		Email:    "faiq@akordium.com",
		Password: "strongPassword123",
		Name:     "Faiq Akordium",
		TenantID: &tenantID,
	})
	if err != nil {
		t.Fatalf("failed to register user: %v", err)
	}
	if user.Email != "faiq@akordium.com" {
		t.Errorf("expected email 'faiq@akordium.com', got '%s'", user.Email)
	}

	// 2. Duplicate Registration
	_, err = uc.Register(ctx, usecase.RegisterCommand{
		Email:    "faiq@akordium.com",
		Password: "anotherPassword",
		Name:     "Duplicate",
	})
	if !errors.Is(err, shared.ErrAlreadyExists) {
		t.Errorf("expected ErrAlreadyExists, got %v", err)
	}

	// 3. Create Admin Role and Assign to Tenant
	role, err := uc.CreateRole(ctx, usecase.CreateRoleCommand{
		TenantID:      tenantID,
		Code:          "admin",
		Name:          "Administrator",
		PermissionIDs: []shared.ID{repo.permissions[0].ID, repo.permissions[1].ID},
	})
	if err != nil {
		t.Fatalf("failed to create role: %v", err)
	}

	// Assign role to member
	err = uc.AssignMemberRoles(ctx, usecase.AssignMemberRolesCommand{
		TenantID: tenantID,
		UserID:   user.ID,
		RoleIDs:  []shared.ID{role.ID},
	})
	if err != nil {
		t.Fatalf("failed to assign role: %v", err)
	}

	// 4. Login with single tenant (auto-resolves token)
	loginRes, err := uc.Login(ctx, usecase.LoginCommand{
		Email:    "faiq@akordium.com",
		Password: "strongPassword123",
	})
	if err != nil {
		t.Fatalf("failed to login: %v", err)
	}
	if loginRes.Token == "" {
		t.Fatal("expected JWT token in login response")
	}
	if len(loginRes.Roles) != 1 || loginRes.Roles[0] != "admin" {
		t.Errorf("expected role 'admin', got %v", loginRes.Roles)
	}
	if len(loginRes.Permissions) != 2 {
		t.Errorf("expected 2 permissions, got %d", len(loginRes.Permissions))
	}

	// 5. Validate the issued JWT token
	claims, err := tokenMgr.ValidateToken(loginRes.Token)
	if err != nil {
		t.Fatalf("failed to validate issued token: %v", err)
	}
	if claims.UserID != user.ID {
		t.Errorf("expected claims UserID %v, got %v", user.ID, claims.UserID)
	}
	if claims.TenantID != tenantID {
		t.Errorf("expected claims TenantID %v, got %v", tenantID, claims.TenantID)
	}
}

func TestIdentityUsecase_LoginInvalidCredentials(t *testing.T) {
	ctx := context.Background()
	repo := newMockIdentityRepo()
	tokenMgr := auth.NewTokenManager("test-secret", "issuer")
	uc := usecase.NewUsecase(repo, nil, tokenMgr, time.Hour)

	_, err := uc.Register(ctx, usecase.RegisterCommand{
		Email:    "user@akordium.com",
		Password: "correctPassword123",
		Name:     "Regular User",
	})
	if err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	// Wrong password
	_, err = uc.Login(ctx, usecase.LoginCommand{
		Email:    "user@akordium.com",
		Password: "wrongPassword",
	})
	if !errors.Is(err, shared.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}

	// User not found
	_, err = uc.Login(ctx, usecase.LoginCommand{
		Email:    "nonexistent@akordium.com",
		Password: "anyPassword",
	})
	if !errors.Is(err, shared.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}
