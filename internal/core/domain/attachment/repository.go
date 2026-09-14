package attachment

import (
	"context"

	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
)

// Repository defines data access operations for physical files and polymorphic entity attachments.
type Repository interface {
	// File operations
	CreateFile(ctx context.Context, file *File) error
	GetFileByID(ctx context.Context, tenantID, id shared.ID) (*File, error)
	GetFileByHash(ctx context.Context, tenantID shared.ID, hash string) (*File, error)
	ListFilesByTenant(ctx context.Context, tenantID shared.ID, limit, offset int32) ([]File, error)
	DeleteFile(ctx context.Context, tenantID, id shared.ID) error

	// Attachment operations
	CreateAttachment(ctx context.Context, att *EntityAttachment) error
	GetAttachmentByID(ctx context.Context, tenantID, id shared.ID) (*EntityAttachment, error)
	ListAttachmentsByEntity(ctx context.Context, tenantID shared.ID, entityType string, entityID shared.ID) ([]AttachmentWithFile, error)
	DeleteAttachment(ctx context.Context, tenantID, id shared.ID) error
	CountFileReferences(ctx context.Context, tenantID, fileID shared.ID) (int64, error)
}
