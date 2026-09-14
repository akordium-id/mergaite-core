package audit

import (
	"context"
	"time"

	"github.com/akordium-id/mergiate-core/internal/core/domain/audit"
	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
)

type AuditListResult struct {
	Items    []audit.AuditLog `json:"items"`
	Total    int64            `json:"total"`
	Page     int32            `json:"page"`
	PageSize int32            `json:"page_size"`
}

type RecordAuditCommand struct {
	ActorID    *shared.ID      `json:"actor_id,omitempty"`
	ActorType  audit.ActorType `json:"actor_type"`
	Action     audit.Action    `json:"action"`
	EntityType string          `json:"entity_type"`
	EntityID   shared.ID       `json:"entity_id"`
	Changes    map[string]any  `json:"changes,omitempty"`
	Metadata   map[string]any  `json:"metadata,omitempty"`
}

type Usecase interface {
	Record(ctx context.Context, cmd RecordAuditCommand) (*audit.AuditLog, error)
	List(ctx context.Context, page, pageSize int32, entityType *string, entityID, actorID *shared.ID, action *audit.Action) (*AuditListResult, error)
}

type usecase struct {
	repo audit.Repository
}

// NewUsecase constructs a new Audit usecase.
func NewUsecase(repo audit.Repository) Usecase {
	return &usecase{repo: repo}
}

func (u *usecase) Record(ctx context.Context, cmd RecordAuditCommand) (*audit.AuditLog, error) {
	tenantID, err := shared.RequireTenantID(ctx)
	if err != nil {
		return nil, err
	}

	id, err := shared.NewID()
	if err != nil {
		return nil, err
	}

	actorType := cmd.ActorType
	if actorType == "" {
		actorType = audit.ActorTypeUser
	}

	log := &audit.AuditLog{
		ID:         id,
		TenantID:   tenantID,
		ActorID:    cmd.ActorID,
		ActorType:  actorType,
		Action:     cmd.Action,
		EntityType: cmd.EntityType,
		EntityID:   cmd.EntityID,
		Changes:    cmd.Changes,
		Metadata:   cmd.Metadata,
		CreatedAt:  time.Now().UTC(),
	}

	if err := u.repo.Create(ctx, log); err != nil {
		return nil, err
	}

	return log, nil
}

func (u *usecase) List(ctx context.Context, page, pageSize int32, entityType *string, entityID, actorID *shared.ID, action *audit.Action) (*AuditListResult, error) {
	tenantID, err := shared.RequireTenantID(ctx)
	if err != nil {
		return nil, err
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	filter := audit.Filter{
		EntityType: entityType,
		EntityID:   entityID,
		ActorID:    actorID,
		Action:     action,
		Limit:      pageSize,
		Offset:     offset,
	}

	items, total, err := u.repo.List(ctx, tenantID, filter)
	if err != nil {
		return nil, err
	}

	return &AuditListResult{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}
