package attachment

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/akordium-id/mergiate-core/internal/core/domain/attachment"
	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
	"github.com/akordium-id/mergiate-core/pkg/storage"
)

type UploadFileCommand struct {
	TenantID   shared.ID
	Filename   string
	MimeType   string
	Reader     io.Reader
	SizeBytes  int64
	UploadedBy *shared.ID
	IsPublic   bool
	Metadata   map[string]any
}

type AttachFileCommand struct {
	TenantID   shared.ID `json:"tenant_id"`
	FileID     shared.ID `json:"file_id"`
	EntityType string    `json:"entity_type"`
	EntityID   shared.ID `json:"entity_id"`
	Purpose    string    `json:"purpose"`
	Title      string    `json:"title"`
	SortOrder  int       `json:"sort_order"`
}

type Usecase interface {
	UploadFile(ctx context.Context, cmd UploadFileCommand) (*attachment.File, error)
	GetFile(ctx context.Context, tenantID, fileID shared.ID) (*attachment.File, io.ReadCloser, error)
	GetFileMetadata(ctx context.Context, tenantID, fileID shared.ID) (*attachment.File, error)
	DeleteFile(ctx context.Context, tenantID, fileID shared.ID) error
	AttachFile(ctx context.Context, cmd AttachFileCommand) (*attachment.EntityAttachment, error)
	ListEntityAttachments(ctx context.Context, tenantID shared.ID, entityType string, entityID shared.ID) ([]attachment.AttachmentWithFile, error)
	DetachFile(ctx context.Context, tenantID, attachmentID shared.ID) error
}

type usecase struct {
	repo          attachment.Repository
	storageDriver storage.Driver
	maxFileSize   int64
}

// NewUsecase constructs a File & Attachment usecase instance.
func NewUsecase(repo attachment.Repository, driver storage.Driver, maxFileSize ...int64) Usecase {
	maxSize := int64(attachment.DefaultMaxFileSize)
	if len(maxFileSize) > 0 && maxFileSize[0] > 0 {
		maxSize = maxFileSize[0]
	}
	return &usecase{
		repo:          repo,
		storageDriver: driver,
		maxFileSize:   maxSize,
	}
}

func (u *usecase) UploadFile(ctx context.Context, cmd UploadFileCommand) (*attachment.File, error) {
	if cmd.TenantID == shared.NilID() {
		return nil, shared.ErrTenantRequired
	}

	sanitizedName := attachment.SanitizeFilename(cmd.Filename)
	if sanitizedName == "" {
		return nil, fmt.Errorf("%w: filename cannot be empty", shared.ErrInvalidInput)
	}

	if cmd.SizeBytes > u.maxFileSize {
		return nil, fmt.Errorf("%w: file size exceeds maximum limit of %d bytes", shared.ErrInvalidInput, u.maxFileSize)
	}

	mime := strings.ToLower(strings.TrimSpace(cmd.MimeType))
	if !attachment.IsAllowedMIME(mime) {
		return nil, fmt.Errorf("%w: mime type '%s' is not allowed", shared.ErrInvalidInput, mime)
	}

	fileID, err := shared.NewID()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	ext := filepath.Ext(sanitizedName)
	storagePath := fmt.Sprintf("uploads/%s/%04d/%02d/%s%s", cmd.TenantID, now.Year(), now.Month(), fileID, ext)

	hasher := sha256.New()
	teeReader := io.TeeReader(cmd.Reader, hasher)

	// Stream file to storage driver
	err = u.storageDriver.Put(ctx, storagePath, teeReader, cmd.SizeBytes, mime)
	if err != nil {
		return nil, fmt.Errorf("failed to save file to storage: %w", err)
	}

	hashStr := hex.EncodeToString(hasher.Sum(nil))

	file := &attachment.File{
		ID:            fileID,
		TenantID:      cmd.TenantID,
		StorageDriver: "local",
		StoragePath:   storagePath,
		Filename:      sanitizedName,
		MimeType:      mime,
		SizeBytes:     cmd.SizeBytes,
		SHA256Hash:    hashStr,
		UploadedBy:    cmd.UploadedBy,
		IsPublic:      cmd.IsPublic,
		Metadata:      cmd.Metadata,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := u.repo.CreateFile(ctx, file); err != nil {
		// Compensating transaction: purge unreferenced physical file
		_ = u.storageDriver.Delete(ctx, storagePath)
		return nil, fmt.Errorf("failed to record file metadata: %w", err)
	}

	return file, nil
}

func (u *usecase) GetFile(ctx context.Context, tenantID, fileID shared.ID) (*attachment.File, io.ReadCloser, error) {
	file, err := u.repo.GetFileByID(ctx, tenantID, fileID)
	if err != nil {
		return nil, nil, err
	}

	rc, err := u.storageDriver.Get(ctx, file.StoragePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to retrieve file from storage: %w", err)
	}

	return file, rc, nil
}

func (u *usecase) GetFileMetadata(ctx context.Context, tenantID, fileID shared.ID) (*attachment.File, error) {
	return u.repo.GetFileByID(ctx, tenantID, fileID)
}

func (u *usecase) DeleteFile(ctx context.Context, tenantID, fileID shared.ID) error {
	file, err := u.repo.GetFileByID(ctx, tenantID, fileID)
	if err != nil {
		return err
	}

	// Delete from storage driver
	_ = u.storageDriver.Delete(ctx, file.StoragePath)

	// Delete from database (cascades to entity_attachments)
	return u.repo.DeleteFile(ctx, tenantID, fileID)
}

func (u *usecase) AttachFile(ctx context.Context, cmd AttachFileCommand) (*attachment.EntityAttachment, error) {
	if cmd.TenantID == shared.NilID() {
		return nil, shared.ErrTenantRequired
	}
	if cmd.FileID == shared.NilID() {
		return nil, fmt.Errorf("%w: file_id is required", shared.ErrInvalidInput)
	}
	if cmd.EntityType == "" {
		return nil, fmt.Errorf("%w: entity_type is required", shared.ErrInvalidInput)
	}
	if cmd.EntityID == shared.NilID() {
		return nil, fmt.Errorf("%w: entity_id is required", shared.ErrInvalidInput)
	}

	// Verify file exists in tenant
	_, err := u.repo.GetFileByID(ctx, cmd.TenantID, cmd.FileID)
	if err != nil {
		return nil, err
	}

	attID, err := shared.NewID()
	if err != nil {
		return nil, err
	}

	purpose := strings.TrimSpace(cmd.Purpose)
	if purpose == "" {
		purpose = attachment.PurposeAttachment
	}

	att := &attachment.EntityAttachment{
		ID:         attID,
		TenantID:   cmd.TenantID,
		FileID:     cmd.FileID,
		EntityType: strings.ToLower(cmd.EntityType),
		EntityID:   cmd.EntityID,
		Purpose:    purpose,
		Title:      cmd.Title,
		SortOrder:  cmd.SortOrder,
		CreatedAt:  time.Now().UTC(),
	}

	if err := u.repo.CreateAttachment(ctx, att); err != nil {
		return nil, err
	}

	return att, nil
}

func (u *usecase) ListEntityAttachments(ctx context.Context, tenantID shared.ID, entityType string, entityID shared.ID) ([]attachment.AttachmentWithFile, error) {
	if tenantID == shared.NilID() {
		return nil, shared.ErrTenantRequired
	}
	return u.repo.ListAttachmentsByEntity(ctx, tenantID, strings.ToLower(entityType), entityID)
}

func (u *usecase) DetachFile(ctx context.Context, tenantID, attachmentID shared.ID) error {
	if tenantID == shared.NilID() {
		return shared.ErrTenantRequired
	}
	return u.repo.DeleteAttachment(ctx, tenantID, attachmentID)
}
