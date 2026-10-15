package lock_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/okf-memory/okf-agent-memory/pkg/lock"
)

func TestLockfile_ReadWriteUpsert(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "okf.lock")

	lf := &lock.Lockfile{Version: 1}
	entry := lock.BundleEntry{
		ID:          "peter/django-5-rules",
		Source:      "registry.okf-memory.dev/bundles/peter/django-5-rules",
		Version:     "1.0.0",
		Hash:        "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		InstalledAt: time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC),
	}
	lf.Upsert(entry)

	if err := lock.WriteLockfile(lockPath, lf); err != nil {
		t.Fatalf("WriteLockfile failed: %v", err)
	}

	loaded, err := lock.ReadLockfile(lockPath)
	if err != nil {
		t.Fatalf("ReadLockfile failed: %v", err)
	}
	if len(loaded.Bundles) != 1 {
		t.Fatalf("expected 1 bundle, got %d", len(loaded.Bundles))
	}
	if loaded.Bundles[0].ID != "peter/django-5-rules" {
		t.Errorf("expected bundle ID peter/django-5-rules, got %s", loaded.Bundles[0].ID)
	}
	if loaded.Bundles[0].Hash != entry.Hash {
		t.Errorf("expected hash %s, got %s", entry.Hash, loaded.Bundles[0].Hash)
	}

	// Test Remove
	removed := loaded.Remove("peter/django-5-rules")
	if !removed || len(loaded.Bundles) != 0 {
		t.Errorf("expected bundle to be removed")
	}
}
