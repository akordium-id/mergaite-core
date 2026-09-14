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

type ServiceAccountHandler struct {
	usecase  identityusecase.ServiceAccountUsecase
	tokenMgr auth.TokenManager
}

// NewServiceAccountHandler constructs a REST handler for Service Accounts and API Keys.
func NewServiceAccountHandler(usecase identityusecase.ServiceAccountUsecase, tokenMgr auth.TokenManager) *ServiceAccountHandler {
	return &ServiceAccountHandler{
		usecase:  usecase,
		tokenMgr: tokenMgr,
	}
}

// RegisterRoutes registers all Service Account and API Key endpoints under /service-accounts.
func (h *ServiceAccountHandler) RegisterRoutes(r chi.Router) {
	r.Route("/service-accounts", func(r chi.Router) {
		r.Use(middleware.AuthRequired(h.tokenMgr, h.usecase))

		r.Get("/", h.ListServiceAccounts)
		r.Post("/", h.CreateServiceAccount)
		r.Get("/{id}", h.GetServiceAccount)
		r.Put("/{id}", h.UpdateServiceAccount)
		r.Delete("/{id}", h.DeleteServiceAccount)

		// API Keys
		r.Route("/{id}/keys", func(kr chi.Router) {
			kr.Get("/", h.ListAPIKeys)
			kr.Post("/", h.CreateAPIKey)
			kr.Delete("/{keyId}", h.RevokeAPIKey)
		})
	})
}

// ----------------------------------------------------------------------------
// Service Account Handlers
// ----------------------------------------------------------------------------

func (h *ServiceAccountHandler) CreateServiceAccount(w http.ResponseWriter, r *http.Request) {
	var input identityusecase.CreateServiceAccountInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", "Failed to parse JSON body")
		return
	}

	sa, err := h.usecase.CreateServiceAccount(r.Context(), input)
	if err != nil {
		response.FromDomainError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, sa)
}

func (h *ServiceAccountHandler) GetServiceAccount(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := shared.ParseID(idStr)
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "Invalid service account ID format")
		return
	}

	sa, err := h.usecase.GetServiceAccount(r.Context(), id)
	if err != nil {
		response.FromDomainError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, sa)
}

func (h *ServiceAccountHandler) ListServiceAccounts(w http.ResponseWriter, r *http.Request) {
	list, err := h.usecase.ListServiceAccounts(r.Context())
	if err != nil {
		response.FromDomainError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, list)
}

func (h *ServiceAccountHandler) UpdateServiceAccount(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := shared.ParseID(idStr)
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "Invalid service account ID format")
		return
	}

	var input identityusecase.UpdateServiceAccountInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", "Failed to parse JSON body")
		return
	}

	sa, err := h.usecase.UpdateServiceAccount(r.Context(), id, input)
	if err != nil {
		response.FromDomainError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, sa)
}

func (h *ServiceAccountHandler) DeleteServiceAccount(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := shared.ParseID(idStr)
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "Invalid service account ID format")
		return
	}

	if err := h.usecase.DeleteServiceAccount(r.Context(), id); err != nil {
		response.FromDomainError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"message": "Service account deleted successfully",
	})
}

// ----------------------------------------------------------------------------
// API Key Handlers
// ----------------------------------------------------------------------------

func (h *ServiceAccountHandler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	saID, err := shared.ParseID(idStr)
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "Invalid service account ID format")
		return
	}

	var input identityusecase.CreateAPIKeyInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", "Failed to parse JSON body")
		return
	}

	keyWithSecret, err := h.usecase.CreateAPIKey(r.Context(), saID, input)
	if err != nil {
		response.FromDomainError(w, err)
		return
	}

	// 201 Created with secret_key for one-time display
	response.JSON(w, http.StatusCreated, keyWithSecret)
}

func (h *ServiceAccountHandler) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	saID, err := shared.ParseID(idStr)
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "Invalid service account ID format")
		return
	}

	keys, err := h.usecase.ListAPIKeys(r.Context(), saID)
	if err != nil {
		response.FromDomainError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, keys)
}

func (h *ServiceAccountHandler) RevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	saIDStr := chi.URLParam(r, "id")
	saID, err := shared.ParseID(saIDStr)
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "Invalid service account ID format")
		return
	}

	keyIDStr := chi.URLParam(r, "keyId")
	keyID, err := shared.ParseID(keyIDStr)
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "Invalid API key ID format")
		return
	}

	if err := h.usecase.RevokeAPIKey(r.Context(), saID, keyID); err != nil {
		response.FromDomainError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"message": "API key revoked successfully",
	})
}
