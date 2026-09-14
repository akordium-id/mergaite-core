package sequence_test

import (
	"context"
	"errors"
	"testing"
	"time"

	domain "github.com/akordium-id/mergiate-core/internal/core/domain/sequence"
	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
	usecase "github.com/akordium-id/mergiate-core/internal/core/usecase/sequence"
)

type mockSequenceRepo struct {
	sequences map[shared.ID]*domain.Sequence
}

func newMockRepo() *mockSequenceRepo {
	return &mockSequenceRepo{
		sequences: make(map[shared.ID]*domain.Sequence),
	}
}

func (m *mockSequenceRepo) Create(ctx context.Context, seq *domain.Sequence) error {
	m.sequences[seq.ID] = seq
	return nil
}

func (m *mockSequenceRepo) GetByID(ctx context.Context, tenantID, id shared.ID) (*domain.Sequence, error) {
	seq, ok := m.sequences[id]
	if !ok || seq.TenantID != tenantID {
		return nil, shared.ErrNotFound
	}
	return seq, nil
}

func (m *mockSequenceRepo) GetByCode(ctx context.Context, tenantID shared.ID, code string) (*domain.Sequence, error) {
	for _, s := range m.sequences {
		if s.TenantID == tenantID && s.Code == code {
			return s, nil
		}
	}
	return nil, shared.ErrNotFound
}

func (m *mockSequenceRepo) GetByEntity(ctx context.Context, tenantID shared.ID, entityType, subType string) (*domain.Sequence, error) {
	for _, s := range m.sequences {
		if s.TenantID == tenantID && s.EntityType == entityType && s.SubType == subType && s.IsActive {
			return s, nil
		}
	}
	return nil, shared.ErrNotFound
}

func (m *mockSequenceRepo) List(ctx context.Context, tenantID shared.ID) ([]domain.Sequence, error) {
	var list []domain.Sequence
	for _, s := range m.sequences {
		if s.TenantID == tenantID {
			list = append(list, *s)
		}
	}
	return list, nil
}

func (m *mockSequenceRepo) Update(ctx context.Context, seq *domain.Sequence) error {
	m.sequences[seq.ID] = seq
	return nil
}

func (m *mockSequenceRepo) Delete(ctx context.Context, tenantID, id shared.ID) error {
	if _, ok := m.sequences[id]; !ok {
		return shared.ErrNotFound
	}
	delete(m.sequences, id)
	return nil
}

func (m *mockSequenceRepo) AcquireNextNumber(ctx context.Context, tenantID shared.ID, entityType, subType string, extraTokens map[string]string) (string, error) {
	seq, err := m.GetByEntity(ctx, tenantID, entityType, subType)
	if err != nil {
		return "", err
	}

	now := time.Now().UTC()
	nextVal, wasReset := seq.NextValue(now)
	formatted := domain.FormatNumber(seq.Template, seq.Prefix, seq.Suffix, nextVal, seq.Padding, now, extraTokens)

	seq.CurrentValue = nextVal
	seq.LastNumber = formatted
	if wasReset || seq.LastResetAt == nil {
		seq.LastResetAt = &now
	}

	return formatted, nil
}

func TestFormatNumber_Tokens(t *testing.T) {
	fixedTime := time.Date(2026, time.September, 14, 15, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		template string
		prefix   string
		suffix   string
		val      int64
		padding  int
		extra    map[string]string
		expected string
	}{
		{
			name:     "Standard Quotation Template",
			template: "{PREFIX}/{YYYY}/{MM}/{SEQ:4}",
			prefix:   "QUO",
			val:      1,
			padding:  4,
			expected: "QUO/2026/09/0001",
		},
		{
			name:     "Invoice with 2-digit year and 5 digits padding",
			template: "{PREFIX}-{YY}-{SEQ:5}",
			prefix:   "INV",
			val:      42,
			padding:  5,
			expected: "INV-26-00042",
		},
		{
			name:     "Purchase Order with Branch Code and Day",
			template: "{PREFIX}/{ORG}/{YYYY}{MM}{DD}/{SEQ:3}",
			prefix:   "PO",
			val:      7,
			padding:  3,
			extra:    map[string]string{"ORG": "JKT"},
			expected: "PO/JKT/20260914/007",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := domain.FormatNumber(tt.template, tt.prefix, tt.suffix, tt.val, tt.padding, fixedTime, tt.extra)
			if actual != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, actual)
			}
		})
	}
}

func TestSequence_NextValueAndResetPolicy(t *testing.T) {
	now := time.Date(2026, time.January, 1, 10, 0, 0, 0, time.UTC)
	lastYear := time.Date(2025, time.December, 31, 23, 59, 59, 0, time.UTC)

	seq := &domain.Sequence{
		StartValue:   1,
		IncrementBy:  1,
		CurrentValue: 550,
		ResetPolicy:  domain.ResetYearly,
		LastResetAt:  &lastYear,
	}

	nextVal, wasReset := seq.NextValue(now)
	if !wasReset {
		t.Error("expected yearly reset to trigger")
	}
	if nextVal != 1 {
		t.Errorf("expected reset to start_value 1, got %d", nextVal)
	}
}

func TestSequenceUsecase_CreateAndAcquire(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepo()
	uc := usecase.NewUsecase(repo)
	tenantID := shared.MustNewID()

	// 1. Create Sequence
	seq, err := uc.CreateSequence(ctx, usecase.CreateSequenceCommand{
		TenantID:    tenantID,
		Code:        "seq_quotation",
		Name:        "Quotation Numbering",
		EntityType:  "document",
		SubType:     "quotation",
		Prefix:      "QUO",
		Template:    "{PREFIX}/{YYYY}/{SEQ:4}",
		Padding:     4,
		StartValue:  1,
		IncrementBy: 1,
		ResetPolicy: domain.ResetYearly,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if seq.Code != "seq_quotation" {
		t.Errorf("expected code 'seq_quotation', got '%s'", seq.Code)
	}

	// 2. Duplicate Code error
	_, err = uc.CreateSequence(ctx, usecase.CreateSequenceCommand{
		TenantID:   tenantID,
		Code:       "seq_quotation",
		Name:       "Duplicate",
		EntityType: "document",
		SubType:    "quotation",
	})
	if !errors.Is(err, shared.ErrAlreadyExists) {
		t.Errorf("expected ErrAlreadyExists, got %v", err)
	}

	// 3. Acquire First Number
	num1, err := uc.AcquireNextNumber(ctx, usecase.AcquireNextCommand{
		TenantID:   tenantID,
		EntityType: "document",
		SubType:    "quotation",
	})
	if err != nil {
		t.Fatalf("unexpected error acquiring number: %v", err)
	}

	yearStr := time.Now().Format("2006")
	expectedNum1 := "QUO/" + yearStr + "/0001"
	if num1 != expectedNum1 {
		t.Errorf("expected '%s', got '%s'", expectedNum1, num1)
	}

	// 4. Acquire Second Number
	num2, err := uc.AcquireNextNumber(ctx, usecase.AcquireNextCommand{
		TenantID:   tenantID,
		EntityType: "document",
		SubType:    "quotation",
	})
	if err != nil {
		t.Fatalf("unexpected error acquiring number: %v", err)
	}

	expectedNum2 := "QUO/" + yearStr + "/0002"
	if num2 != expectedNum2 {
		t.Errorf("expected '%s', got '%s'", expectedNum2, num2)
	}
}
