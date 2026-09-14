package sequence

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/akordium-id/mergiate-core/internal/core/domain/sequence"
	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
)

type CreateSequenceCommand struct {
	TenantID    shared.ID            `json:"tenant_id"`
	Code        string               `json:"code"`
	Name        string               `json:"name"`
	EntityType  string               `json:"entity_type"`
	SubType     string               `json:"sub_type"`
	Prefix      string               `json:"prefix"`
	Suffix      string               `json:"suffix"`
	Template    string               `json:"template"`
	Padding     int                  `json:"padding"`
	StartValue  int64                `json:"start_value"`
	IncrementBy int                  `json:"increment_by"`
	ResetPolicy sequence.ResetPolicy `json:"reset_policy"`
}

type UpdateSequenceCommand struct {
	TenantID    shared.ID            `json:"tenant_id"`
	ID          shared.ID            `json:"id"`
	Name        string               `json:"name"`
	Prefix      string               `json:"prefix"`
	Suffix      string               `json:"suffix"`
	Template    string               `json:"template"`
	Padding     int                  `json:"padding"`
	ResetPolicy sequence.ResetPolicy `json:"reset_policy"`
	IsActive    bool                 `json:"is_active"`
}

type AcquireNextCommand struct {
	TenantID    shared.ID         `json:"tenant_id"`
	EntityType  string            `json:"entity_type"`
	SubType     string            `json:"sub_type"`
	ExtraTokens map[string]string `json:"extra_tokens,omitempty"`
}

type PreviewCommand struct {
	TenantID    shared.ID         `json:"tenant_id"`
	EntityType  string            `json:"entity_type"`
	SubType     string            `json:"sub_type"`
	ExtraTokens map[string]string `json:"extra_tokens,omitempty"`
}

type Usecase interface {
	CreateSequence(ctx context.Context, cmd CreateSequenceCommand) (*sequence.Sequence, error)
	GetSequence(ctx context.Context, tenantID, id shared.ID) (*sequence.Sequence, error)
	GetSequenceByEntity(ctx context.Context, tenantID shared.ID, entityType, subType string) (*sequence.Sequence, error)
	ListSequences(ctx context.Context, tenantID shared.ID) ([]sequence.Sequence, error)
	UpdateSequence(ctx context.Context, cmd UpdateSequenceCommand) (*sequence.Sequence, error)
	DeleteSequence(ctx context.Context, tenantID, id shared.ID) error

	AcquireNextNumber(ctx context.Context, cmd AcquireNextCommand) (string, error)
	PreviewNextNumber(ctx context.Context, cmd PreviewCommand) (string, error)
}

type usecase struct {
	repo sequence.Repository
}

// NewUsecase constructs an auto-numbering sequence usecase instance.
func NewUsecase(repo sequence.Repository) Usecase {
	return &usecase{repo: repo}
}

func (u *usecase) CreateSequence(ctx context.Context, cmd CreateSequenceCommand) (*sequence.Sequence, error) {
	if cmd.TenantID == shared.NilID() {
		return nil, shared.ErrTenantRequired
	}

	code := strings.ToLower(strings.TrimSpace(cmd.Code))
	name := strings.TrimSpace(cmd.Name)
	entityType := strings.ToLower(strings.TrimSpace(cmd.EntityType))
	subType := strings.ToLower(strings.TrimSpace(cmd.SubType))

	if code == "" || name == "" || entityType == "" || subType == "" {
		return nil, fmt.Errorf("%w: code, name, entity_type, and sub_type are required", shared.ErrInvalidInput)
	}

	if cmd.ResetPolicy != "" && !cmd.ResetPolicy.IsValid() {
		return nil, fmt.Errorf("%w: invalid reset_policy '%s'", shared.ErrInvalidInput, cmd.ResetPolicy)
	}
	if cmd.ResetPolicy == "" {
		cmd.ResetPolicy = sequence.ResetNever
	}

	if cmd.Padding <= 0 {
		cmd.Padding = 4
	}
	if cmd.StartValue <= 0 {
		cmd.StartValue = 1
	}
	if cmd.IncrementBy <= 0 {
		cmd.IncrementBy = 1
	}
	if cmd.Template == "" {
		cmd.Template = "{PREFIX}/{YYYY}/{MM}/{SEQ:4}"
	}

	// Check existing by code
	existing, err := u.repo.GetByCode(ctx, cmd.TenantID, code)
	if err != nil && err != shared.ErrNotFound {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("%w: sequence with code '%s' already exists", shared.ErrAlreadyExists, code)
	}

	// Check existing by entity + sub_type
	existingEntity, err := u.repo.GetByEntity(ctx, cmd.TenantID, entityType, subType)
	if err != nil && err != shared.ErrNotFound {
		return nil, err
	}
	if existingEntity != nil {
		return nil, fmt.Errorf("%w: active sequence for %s/%s already exists", shared.ErrAlreadyExists, entityType, subType)
	}

	id, err := shared.NewID()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	seq := &sequence.Sequence{
		ID:           id,
		TenantID:     cmd.TenantID,
		Code:         code,
		Name:         name,
		EntityType:   entityType,
		SubType:      subType,
		Prefix:       cmd.Prefix,
		Suffix:       cmd.Suffix,
		Template:     cmd.Template,
		Padding:      cmd.Padding,
		StartValue:   cmd.StartValue,
		IncrementBy:  cmd.IncrementBy,
		CurrentValue: 0,
		ResetPolicy:  cmd.ResetPolicy,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := u.repo.Create(ctx, seq); err != nil {
		return nil, err
	}

	return seq, nil
}

func (u *usecase) GetSequence(ctx context.Context, tenantID, id shared.ID) (*sequence.Sequence, error) {
	if tenantID == shared.NilID() {
		return nil, shared.ErrTenantRequired
	}
	return u.repo.GetByID(ctx, tenantID, id)
}

func (u *usecase) GetSequenceByEntity(ctx context.Context, tenantID shared.ID, entityType, subType string) (*sequence.Sequence, error) {
	if tenantID == shared.NilID() {
		return nil, shared.ErrTenantRequired
	}
	return u.repo.GetByEntity(ctx, tenantID, strings.ToLower(entityType), strings.ToLower(subType))
}

func (u *usecase) ListSequences(ctx context.Context, tenantID shared.ID) ([]sequence.Sequence, error) {
	if tenantID == shared.NilID() {
		return nil, shared.ErrTenantRequired
	}
	return u.repo.List(ctx, tenantID)
}

func (u *usecase) UpdateSequence(ctx context.Context, cmd UpdateSequenceCommand) (*sequence.Sequence, error) {
	if cmd.TenantID == shared.NilID() {
		return nil, shared.ErrTenantRequired
	}

	seq, err := u.repo.GetByID(ctx, cmd.TenantID, cmd.ID)
	if err != nil {
		return nil, err
	}

	if cmd.Name != "" {
		seq.Name = strings.TrimSpace(cmd.Name)
	}
	seq.Prefix = cmd.Prefix
	seq.Suffix = cmd.Suffix
	if cmd.Template != "" {
		seq.Template = cmd.Template
	}
	if cmd.Padding > 0 {
		seq.Padding = cmd.Padding
	}
	if cmd.ResetPolicy != "" {
		if !cmd.ResetPolicy.IsValid() {
			return nil, fmt.Errorf("%w: invalid reset_policy '%s'", shared.ErrInvalidInput, cmd.ResetPolicy)
		}
		seq.ResetPolicy = cmd.ResetPolicy
	}
	seq.IsActive = cmd.IsActive
	seq.UpdatedAt = time.Now().UTC()

	if err := u.repo.Update(ctx, seq); err != nil {
		return nil, err
	}

	return seq, nil
}

func (u *usecase) DeleteSequence(ctx context.Context, tenantID, id shared.ID) error {
	if tenantID == shared.NilID() {
		return shared.ErrTenantRequired
	}
	return u.repo.Delete(ctx, tenantID, id)
}

func (u *usecase) AcquireNextNumber(ctx context.Context, cmd AcquireNextCommand) (string, error) {
	if cmd.TenantID == shared.NilID() {
		return "", shared.ErrTenantRequired
	}

	entityType := strings.ToLower(strings.TrimSpace(cmd.EntityType))
	subType := strings.ToLower(strings.TrimSpace(cmd.SubType))

	return u.repo.AcquireNextNumber(ctx, cmd.TenantID, entityType, subType, cmd.ExtraTokens)
}

func (u *usecase) PreviewNextNumber(ctx context.Context, cmd PreviewCommand) (string, error) {
	if cmd.TenantID == shared.NilID() {
		return "", shared.ErrTenantRequired
	}

	entityType := strings.ToLower(strings.TrimSpace(cmd.EntityType))
	subType := strings.ToLower(strings.TrimSpace(cmd.SubType))

	seq, err := u.repo.GetByEntity(ctx, cmd.TenantID, entityType, subType)
	if err != nil {
		return "", err
	}

	now := time.Now().UTC()
	nextVal, _ := seq.NextValue(now)
	formatted := sequence.FormatNumber(seq.Template, seq.Prefix, seq.Suffix, nextVal, seq.Padding, now, cmd.ExtraTokens)

	return formatted, nil
}
