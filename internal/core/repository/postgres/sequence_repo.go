package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/akordium-id/mergiate-core/internal/core/domain/sequence"
	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
	"github.com/akordium-id/mergiate-core/internal/core/repository/postgres/sqlc"
)

type sequenceRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

// NewSequenceRepository creates a PostgreSQL implementation of sequence.Repository.
func NewSequenceRepository(pool *pgxpool.Pool) sequence.Repository {
	return &sequenceRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

func (r *sequenceRepository) Create(ctx context.Context, seq *sequence.Sequence) error {
	var lastReset pgtype.Timestamptz
	if seq.LastResetAt != nil {
		lastReset = pgtype.Timestamptz{Time: *seq.LastResetAt, Valid: true}
	}

	var lastNum *string
	if seq.LastNumber != "" {
		lastNum = &seq.LastNumber
	}

	params := sqlc.CreateSequenceParams{
		ID:           shared.ToPgUUID(seq.ID),
		TenantID:     shared.ToPgUUID(seq.TenantID),
		Code:         seq.Code,
		Name:         seq.Name,
		EntityType:   seq.EntityType,
		SubType:      seq.SubType,
		Prefix:       seq.Prefix,
		Suffix:       seq.Suffix,
		Template:     seq.Template,
		Padding:      int32(seq.Padding),
		StartValue:   seq.StartValue,
		IncrementBy:  int32(seq.IncrementBy),
		CurrentValue: seq.CurrentValue,
		ResetPolicy:  string(seq.ResetPolicy),
		LastNumber:   lastNum,
		LastResetAt:  lastReset,
		IsActive:     seq.IsActive,
		CreatedAt:    pgtype.Timestamptz{Time: seq.CreatedAt, Valid: true},
		UpdatedAt:    pgtype.Timestamptz{Time: seq.UpdatedAt, Valid: true},
	}

	row, err := r.queries.CreateSequence(ctx, params)
	if err != nil {
		return err
	}

	seq.ID = shared.FromPgUUID(row.ID)
	seq.CreatedAt = row.CreatedAt.Time
	seq.UpdatedAt = row.UpdatedAt.Time
	return nil
}

func (r *sequenceRepository) GetByID(ctx context.Context, tenantID, id shared.ID) (*sequence.Sequence, error) {
	row, err := r.queries.GetSequenceByID(ctx, sqlc.GetSequenceByIDParams{
		TenantID: shared.ToPgUUID(tenantID),
		ID:       shared.ToPgUUID(id),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}

	return toDomainSequence(row), nil
}

func (r *sequenceRepository) GetByCode(ctx context.Context, tenantID shared.ID, code string) (*sequence.Sequence, error) {
	row, err := r.queries.GetSequenceByCode(ctx, sqlc.GetSequenceByCodeParams{
		TenantID: shared.ToPgUUID(tenantID),
		Code:     code,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}

	return toDomainSequence(row), nil
}

func (r *sequenceRepository) GetByEntity(ctx context.Context, tenantID shared.ID, entityType, subType string) (*sequence.Sequence, error) {
	row, err := r.queries.GetSequenceByEntity(ctx, sqlc.GetSequenceByEntityParams{
		TenantID:   shared.ToPgUUID(tenantID),
		EntityType: entityType,
		SubType:    subType,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}

	return toDomainSequence(row), nil
}

func (r *sequenceRepository) List(ctx context.Context, tenantID shared.ID) ([]sequence.Sequence, error) {
	rows, err := r.queries.ListSequences(ctx, shared.ToPgUUID(tenantID))
	if err != nil {
		return nil, err
	}

	sequences := make([]sequence.Sequence, len(rows))
	for i, row := range rows {
		sequences[i] = *toDomainSequence(row)
	}
	return sequences, nil
}

func (r *sequenceRepository) Update(ctx context.Context, seq *sequence.Sequence) error {
	params := sqlc.UpdateSequenceParams{
		TenantID:    shared.ToPgUUID(seq.TenantID),
		ID:          shared.ToPgUUID(seq.ID),
		Name:        seq.Name,
		Prefix:      seq.Prefix,
		Suffix:      seq.Suffix,
		Template:    seq.Template,
		Padding:     int32(seq.Padding),
		ResetPolicy: string(seq.ResetPolicy),
		IsActive:    seq.IsActive,
		UpdatedAt:   pgtype.Timestamptz{Time: seq.UpdatedAt, Valid: true},
	}

	row, err := r.queries.UpdateSequence(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return shared.ErrNotFound
		}
		return err
	}

	seq.UpdatedAt = row.UpdatedAt.Time
	return nil
}

func (r *sequenceRepository) Delete(ctx context.Context, tenantID, id shared.ID) error {
	return r.queries.DeleteSequence(ctx, sqlc.DeleteSequenceParams{
		TenantID: shared.ToPgUUID(tenantID),
		ID:       shared.ToPgUUID(id),
	})
}

// AcquireNextNumber locks the sequence row with SELECT ... FOR UPDATE within a transaction,
// safely computes the next incremented/reset sequence number, saves the state, and commits.
func (r *sequenceRepository) AcquireNextNumber(ctx context.Context, tenantID shared.ID, entityType, subType string, extraTokens map[string]string) (string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	row, err := qtx.GetSequenceForUpdate(ctx, sqlc.GetSequenceForUpdateParams{
		TenantID:   shared.ToPgUUID(tenantID),
		EntityType: entityType,
		SubType:    subType,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", shared.ErrNotFound
		}
		return "", err
	}

	seq := toDomainSequence(row)
	now := time.Now().UTC()
	nextVal, wasReset := seq.NextValue(now)
	formatted := sequence.FormatNumber(seq.Template, seq.Prefix, seq.Suffix, nextVal, seq.Padding, now, extraTokens)

	var lastReset pgtype.Timestamptz
	if wasReset || seq.LastResetAt == nil {
		lastReset = pgtype.Timestamptz{Time: now, Valid: true}
	} else {
		lastReset = row.LastResetAt
	}

	err = qtx.UpdateSequenceValue(ctx, sqlc.UpdateSequenceValueParams{
		TenantID:     shared.ToPgUUID(tenantID),
		ID:           row.ID,
		CurrentValue: nextVal,
		LastNumber:   &formatted,
		LastResetAt:  lastReset,
		UpdatedAt:    pgtype.Timestamptz{Time: now, Valid: true},
	})
	if err != nil {
		return "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", err
	}

	return formatted, nil
}

func toDomainSequence(row sqlc.NumberSequence) *sequence.Sequence {
	var lastReset *time.Time
	if row.LastResetAt.Valid {
		lastReset = &row.LastResetAt.Time
	}

	return &sequence.Sequence{
		ID:           shared.FromPgUUID(row.ID),
		TenantID:     shared.FromPgUUID(row.TenantID),
		Code:         row.Code,
		Name:         row.Name,
		EntityType:   row.EntityType,
		SubType:      row.SubType,
		Prefix:       row.Prefix,
		Suffix:       row.Suffix,
		Template:     row.Template,
		Padding:      int(row.Padding),
		StartValue:   row.StartValue,
		IncrementBy:  int(row.IncrementBy),
		CurrentValue: row.CurrentValue,
		ResetPolicy:  sequence.ResetPolicy(row.ResetPolicy),
		LastNumber:   strFromPtr(row.LastNumber),
		LastResetAt:  lastReset,
		IsActive:     row.IsActive,
		CreatedAt:    row.CreatedAt.Time,
		UpdatedAt:    row.UpdatedAt.Time,
	}
}
