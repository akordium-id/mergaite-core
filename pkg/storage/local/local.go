package local

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/akordium-id/mergiate-core/pkg/storage"
)

var (
	ErrPathTraversal = errors.New("path traversal detected")
	ErrFileNotFound  = errors.New("file not found in local storage")
)

type localDriver struct {
	baseDir string
}

// NewDriver constructs a local disk storage driver anchored at baseDir.
func NewDriver(baseDir string) (storage.Driver, error) {
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve storage base directory: %w", err)
	}

	if err := os.MkdirAll(absBase, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage base directory: %w", err)
	}

	return &localDriver{baseDir: absBase}, nil
}

func (d *localDriver) resolvePath(relPath string) (string, error) {
	cleanRel := filepath.Clean(relPath)
	if strings.HasPrefix(cleanRel, "/") || strings.HasPrefix(cleanRel, "\\") {
		cleanRel = cleanRel[1:]
	}

	fullPath := filepath.Join(d.baseDir, cleanRel)
	if !strings.HasPrefix(fullPath, d.baseDir) {
		return "", ErrPathTraversal
	}

	return fullPath, nil
}

func (d *localDriver) Put(ctx context.Context, relPath string, r io.Reader, size int64, contentType string) error {
	fullPath, err := d.resolvePath(relPath)
	if err != nil {
		return err
	}

	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write to temporary file first for atomic write
	tmpFile, err := os.CreateTemp(dir, "upload-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpName := tmpFile.Name()

	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpName)
	}()

	written, err := io.Copy(tmpFile, r)
	if err != nil {
		return fmt.Errorf("failed to write file content: %w", err)
	}

	if size > 0 && written != size {
		return fmt.Errorf("file size mismatch: expected %d, wrote %d", size, written)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Rename(tmpName, fullPath); err != nil {
		return fmt.Errorf("failed to commit file to %s: %w", fullPath, err)
	}

	return nil
}

func (d *localDriver) Get(ctx context.Context, relPath string) (io.ReadCloser, error) {
	fullPath, err := d.resolvePath(relPath)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrFileNotFound
		}
		return nil, fmt.Errorf("failed to open file %s: %w", fullPath, err)
	}

	return f, nil
}

func (d *localDriver) Delete(ctx context.Context, relPath string) error {
	fullPath, err := d.resolvePath(relPath)
	if err != nil {
		return err
	}

	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file %s: %w", fullPath, err)
	}

	return nil
}

func (d *localDriver) Exists(ctx context.Context, relPath string) (bool, error) {
	fullPath, err := d.resolvePath(relPath)
	if err != nil {
		return false, err
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return !info.IsDir(), nil
}

func (d *localDriver) GetURL(ctx context.Context, relPath string, expiry time.Duration) (string, error) {
	// For local storage, returns the relative API path
	return fmt.Sprintf("/api/v1/files/download?path=%s", relPath), nil
}
