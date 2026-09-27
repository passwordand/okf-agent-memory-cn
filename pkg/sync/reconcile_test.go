package sync_test

import (
	"testing"

	"github.com/okf-memory/okf-agent-memory/pkg/sync"
	"github.com/okf-memory/okf-agent-memory/pkg/vault"
)

func TestReconcile_DisjointMerge(t *testing.T) {
	// Base has f1, f2, f3
	base := vault.NewTree()
	base.Entries["f1.md"] = vault.TreeEntry{BlobHash: "b1_orig", PlaintextHash: "p1_orig", Size: 10}
	base.Entries["f2.md"] = vault.TreeEntry{BlobHash: "b2_orig", PlaintextHash: "p2_orig", Size: 20}
	base.Entries["f3.md"] = vault.TreeEntry{BlobHash: "b3_orig", PlaintextHash: "p3_orig", Size: 30}

	// Local modifies f1 and adds f4_local
	local := vault.NewTree()
	local.Entries["f1.md"] = vault.TreeEntry{BlobHash: "b1_local", PlaintextHash: "p1_local", Size: 15}
	local.Entries["f2.md"] = vault.TreeEntry{BlobHash: "b2_orig", PlaintextHash: "p2_orig", Size: 20}
	local.Entries["f3.md"] = vault.TreeEntry{BlobHash: "b3_orig", PlaintextHash: "p3_orig", Size: 30}
	local.Entries["f4_local.md"] = vault.TreeEntry{BlobHash: "b4_local", PlaintextHash: "p4_local", Size: 40}

	// Remote modifies f2 and deletes f3
	remote := vault.NewTree()
	remote.Entries["f1.md"] = vault.TreeEntry{BlobHash: "b1_orig", PlaintextHash: "p1_orig", Size: 10}
	remote.Entries["f2.md"] = vault.TreeEntry{BlobHash: "b2_remote", PlaintextHash: "p2_remote", Size: 25}

	result, err := sync.Reconcile(base, local, remote)
	if err != nil {
		t.Fatalf("unexpected Reconcile error: %v", err)
	}

	if len(result.Conflicts) != 0 {
		t.Fatalf("expected 0 conflicts for disjoint changes, got: %v", result.Conflicts)
	}

	merged := result.MergedTree
	if len(merged.Entries) != 3 {
		t.Fatalf("expected 3 entries in merged tree (f1, f2, f4_local), got %d: %+v", len(merged.Entries), merged.Entries)
	}

	// f1 must be local version
	if merged.Entries["f1.md"].BlobHash != "b1_local" {
		t.Fatalf("f1 should be local version, got: %+v", merged.Entries["f1.md"])
	}

	// f2 must be remote version
	if merged.Entries["f2.md"].BlobHash != "b2_remote" {
		t.Fatalf("f2 should be remote version, got: %+v", merged.Entries["f2.md"])
	}

	// f3 must be deleted
	if _, exists := merged.Entries["f3.md"]; exists {
		t.Fatalf("f3 should be deleted in merged tree")
	}

	// f4_local must be added
	if merged.Entries["f4_local.md"].BlobHash != "b4_local" {
		t.Fatalf("f4_local should be added, got: %+v", merged.Entries["f4_local.md"])
	}
}

func TestReconcile_IdenticalConcurrentModifications(t *testing.T) {
	base := vault.NewTree()
	base.Entries["f1.md"] = vault.TreeEntry{BlobHash: "b1_orig", PlaintextHash: "p1_orig"}

	local := vault.NewTree()
	local.Entries["f1.md"] = vault.TreeEntry{BlobHash: "b1_same", PlaintextHash: "p1_same"}

	remote := vault.NewTree()
	remote.Entries["f1.md"] = vault.TreeEntry{BlobHash: "b1_same", PlaintextHash: "p1_same"}

	result, err := sync.Reconcile(base, local, remote)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Conflicts) != 0 {
		t.Fatalf("expected 0 conflicts for identical edits, got %d", len(result.Conflicts))
	}
	if result.MergedTree.Entries["f1.md"].BlobHash != "b1_same" {
		t.Fatalf("merged tree f1 mismatch")
	}
}

func TestReconcile_CollisionDetectionAndForking(t *testing.T) {
	base := vault.NewTree()
	base.Entries["ideas.md"] = vault.TreeEntry{BlobHash: "b_orig", PlaintextHash: "p_orig"}

	local := vault.NewTree()
	local.Entries["ideas.md"] = vault.TreeEntry{BlobHash: "b_local", PlaintextHash: "p_local"}

	remote := vault.NewTree()
	remote.Entries["ideas.md"] = vault.TreeEntry{BlobHash: "b_remote", PlaintextHash: "p_remote"}

	result, err := sync.Reconcile(base, local, remote)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(result.Conflicts))
	}

	c := result.Conflicts[0]
	if c.Path != "ideas.md" {
		t.Fatalf("expected conflict on ideas.md, got %q", c.Path)
	}
	if c.LocalEntry.BlobHash != "b_local" || c.RemoteEntry.BlobHash != "b_remote" {
		t.Fatalf("conflict entries mismatch: %+v", c)
	}

	// Apply CLI failsafe: remote replaces ideas.md, local is preserved as ideas.conflict-local.md
	sync.ApplyConflictFailsafe(result.MergedTree, result.Conflicts)

	if result.MergedTree.Entries["ideas.md"].BlobHash != "b_remote" {
		t.Fatalf("ideas.md should contain remote version")
	}

	forkedPath := "ideas.conflict-local.md"
	if result.MergedTree.Entries[forkedPath].BlobHash != "b_local" {
		t.Fatalf("ideas.conflict-local.md should contain local version, got: %+v", result.MergedTree.Entries[forkedPath])
	}
}

func TestConflictLocalPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"notes.md", "notes.conflict-local.md"},
		{"docs/architecture/spec.md", "docs/architecture/spec.conflict-local.md"},
		{"unformatted", "unformatted.conflict-local"},
		{"archive.tar.gz", "archive.tar.conflict-local.gz"},
	}

	for _, tc := range tests {
		got := sync.ConflictLocalPath(tc.input)
		if got != tc.expected {
			t.Errorf("ConflictLocalPath(%q) = %q; want %q", tc.input, got, tc.expected)
		}
	}
}
