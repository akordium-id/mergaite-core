package attachment_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/akordium-id/mergiate-core/internal/core/domain/attachment"
	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
	usecase "github.com/akordium-id/mergiate-core/internal/core/usecase/attachment"
)

type mockStorageDriver struct {
	files map[string][]byte
}

func newMockStorageDriver() *mockStorageDriver {
	return &mockStorageDriver{files: make(map[string][]byte)}
}

func (m *mockStorageDriver) Put(ctx context.Context, path string, r io.Reader, size int64, contentType string) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	m.files[path] = data
	return nil
}

func (m *mockStorageDriver) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	data, ok := m.files[path]
	if !ok {
		return nil, errors.New("file not found in mock storage")
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (m *mockStorageDriver) Delete(ctx context.Context, path string) error {
	delete(m.files, path)
	return nil
}

func (m *mockStorageDriver) Exists(ctx context.Context, path string) (bool, error) {
	_, ok := m.files[path]
	return ok, nil
}

func (m *mockStorageDriver) GetURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	return "/api/v1/files/download?path=" + path, nil
}

type mockAttachmentRepo struct {
	files       map[shared.ID]*attachment.File
	attachments map[shared.ID]*attachment.EntityAttachment
}

func newMockAttachmentRepo() *mockAttachmentRepo {
	return &mockAttachmentRepo{
		files:       make(map[shared.ID]*attachment.File),
		attachments: make(map[shared.ID]*attachment.EntityAttachment),
	}
}

func (m *mockAttachmentRepo) CreateFile(ctx context.Context, file *attachment.File) error {
	m.files[file.ID] = file
	return nil
}

func (m *mockAttachmentRepo) GetFileByID(ctx context.Context, tenantID, id shared.ID) (*attachment.File, error) {
	f, ok := m.files[id]
	if !ok || f.TenantID != tenantID {
		return nil, shared.ErrNotFound
	}
	return f, nil
}

func (m *mockAttachmentRepo) GetFileByHash(ctx context.Context, tenantID shared.ID, hash string) (*attachment.File, error) {
	for _, f := range m.files {
		if f.TenantID == tenantID && f.SHA256Hash == hash {
			return f, nil
		}
	}
	return nil, shared.ErrNotFound
}

func (m *mockAttachmentRepo) ListFilesByTenant(ctx context.Context, tenantID shared.ID, limit, offset int32) ([]attachment.File, error) {
	var res []attachment.File
	for _, f := range m.files {
		if f.TenantID == tenantID {
			res = append(res, *f)
		}
	}
	return res, nil
}

func (m *mockAttachmentRepo) DeleteFile(ctx context.Context, tenantID, id shared.ID) error {
	delete(m.files, id)
	return nil
}

func (m *mockAttachmentRepo) CreateAttachment(ctx context.Context, att *attachment.EntityAttachment) error {
	m.attachments[att.ID] = att
	return nil
}

func (m *mockAttachmentRepo) GetAttachmentByID(ctx context.Context, tenantID, id shared.ID) (*attachment.EntityAttachment, error) {
	att, ok := m.attachments[id]
	if !ok || att.TenantID != tenantID {
		return nil, shared.ErrNotFound
	}
	return att, nil
}

func (m *mockAttachmentRepo) ListAttachmentsByEntity(ctx context.Context, tenantID shared.ID, entityType string, entityID shared.ID) ([]attachment.AttachmentWithFile, error) {
	var list []attachment.AttachmentWithFile
	for _, att := range m.attachments {
		if att.TenantID == tenantID && att.EntityType == entityType && att.EntityID == entityID {
			file := m.files[att.FileID]
			list = append(list, attachment.AttachmentWithFile{
				AttachmentID:  att.ID,
				TenantID:      att.TenantID,
				FileID:        att.FileID,
				EntityType:    att.EntityType,
				EntityID:      att.EntityID,
				Purpose:       att.Purpose,
				Title:         att.Title,
				SortOrder:     att.SortOrder,
				AttachedAt:    att.CreatedAt,
				StorageDriver: file.StorageDriver,
				StoragePath:   file.StoragePath,
				Filename:      file.Filename,
				MimeType:      file.MimeType,
				SizeBytes:     file.SizeBytes,
				SHA256Hash:    file.SHA256Hash,
				IsPublic:      file.IsPublic,
				FileCreatedAt: file.CreatedAt,
			})
		}
	}
	return list, nil
}

func (m *mockAttachmentRepo) DeleteAttachment(ctx context.Context, tenantID, id shared.ID) error {
	delete(m.attachments, id)
	return nil
}

func (m *mockAttachmentRepo) CountFileReferences(ctx context.Context, tenantID, fileID shared.ID) (int64, error) {
	var count int64
	for _, att := range m.attachments {
		if att.TenantID == tenantID && att.FileID == fileID {
			count++
		}
	}
	return count, nil
}

func TestAttachmentUsecase_UploadLifecycle(t *testing.T) {
	ctx := context.Background()
	driver := newMockStorageDriver()
	repo := newMockAttachmentRepo()
	uc := usecase.NewUsecase(repo, driver, 10*1024*1024)

	tenantID := shared.MustNewID()
	payload := []byte("%PDF-1.4 sample pdf content for quotation")
	hasher := sha256.New()
	hasher.Write(payload)
	expectedHash := hex.EncodeToString(hasher.Sum(nil))

	// 1. Upload File
	file, err := uc.UploadFile(ctx, usecase.UploadFileCommand{
		TenantID:  tenantID,
		Filename:  "quotation-scan.pdf",
		MimeType:  "application/pdf",
		Reader:    bytes.NewReader(payload),
		SizeBytes: int64(len(payload)),
	})
	if err != nil {
		t.Fatalf("UploadFile failed: %v", err)
	}

	if file.SHA256Hash != expectedHash {
		t.Errorf("expected hash %s, got %s", expectedHash, file.SHA256Hash)
	}
	if file.Filename != "quotation-scan.pdf" {
		t.Errorf("expected filename 'quotation-scan.pdf', got %s", file.Filename)
	}

	// 2. Read file back
	retrievedFile, rc, err := uc.GetFile(ctx, tenantID, file.ID)
	if err != nil {
		t.Fatalf("GetFile failed: %v", err)
	}
	defer rc.Close()

	if retrievedFile.ID != file.ID {
		t.Errorf("expected file ID %s, got %s", file.ID, retrievedFile.ID)
	}
	body, _ := io.ReadAll(rc)
	if string(body) != string(payload) {
		t.Errorf("payload mismatch: expected %q, got %q", string(payload), string(body))
	}
}

func TestAttachmentUsecase_Validation(t *testing.T) {
	ctx := context.Background()
	driver := newMockStorageDriver()
	repo := newMockAttachmentRepo()
	uc := usecase.NewUsecase(repo, driver, 1024) // 1KB limit

	tenantID := shared.MustNewID()

	// 1. Size exceeds limit
	largePayload := make([]byte, 2048)
	_, err := uc.UploadFile(ctx, usecase.UploadFileCommand{
		TenantID:  tenantID,
		Filename:  "toolarge.pdf",
		MimeType:  "application/pdf",
		Reader:    bytes.NewReader(largePayload),
		SizeBytes: int64(len(largePayload)),
	})
	if err == nil {
		t.Fatal("expected error for file exceeding size limit, got nil")
	}

	// 2. Disallowed MIME type
	_, err = uc.UploadFile(ctx, usecase.UploadFileCommand{
		TenantID:  tenantID,
		Filename:  "malicious.exe",
		MimeType:  "application/x-msdownload",
		Reader:    bytes.NewReader([]byte("MZ...")),
		SizeBytes: 5,
	})
	if err == nil {
		t.Fatal("expected error for unsupported MIME type, got nil")
	}
}

func TestAttachmentUsecase_AttachAndDetach(t *testing.T) {
	ctx := context.Background()
	driver := newMockStorageDriver()
	repo := newMockAttachmentRepo()
	uc := usecase.NewUsecase(repo, driver)

	tenantID := shared.MustNewID()
	docID := shared.MustNewID()

	// Upload file
	file, err := uc.UploadFile(ctx, usecase.UploadFileCommand{
		TenantID:  tenantID,
		Filename:  "contract.pdf",
		MimeType:  "application/pdf",
		Reader:    bytes.NewReader([]byte("contract terms")),
		SizeBytes: 14,
	})
	if err != nil {
		t.Fatalf("upload failed: %v", err)
	}

	// Attach to document
	att, err := uc.AttachFile(ctx, usecase.AttachFileCommand{
		TenantID:   tenantID,
		FileID:     file.ID,
		EntityType: "document",
		EntityID:   docID,
		Purpose:    "signed_contract",
		Title:      "Signed Sales Agreement",
		SortOrder:  1,
	})
	if err != nil {
		t.Fatalf("AttachFile failed: %v", err)
	}
	if att.Purpose != "signed_contract" {
		t.Errorf("expected purpose 'signed_contract', got %s", att.Purpose)
	}

	// List entity attachments
	list, err := uc.ListEntityAttachments(ctx, tenantID, "document", docID)
	if err != nil {
		t.Fatalf("ListEntityAttachments failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(list))
	}
	if list[0].Filename != "contract.pdf" {
		t.Errorf("expected joined filename 'contract.pdf', got %s", list[0].Filename)
	}

	// Detach
	err = uc.DetachFile(ctx, tenantID, att.ID)
	if err != nil {
		t.Fatalf("DetachFile failed: %v", err)
	}

	listAfter, _ := uc.ListEntityAttachments(ctx, tenantID, "document", docID)
	if len(listAfter) != 0 {
		t.Errorf("expected 0 attachments after detach, got %d", len(listAfter))
	}
}
