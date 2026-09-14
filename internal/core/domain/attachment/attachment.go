package attachment

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
)

const (
	PurposeAttachment   = "attachment"
	PurposePrimaryImage = "primary_image"
	PurposeTaxInvoice   = "tax_invoice"
	PurposeContract     = "contract"
	PurposeScan         = "scan"

	DefaultMaxFileSize = 25 * 1024 * 1024 // 25 MB
)

// AllowedDefaultMIMETypes defines standard accepted business document & media formats.
var AllowedDefaultMIMETypes = map[string]bool{
	"application/pdf":    true,
	"image/jpeg":         true,
	"image/png":          true,
	"image/webp":         true,
	"image/gif":          true,
	"text/plain":         true,
	"text/csv":           true,
	"application/json":   true,
	"application/zip":    true,
	"application/msword": true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	"application/vnd.ms-excel": true,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         true,
}

// File represents a physical file object stored in local disk or S3 object storage.
type File struct {
	ID            shared.ID      `json:"id"`
	TenantID      shared.ID      `json:"tenant_id"`
	StorageDriver string         `json:"storage_driver"`
	StoragePath   string         `json:"storage_path"`
	Filename      string         `json:"filename"`
	MimeType      string         `json:"mime_type"`
	SizeBytes     int64          `json:"size_bytes"`
	SHA256Hash    string         `json:"sha256_hash"`
	UploadedBy    *shared.ID     `json:"uploaded_by,omitempty"`
	IsPublic      bool           `json:"is_public"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

// EntityAttachment links a physical file to a specific domain entity with contextual purpose.
type EntityAttachment struct {
	ID         shared.ID `json:"id"`
	TenantID   shared.ID `json:"tenant_id"`
	FileID     shared.ID `json:"file_id"`
	EntityType string    `json:"entity_type"`
	EntityID   shared.ID `json:"entity_id"`
	Purpose    string    `json:"purpose"`
	Title      string    `json:"title"`
	SortOrder  int       `json:"sort_order"`
	CreatedAt  time.Time `json:"created_at"`
}

// AttachmentWithFile contains joined entity attachment and physical file metadata.
type AttachmentWithFile struct {
	AttachmentID  shared.ID      `json:"attachment_id"`
	TenantID      shared.ID      `json:"tenant_id"`
	FileID        shared.ID      `json:"file_id"`
	EntityType    string         `json:"entity_type"`
	EntityID      shared.ID      `json:"entity_id"`
	Purpose       string         `json:"purpose"`
	Title         string         `json:"title"`
	SortOrder     int            `json:"sort_order"`
	AttachedAt    time.Time      `json:"attached_at"`
	StorageDriver string         `json:"storage_driver"`
	StoragePath   string         `json:"storage_path"`
	Filename      string         `json:"filename"`
	MimeType      string         `json:"mime_type"`
	SizeBytes     int64          `json:"size_bytes"`
	SHA256Hash    string         `json:"sha256_hash"`
	UploadedBy    *shared.ID     `json:"uploaded_by,omitempty"`
	IsPublic      bool           `json:"is_public"`
	FileMetadata  map[string]any `json:"file_metadata,omitempty"`
	FileCreatedAt time.Time      `json:"file_created_at"`
}

// SanitizeFilename cleans and strips unsafe path characters from upload filename.
func SanitizeFilename(name string) string {
	base := filepath.Base(name)
	base = strings.ReplaceAll(base, "\\", "_")
	base = strings.ReplaceAll(base, "/", "_")
	base = strings.ReplaceAll(base, "..", "_")
	base = strings.TrimSpace(base)
	if base == "" || base == "." {
		return "unnamed-file"
	}
	return base
}

// IsAllowedMIME checks if a given MIME type is supported.
func IsAllowedMIME(mime string) bool {
	clean := strings.ToLower(strings.TrimSpace(mime))
	if idx := strings.Index(clean, ";"); idx != -1 {
		clean = strings.TrimSpace(clean[:idx])
	}
	return AllowedDefaultMIMETypes[clean]
}
