package audit_test

import (
	"context"
	"testing"

	domainaudit "github.com/akordium-id/mergaite-core/internal/core/domain/audit"
	"github.com/akordium-id/mergaite-core/internal/core/domain/shared"
	usecaseaudit "github.com/akordium-id/mergaite-core/internal/core/usecase/audit"
)

type mockAuditRepo struct {
	logs []domainaudit.AuditLog
}

func (m *mockAuditRepo) Create(ctx context.Context, log *domainaudit.AuditLog) error {
	m.logs = append(m.logs, *log)
	return nil
}

func (m *mockAuditRepo) List(ctx context.Context, tenantID shared.ID, filter domainaudit.Filter) ([]domainaudit.AuditLog, int64, error) {
	var filtered []domainaudit.AuditLog
	for _, l := range m.logs {
		if l.TenantID != tenantID {
			continue
		}
		if filter.EntityType != nil && l.EntityType != *filter.EntityType {
			continue
		}
		filtered = append(filtered, l)
	}
	return filtered, int64(len(filtered)), nil
}

func TestAuditUsecase_RecordAndList(t *testing.T) {
	repo := &mockAuditRepo{}
	uc := usecaseaudit.NewUsecase(repo)

	tenantID, _ := shared.NewID()
	actorID, _ := shared.NewID()
	entityID, _ := shared.NewID()

	ctx := shared.WithTenantID(context.Background(), tenantID)

	// 1. Record an audit log
	log, err := uc.Record(ctx, usecaseaudit.RecordAuditCommand{
		ActorID:    &actorID,
		ActorType:  domainaudit.ActorTypeUser,
		Action:     domainaudit.ActionUpdate,
		EntityType: "product",
		EntityID:   entityID,
		Changes: map[string]any{
			"old_price": 10000,
			"new_price": 12000,
		},
		Metadata: map[string]any{"ip": "127.0.0.1"},
	})
	if err != nil {
		t.Fatalf("failed to record audit: %v", err)
	}

	if log.Action != domainaudit.ActionUpdate {
		t.Errorf("expected action 'update', got '%s'", log.Action)
	}

	// 2. List audit logs with matching filter
	entityTypeProduct := "product"
	res, err := uc.List(ctx, 1, 10, &entityTypeProduct, nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to list audit logs: %v", err)
	}
	if res.Total != 1 {
		t.Errorf("expected total 1, got %d", res.Total)
	}

	// 3. List audit logs with non-matching filter
	entityTypeDoc := "document"
	resDoc, err := uc.List(ctx, 1, 10, &entityTypeDoc, nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to list audit logs: %v", err)
	}
	if resDoc.Total != 0 {
		t.Errorf("expected total 0, got %d", resDoc.Total)
	}
}
