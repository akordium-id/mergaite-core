package v1

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/akordium-id/mergiate-core/internal/core/delivery/http/middleware"
	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
	identityusecase "github.com/akordium-id/mergiate-core/internal/core/usecase/identity"
	"github.com/akordium-id/mergiate-core/pkg/auth"
	"github.com/akordium-id/mergiate-core/pkg/response"
)

type IdentityHandler struct {
	usecase  identityusecase.Usecase
	tokenMgr auth.TokenManager
}

func NewIdentityHandler(usecase identityusecase.Usecase, tokenMgr auth.TokenManager) *IdentityHandler {
	return &IdentityHandler{
		usecase:  usecase,
		tokenMgr: tokenMgr,
	}
}

func (h *IdentityHandler) RegisterRoutes(r chi.Router) {
	r.Route("/iam", func(r chi.Router) {
		r.Use(middleware.AuthRequired(h.tokenMgr))

		// Permissions catalog
		r.Get("/permissions", h.ListPermissions)

		// Member Management
		r.Route("/members", func(r chi.Router) {
			r.With(middleware.RequireAnyPermission("iam:manage", "user:manage", "*")).Get("/", h.ListMembers)
			r.With(middleware.RequireAnyPermission("iam:manage", "user:manage", "*")).Post("/", h.AddMember)
			r.With(middleware.RequireAnyPermission("iam:manage", "user:manage", "*")).Post("/{id}/roles", h.AssignMemberRoles)
			r.With(middleware.RequireAnyPermission("iam:manage", "user:manage", "*")).Delete("/{id}/roles/{roleId}", h.RemoveMemberRole)
		})

		// Role Management
		r.Route("/roles", func(r chi.Router) {
			r.With(middleware.RequireAnyPermission("iam:manage", "role:manage", "*")).Get("/", h.ListRoles)
			r.With(middleware.RequireAnyPermission("iam:manage", "role:manage", "*")).Post("/", h.CreateRole)
			r.With(middleware.RequireAnyPermission("iam:manage", "role:manage", "*")).Get("/{id}", h.GetRole)
			r.With(middleware.RequireAnyPermission("iam:manage", "role:manage", "*")).Post("/{id}/permissions", h.AssignRolePermissions)
		})
	})
}

// ----------------------------------------------------------------------------
// Members
// ----------------------------------------------------------------------------

func (h *IdentityHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	tenantID, err := shared.RequireTenantID(r.Context())
	if err != nil {
		response.FromDomainError(w, err)
		return
	}

	members, err := h.usecase.ListMembers(r.Context(), tenantID)
	if err != nil {
		response.FromDomainError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, members)
}

type addMemberReq struct {
	UserID  shared.ID   `json:"user_id"`
	RoleIDs []shared.ID `json:"role_ids"`
}

func (h *IdentityHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	tenantID, err := shared.RequireTenantID(r.Context())
	if err != nil {
		response.FromDomainError(w, err)
		return
	}

	var req addMemberReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", "Malformed request body")
		return
	}

	member, err := h.usecase.AddMember(r.Context(), identityusecase.AddMemberCommand{
		TenantID: tenantID,
		UserID:   req.UserID,
		RoleIDs:  req.RoleIDs,
	})
	if err != nil {
		response.FromDomainError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, member)
}

type assignRolesReq struct {
	RoleIDs []shared.ID `json:"role_ids"`
}

func (h *IdentityHandler) AssignMemberRoles(w http.ResponseWriter, r *http.Request) {
	tenantID, err := shared.RequireTenantID(r.Context())
	if err != nil {
		response.FromDomainError(w, err)
		return
	}

	targetUserID, err := shared.ParseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "Invalid target user ID")
		return
	}

	var req assignRolesReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", "Malformed request body")
		return
	}

	err = h.usecase.AssignMemberRoles(r.Context(), identityusecase.AssignMemberRolesCommand{
		TenantID: tenantID,
		UserID:   targetUserID,
		RoleIDs:  req.RoleIDs,
	})
	if err != nil {
		response.FromDomainError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "roles assigned successfully"})
}

func (h *IdentityHandler) RemoveMemberRole(w http.ResponseWriter, r *http.Request) {
	tenantID, err := shared.RequireTenantID(r.Context())
	if err != nil {
		response.FromDomainError(w, err)
		return
	}

	targetUserID, err := shared.ParseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "Invalid target user ID")
		return
	}

	roleID, err := shared.ParseID(chi.URLParam(r, "roleId"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "Invalid role ID")
		return
	}

	err = h.usecase.RemoveMemberRole(r.Context(), tenantID, targetUserID, roleID)
	if err != nil {
		response.FromDomainError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "role removed successfully"})
}

// ----------------------------------------------------------------------------
// Roles
// ----------------------------------------------------------------------------

func (h *IdentityHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
	tenantID, err := shared.RequireTenantID(r.Context())
	if err != nil {
		response.FromDomainError(w, err)
		return
	}

	roles, err := h.usecase.ListRoles(r.Context(), tenantID)
	if err != nil {
		response.FromDomainError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, roles)
}

type createRoleReq struct {
	Code          string      `json:"code"`
	Name          string      `json:"name"`
	Description   string      `json:"description"`
	PermissionIDs []shared.ID `json:"permission_ids"`
}

func (h *IdentityHandler) CreateRole(w http.ResponseWriter, r *http.Request) {
	tenantID, err := shared.RequireTenantID(r.Context())
	if err != nil {
		response.FromDomainError(w, err)
		return
	}

	var req createRoleReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", "Malformed request body")
		return
	}

	role, err := h.usecase.CreateRole(r.Context(), identityusecase.CreateRoleCommand{
		TenantID:      tenantID,
		Code:          req.Code,
		Name:          req.Name,
		Description:   req.Description,
		PermissionIDs: req.PermissionIDs,
	})
	if err != nil {
		response.FromDomainError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, role)
}

func (h *IdentityHandler) GetRole(w http.ResponseWriter, r *http.Request) {
	tenantID, err := shared.RequireTenantID(r.Context())
	if err != nil {
		response.FromDomainError(w, err)
		return
	}

	roleID, err := shared.ParseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "Invalid role ID")
		return
	}

	role, err := h.usecase.GetRole(r.Context(), tenantID, roleID)
	if err != nil {
		response.FromDomainError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, role)
}

type assignPermsReq struct {
	PermissionIDs []shared.ID `json:"permission_ids"`
}

func (h *IdentityHandler) AssignRolePermissions(w http.ResponseWriter, r *http.Request) {
	tenantID, err := shared.RequireTenantID(r.Context())
	if err != nil {
		response.FromDomainError(w, err)
		return
	}

	roleID, err := shared.ParseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "Invalid role ID")
		return
	}

	var req assignPermsReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", "Malformed request body")
		return
	}

	err = h.usecase.AssignRolePermissions(r.Context(), identityusecase.AssignRolePermissionsCommand{
		TenantID:      tenantID,
		RoleID:        roleID,
		PermissionIDs: req.PermissionIDs,
	})
	if err != nil {
		response.FromDomainError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "permissions assigned successfully"})
}

func (h *IdentityHandler) ListPermissions(w http.ResponseWriter, r *http.Request) {
	perms, err := h.usecase.ListPermissions(r.Context())
	if err != nil {
		response.FromDomainError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, perms)
}
