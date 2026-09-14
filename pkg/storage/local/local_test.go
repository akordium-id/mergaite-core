package local_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	"github.com/akordium-id/mergiate-core/pkg/storage/local"
)

func TestLocalDriver_LifecycleAndSecurity(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "storage-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	driver, err := local.NewDriver(tempDir)
	if err != nil {
		t.Fatalf("failed to initialize local driver: %v", err)
	}

	ctx := context.Background()
	relPath := "tenant1/documents/2026/09/sample.txt"
	content := []byte("Mergiate Core Enterprise Document Content")

	// 1. Put
	err = driver.Put(ctx, relPath, bytes.NewReader(content), int64(len(content)), "text/plain")
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// 2. Exists
	exists, err := driver.Exists(ctx, relPath)
	if err != nil || !exists {
		t.Fatalf("expected file to exist, got exists=%v, err=%v", exists, err)
	}

	// 3. Get
	rc, err := driver.Get(ctx, relPath)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	defer rc.Close()

	readBytes, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	if string(readBytes) != string(content) {
		t.Fatalf("content mismatch: expected %q, got %q", string(content), string(readBytes))
	}

	// 4. Path Traversal Defense
	badPath := "../../../etc/passwd"
	err = driver.Put(ctx, badPath, bytes.NewReader([]byte("hacked")), 6, "text/plain")
	if err != local.ErrPathTraversal {
		t.Fatalf("expected ErrPathTraversal for bad path, got: %v", err)
	}

	// 5. Delete
	err = driver.Delete(ctx, relPath)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	exists, err = driver.Exists(ctx, relPath)
	if err != nil || exists {
		t.Fatalf("expected file to not exist after delete, got exists=%v", exists)
	}

	// 6. Get non-existent
	_, err = driver.Get(ctx, relPath)
	if err != local.ErrFileNotFound {
		t.Fatalf("expected ErrFileNotFound, got: %v", err)
	}
}
