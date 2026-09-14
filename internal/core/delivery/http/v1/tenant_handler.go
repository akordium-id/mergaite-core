package v1

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/akordium-id/mergaite-core/internal/core/domain/shared"
	"github.com/akordium-id/mergaite-core/internal/core/usecase/tenant"
	"github.com/akordium-id/mergaite-core/pkg/response"
)

type TenantHandler struct {
	usecase tenant.Usecase
}

func NewTenantHandler(usecase tenant.Usecase) *TenantHandler {
	return &TenantHandler{usecase: usecase}
}

func (h *TenantHandler) RegisterRoutes(r chi.Router) {
	r.Route("/tenants", func(r chi.Router) {
		r.Post("/", h.Create)
		r.Get("/", h.List)
		r.Get("/{id}", h.GetByID)
		r.Get("/code/{code}", h.GetByCode)
		r.Put("/{id}", h.Update)
	})
}

func (h *TenantHandler) Create(w http.ResponseWriter, r *http.Request) {
	var cmd tenant.CreateTenantCommand
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", "Malformed request body")
		return
	}

	result, err := h.usecase.CreateTenant(r.Context(), cmd)
	if err != nil {
		response.FromDomainError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, result)
}

func (h *TenantHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := shared.ParseID(idStr)
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "Invalid UUID format")
		return
	}

	result, err := h.usecase.GetTenant(r.Context(), id)
	if err != nil {
		response.FromDomainError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

func (h *TenantHandler) GetByCode(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	result, err := h.usecase.GetTenantByCode(r.Context(), code)
	if err != nil {
		response.FromDomainError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

func (h *TenantHandler) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

	result, err := h.usecase.ListTenants(r.Context(), int32(page), int32(pageSize))
	if err != nil {
		response.FromDomainError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result.Items, map[string]any{
		"total":     result.Total,
		"page":      result.Page,
		"page_size": result.PageSize,
	})
}

func (h *TenantHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := shared.ParseID(idStr)
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "Invalid UUID format")
		return
	}

	var cmd tenant.UpdateTenantCommand
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", "Malformed request body")
		return
	}
	cmd.ID = id

	result, err := h.usecase.UpdateTenant(r.Context(), cmd)
	if err != nil {
		response.FromDomainError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}
