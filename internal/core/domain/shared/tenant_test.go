package shared_test

import (
	"context"
	"errors"
	"testing"

	"github.com/akordium-id/mergaite-core/internal/core/domain/shared"
)

func TestTenant_ContextHandling(t *testing.T) {
	ctx := context.Background()

	// Missing tenant ID
	id, ok := shared.GetTenantID(ctx)
	if ok || id != shared.NilID() {
		t.Errorf("expected missing tenant ID, got %v (ok: %t)", id, ok)
	}

	_, err := shared.RequireTenantID(ctx)
	if !errors.Is(err, shared.ErrTenantRequired) {
		t.Errorf("expected ErrTenantRequired, got %v", err)
	}

	// Injected tenant ID
	expectedID := shared.MustNewID()
	tenantCtx := shared.WithTenantID(ctx, expectedID)

	id, ok = shared.GetTenantID(tenantCtx)
	if !ok || id != expectedID {
		t.Errorf("expected %v (ok: true), got %v (ok: %t)", expectedID, id, ok)
	}

	requiredID, err := shared.RequireTenantID(tenantCtx)
	if err != nil || requiredID != expectedID {
		t.Errorf("expected %v without error, got %v (err: %v)", expectedID, requiredID, err)
	}
}
