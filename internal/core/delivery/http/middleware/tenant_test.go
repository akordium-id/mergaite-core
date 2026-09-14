package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/akordium-id/mergaite-core/internal/core/delivery/http/middleware"
	"github.com/akordium-id/mergaite-core/internal/core/domain/shared"
)

func TestTenantRequired_MissingHeader(t *testing.T) {
	handler := middleware.TenantRequired()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestTenantRequired_InvalidUUID(t *testing.T) {
	handler := middleware.TenantRequired()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set(middleware.HeaderTenantID, "not-a-valid-uuid")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestTenantRequired_ValidUUID(t *testing.T) {
	expectedID := shared.MustNewID()

	var extractedID shared.ID
	handler := middleware.TenantRequired()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := shared.GetTenantID(r.Context())
		if !ok {
			t.Fatal("expected tenant id in context")
		}
		extractedID = id
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set(middleware.HeaderTenantID, expectedID.String())
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if extractedID != expectedID {
		t.Fatalf("expected %v, got %v", expectedID, extractedID)
	}
}
