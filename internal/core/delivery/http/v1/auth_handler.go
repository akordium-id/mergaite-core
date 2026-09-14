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

type AuthHandler struct {
	usecase  identityusecase.Usecase
	tokenMgr auth.TokenManager
}

func NewAuthHandler(usecase identityusecase.Usecase, tokenMgr auth.TokenManager) *AuthHandler {
	return &AuthHandler{
		usecase:  usecase,
		tokenMgr: tokenMgr,
	}
}

func (h *AuthHandler) RegisterRoutes(r chi.Router) {
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", h.Register)
		r.Post("/login", h.Login)

		// Protected endpoints
		r.Group(func(r chi.Router) {
			r.Use(middleware.AuthRequired(h.tokenMgr))
			r.Get("/me", h.Me)
			r.Post("/switch-tenant", h.SwitchTenant)
		})
	})
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var cmd identityusecase.RegisterCommand
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", "Malformed request body")
		return
	}

	user, err := h.usecase.Register(r.Context(), cmd)
	if err != nil {
		response.FromDomainError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, map[string]any{
		"user": user,
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var cmd identityusecase.LoginCommand
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", "Malformed request body")
		return
	}

	res, err := h.usecase.Login(r.Context(), cmd)
	if err != nil {
		response.FromDomainError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, res)
}

type switchTenantReq struct {
	TenantID shared.ID `json:"tenant_id"`
}

func (h *AuthHandler) SwitchTenant(w http.ResponseWriter, r *http.Request) {
	claims, ok := shared.GetAuthClaims(r.Context())
	if !ok || claims == nil {
		response.Err(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	var req switchTenantReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", "Malformed request body")
		return
	}

	res, err := h.usecase.SwitchTenant(r.Context(), identityusecase.SwitchTenantCommand{
		UserID:   claims.UserID,
		TenantID: req.TenantID,
	})
	if err != nil {
		response.FromDomainError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, res)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := shared.GetAuthClaims(r.Context())
	if !ok || claims == nil {
		response.Err(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	profile, err := h.usecase.GetProfile(r.Context(), claims.UserID)
	if err != nil {
		response.FromDomainError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"profile": profile,
		"session": claims,
	})
}
