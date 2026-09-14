package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/akordium-id/mergiate-core/internal/core/domain/attachment"
	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
	"github.com/akordium-id/mergiate-core/internal/core/repository/postgres/sqlc"
)

type attachmentRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

// NewAttachmentRepository constructs a PostgreSQL implementation of attachment.Repository.
func NewAttachmentRepository(pool *pgxpool.Pool) attachment.Repository {
	return &attachmentRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

func (r *attachmentRepository) CreateFile(ctx context.Context, f *attachment.File) error {
	metaJSON, err := json.Marshal(f.Metadata)
	if err != nil {
		metaJSON = []byte("{}")
	}

	var uploadedBy pgtype.UUID
	if f.UploadedBy != nil && *f.UploadedBy != shared.NilID() {
		uploadedBy = shared.ToPgUUID(*f.UploadedBy)
	}

	res, err := r.queries.CreateFile(ctx, sqlc.CreateFileParams{
		ID:            shared.ToPgUUID(f.ID),
		TenantID:      shared.ToPgUUID(f.TenantID),
		StorageDriver: f.StorageDriver,
		StoragePath:   f.StoragePath,
		Filename:      f.Filename,
		MimeType:      f.MimeType,
		SizeBytes:     f.SizeBytes,
		Sha256Hash:    f.SHA256Hash,
		UploadedBy:    uploadedBy,
		IsPublic:      f.IsPublic,
		Metadata:      metaJSON,
	})
	if err != nil {
		return err
	}

	f.CreatedAt = res.CreatedAt.Time
	f.UpdatedAt = res.UpdatedAt.Time
	return nil
}

func (r *attachmentRepository) GetFileByID(ctx context.Context, tenantID, id shared.ID) (*attachment.File, error) {
	row, err := r.queries.GetFileByID(ctx, sqlc.GetFileByIDParams{
		TenantID: shared.ToPgUUID(tenantID),
		ID:       shared.ToPgUUID(id),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}

	return toDomainFile(row), nil
}

func (r *attachmentRepository) GetFileByHash(ctx context.Context, tenantID shared.ID, hash string) (*attachment.File, error) {
	row, err := r.queries.GetFileByHash(ctx, sqlc.GetFileByHashParams{
		TenantID:   shared.ToPgUUID(tenantID),
		Sha256Hash: hash,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}

	return toDomainFile(row), nil
}

func (r *attachmentRepository) ListFilesByTenant(ctx context.Context, tenantID shared.ID, limit, offset int32) ([]attachment.File, error) {
	rows, err := r.queries.ListFilesByTenant(ctx, sqlc.ListFilesByTenantParams{
		TenantID: shared.ToPgUUID(tenantID),
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return nil, err
	}

	files := make([]attachment.File, len(rows))
	for i, row := range rows {
		files[i] = *toDomainFile(row)
	}
	return files, nil
}

func (r *attachmentRepository) DeleteFile(ctx context.Context, tenantID, id shared.ID) error {
	return r.queries.DeleteFile(ctx, sqlc.DeleteFileParams{
		TenantID: shared.ToPgUUID(tenantID),
		ID:       shared.ToPgUUID(id),
	})
}

func (r *attachmentRepository) CreateAttachment(ctx context.Context, att *attachment.EntityAttachment) error {
	res, err := r.queries.CreateAttachment(ctx, sqlc.CreateAttachmentParams{
		ID:         shared.ToPgUUID(att.ID),
		TenantID:   shared.ToPgUUID(att.TenantID),
		FileID:     shared.ToPgUUID(att.FileID),
		EntityType: att.EntityType,
		EntityID:   shared.ToPgUUID(att.EntityID),
		Purpose:    att.Purpose,
		Title:      att.Title,
		SortOrder:  int32(att.SortOrder),
	})
	if err != nil {
		return err
	}

	att.CreatedAt = res.CreatedAt.Time
	return nil
}

func (r *attachmentRepository) GetAttachmentByID(ctx context.Context, tenantID, id shared.ID) (*attachment.EntityAttachment, error) {
	row, err := r.queries.GetAttachmentByID(ctx, sqlc.GetAttachmentByIDParams{
		TenantID: shared.ToPgUUID(tenantID),
		ID:       shared.ToPgUUID(id),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}

	return &attachment.EntityAttachment{
		ID:         shared.FromPgUUID(row.ID),
		TenantID:   shared.FromPgUUID(row.TenantID),
		FileID:     shared.FromPgUUID(row.FileID),
		EntityType: row.EntityType,
		EntityID:   shared.FromPgUUID(row.EntityID),
		Purpose:    row.Purpose,
		Title:      row.Title,
		SortOrder:  int(row.SortOrder),
		CreatedAt:  row.CreatedAt.Time,
	}, nil
}

func (r *attachmentRepository) ListAttachmentsByEntity(ctx context.Context, tenantID shared.ID, entityType string, entityID shared.ID) ([]attachment.AttachmentWithFile, error) {
	rows, err := r.queries.ListAttachmentsByEntity(ctx, sqlc.ListAttachmentsByEntityParams{
		TenantID:   shared.ToPgUUID(tenantID),
		EntityType: entityType,
		EntityID:   shared.ToPgUUID(entityID),
	})
	if err != nil {
		return nil, err
	}

	results := make([]attachment.AttachmentWithFile, len(rows))
	for i, row := range rows {
		var uploader *shared.ID
		if row.UploadedBy.Valid {
			id := shared.FromPgUUID(row.UploadedBy)
			uploader = &id
		}

		var meta map[string]any
		if len(row.FileMetadata) > 0 {
			_ = json.Unmarshal(row.FileMetadata, &meta)
		}

		results[i] = attachment.AttachmentWithFile{
			AttachmentID:  shared.FromPgUUID(row.AttachmentID),
			TenantID:      shared.FromPgUUID(row.TenantID),
			FileID:        shared.FromPgUUID(row.FileID),
			EntityType:    row.EntityType,
			EntityID:      shared.FromPgUUID(row.EntityID),
			Purpose:       row.Purpose,
			Title:         row.Title,
			SortOrder:     int(row.SortOrder),
			AttachedAt:    row.AttachedAt.Time,
			StorageDriver: row.StorageDriver,
			StoragePath:   row.StoragePath,
			Filename:      row.Filename,
			MimeType:      row.MimeType,
			SizeBytes:     row.SizeBytes,
			SHA256Hash:    row.Sha256Hash,
			UploadedBy:    uploader,
			IsPublic:      row.IsPublic,
			FileMetadata:  meta,
			FileCreatedAt: row.FileCreatedAt.Time,
		}
	}

	return results, nil
}

func (r *attachmentRepository) DeleteAttachment(ctx context.Context, tenantID, id shared.ID) error {
	return r.queries.DeleteAttachment(ctx, sqlc.DeleteAttachmentParams{
		TenantID: shared.ToPgUUID(tenantID),
		ID:       shared.ToPgUUID(id),
	})
}

func (r *attachmentRepository) CountFileReferences(ctx context.Context, tenantID, fileID shared.ID) (int64, error) {
	return r.queries.CountFileReferences(ctx, sqlc.CountFileReferencesParams{
		TenantID: shared.ToPgUUID(tenantID),
		FileID:   shared.ToPgUUID(fileID),
	})
}

func toDomainFile(row sqlc.File) *attachment.File {
	var uploader *shared.ID
	if row.UploadedBy.Valid {
		id := shared.FromPgUUID(row.UploadedBy)
		uploader = &id
	}

	var meta map[string]any
	if len(row.Metadata) > 0 {
		_ = json.Unmarshal(row.Metadata, &meta)
	}

	return &attachment.File{
		ID:            shared.FromPgUUID(row.ID),
		TenantID:      shared.FromPgUUID(row.TenantID),
		StorageDriver: row.StorageDriver,
		StoragePath:   row.StoragePath,
		Filename:      row.Filename,
		MimeType:      row.MimeType,
		SizeBytes:     row.SizeBytes,
		SHA256Hash:    row.Sha256Hash,
		UploadedBy:    uploader,
		IsPublic:      row.IsPublic,
		Metadata:      meta,
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
	}
}
