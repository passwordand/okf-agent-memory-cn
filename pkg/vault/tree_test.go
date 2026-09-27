package vault_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/okf-memory/okf-agent-memory/pkg/vault"
)

func TestTreeSerializationAndParsing(t *testing.T) {
	tree := vault.NewTree()
	tree.Entries["knowledge/architecture/sync.md"] = vault.TreeEntry{
		BlobHash:      "a4f81c9e5b6c891f",
		Size:          1024,
		PlaintextHash: "e3b0c44298fc1c14",
		ModifiedAt:    time.Date(2026, 9, 14, 10, 0, 0, 0, time.UTC).Format(time.RFC3339),
	}
	tree.Entries["knowledge/project/overview.md"] = vault.TreeEntry{
		BlobHash:      "b82a3f1d41d8cd98",
		Size:          2048,
		PlaintextHash: "f1a2b3c4d5e67890",
		ModifiedAt:    time.Date(2026, 9, 14, 11, 0, 0, 0, time.UTC).Format(time.RFC3339),
	}

	data, err := tree.Serialize()
	if err != nil {
		t.Fatalf("unexpected serialize error: %v", err)
	}

	parsed, err := vault.ParseTree(data)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if parsed.Version != 1 {
		t.Fatalf("expected version 1, got %d", parsed.Version)
	}

	if len(parsed.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(parsed.Entries))
	}

	entry, ok := parsed.Entries["knowledge/architecture/sync.md"]
	if !ok {
		t.Fatalf("missing entry knowledge/architecture/sync.md")
	}
	if entry.BlobHash != "a4f81c9e5b6c891f" || entry.Size != 1024 || entry.PlaintextHash != "e3b0c44298fc1c14" {
		t.Fatalf("entry fields mismatch: %+v", entry)
	}
}

func TestTreeDeterministicHash(t *testing.T) {
	tree1 := vault.NewTree()
	tree1.Entries["b.md"] = vault.TreeEntry{BlobHash: "blobB", PlaintextHash: "plainB", Size: 20}
	tree1.Entries["a.md"] = vault.TreeEntry{BlobHash: "blobA", PlaintextHash: "plainA", Size: 10}

	tree2 := vault.NewTree()
	tree2.Entries["a.md"] = vault.TreeEntry{BlobHash: "blobA", PlaintextHash: "plainA", Size: 10}
	tree2.Entries["b.md"] = vault.TreeEntry{BlobHash: "blobB", PlaintextHash: "plainB", Size: 20}

	hash1, err := tree1.Hash()
	if err != nil {
		t.Fatalf("tree1 Hash error: %v", err)
	}

	hash2, err := tree2.Hash()
	if err != nil {
		t.Fatalf("tree2 Hash error: %v", err)
	}

	if hash1 != hash2 {
		t.Fatalf("expected deterministic hash regardless of insertion order:\nhash1: %s\nhash2: %s", hash1, hash2)
	}
}

func TestTreeHasChanged(t *testing.T) {
	tree := vault.NewTree()
	tree.Entries["doc.md"] = vault.TreeEntry{
		BlobHash:      "blob1",
		PlaintextHash: "hash-original",
		Size:          50,
	}

	// 1. Non-existent file -> changed
	if !tree.HasChanged("missing.md", "hash-original") {
		t.Fatalf("missing file should report as changed")
	}

	// 2. Same hash -> not changed
	if tree.HasChanged("doc.md", "hash-original") {
		t.Fatalf("identical plaintext hash should report as unchanged")
	}

	// 3. Different hash -> changed
	if !tree.HasChanged("doc.md", "hash-modified") {
		t.Fatalf("different plaintext hash should report as changed")
	}
}

func TestDiffTrees(t *testing.T) {
	base := vault.NewTree()
	base.Entries["unchanged.md"] = vault.TreeEntry{PlaintextHash: "p-same", BlobHash: "b-same"}
	base.Entries["modified.md"] = vault.TreeEntry{PlaintextHash: "p-old", BlobHash: "b-old"}
	base.Entries["deleted.md"] = vault.TreeEntry{PlaintextHash: "p-del", BlobHash: "b-del"}

	target := vault.NewTree()
	target.Entries["unchanged.md"] = vault.TreeEntry{PlaintextHash: "p-same", BlobHash: "b-same"}
	target.Entries["modified.md"] = vault.TreeEntry{PlaintextHash: "p-new", BlobHash: "b-new"}
	target.Entries["added.md"] = vault.TreeEntry{PlaintextHash: "p-add", BlobHash: "b-add"}

	diff := vault.DiffTrees(base, target)

	if len(diff.Added) != 1 || diff.Added[0] != "added.md" {
		t.Fatalf("unexpected diff Added: %v", diff.Added)
	}
	if len(diff.Modified) != 1 || diff.Modified[0] != "modified.md" {
		t.Fatalf("unexpected diff Modified: %v", diff.Modified)
	}
	if len(diff.Deleted) != 1 || diff.Deleted[0] != "deleted.md" {
		t.Fatalf("unexpected diff Deleted: %v", diff.Deleted)
	}
	if len(diff.Unchanged) != 1 || diff.Unchanged[0] != "unchanged.md" {
		t.Fatalf("unexpected diff Unchanged: %v", diff.Unchanged)
	}
}

func TestTreeValidation(t *testing.T) {
	// Bad JSON
	_, err := vault.ParseTree([]byte("{invalid-json"))
	if err == nil {
		t.Fatalf("expected error on invalid JSON")
	}

	// Bad version
	badVerJSON, _ := json.Marshal(map[string]interface{}{
		"version": 99,
		"entries": map[string]interface{}{},
	})
	_, err = vault.ParseTree(badVerJSON)
	if !errors.Is(err, vault.ErrUnsupportedTreeVersion) {
		t.Fatalf("expected ErrUnsupportedTreeVersion, got: %v", err)
	}
}
