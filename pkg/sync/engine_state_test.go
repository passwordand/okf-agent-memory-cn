package sync_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/okf-memory/okf-agent-memory/pkg/sync"
	"github.com/okf-memory/okf-agent-memory/pkg/vault"
)

// Regression tests for issue #51: the sync engine must persist its last-synced
// head and tree inside the bundle so that a new engine instance (i.e. a new
// `okf hub push|pull|sync` process) resumes from that position instead of
// starting from an empty state. Without persistence the second push is always
// rejected with HTTP 409 (head_mismatch) and a subsequent sync moves local
// edits into "*.conflict-local.md" even without any remote writer.

func testVaultKey(b byte) []byte {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	key[0] = b
	return key
}

func TestEngine_PushAcrossEngineInstances_SecondPushNotConflict(t *testing.T) {
	server := sync.NewServer("")
	ctx := context.Background()
	vaultID := "test-vault-cross-push"
	vaultKey := testVaultKey(7)

	dir := t.TempDir()
	indexPath := filepath.Join(dir, "index.md")
	_ = os.WriteFile(indexPath, []byte("# Knowledge Index\nokf_version: 0.2\n"), 0o644)

	author := vault.CommitAuthor{ClientID: "client-1", Agent: "agent"}

	// First invocation: initial push succeeds (no remote head yet).
	engine1 := sync.NewEngine(newTestClient(server.Handler(), "token-1"), vaultID, vaultKey, dir)
	res1, err := engine1.Push(ctx, author, "First push")
	if err != nil {
		t.Fatalf("first engine push error: %v", err)
	}

	// Second invocation, same bundle, fresh engine = fresh process semantics.
	// It must resume from the persisted head; otherwise the hub CAS rejects
	// the commit with 409 head_mismatch.
	engine2 := sync.NewEngine(newTestClient(server.Handler(), "token-2"), vaultID, vaultKey, dir)
	res2, err := engine2.Push(ctx, author, "Second push from a fresh engine")
	if err != nil {
		t.Fatalf("second engine push failed (regression: expected non-409 success): %v", err)
	}

	if res2.CommitHash == "" || res2.CommitHash == res1.CommitHash {
		t.Fatalf("expected a new distinct commit hash, got %q (first %q)", res2.CommitHash, res1.CommitHash)
	}

	// Incremental upload: index.md did not change between the two pushes, so
	// the resumed state must mark it unchanged instead of re-encrypting and
	// re-uploading everything.
	if res2.UnchangedBlobs != 1 {
		t.Fatalf("expected 1 unchanged blob on second push (state resumed), got %d", res2.UnchangedBlobs)
	}

	// The persisted state must never be scanned as bundle content.
	scanned, err := engine2.ScanBundle()
	if err != nil {
		t.Fatalf("ScanBundle error: %v", err)
	}
	for path := range scanned {
		if strings.HasPrefix(path, ".okf/") {
			t.Fatalf("internal state file %s leaked into bundle scan", path)
		}
	}
}

func TestEngine_SyncAcrossEngineInstances_LocalEditsKept(t *testing.T) {
	server := sync.NewServer("")
	ctx := context.Background()
	vaultID := "test-vault-cross-sync"
	vaultKey := testVaultKey(11)

	dir := t.TempDir()
	indexPath := filepath.Join(dir, "index.md")
	original := "# Knowledge Index\nokf_version: 0.2\n"
	_ = os.WriteFile(indexPath, []byte(original), 0o644)

	author := vault.CommitAuthor{ClientID: "client-1", Agent: "agent"}

	// First invocation pushes the base version.
	engine1 := sync.NewEngine(newTestClient(server.Handler(), "token-1"), vaultID, vaultKey, dir)
	if _, err := engine1.Push(ctx, author, "Initial push"); err != nil {
		t.Fatalf("first engine push error: %v", err)
	}

	// New invocation edits a file it previously pushed, then syncs.
	// With no persisted base tree the empty base makes the edit look like an
	// incompatible concurrent modification and the sync moves it to
	// index.conflict-local.md, reverting index.md on a single machine.
	edited := "# Knowledge Index\nokf_version: 0.2\nlocal edit kept\n"
	_ = os.WriteFile(indexPath, []byte(edited), 0o644)

	engine2 := sync.NewEngine(newTestClient(server.Handler(), "token-2"), vaultID, vaultKey, dir)
	syncRes, err := engine2.Sync(ctx, author, "Local edit from a fresh engine")
	if err != nil {
		t.Fatalf("second engine sync failed (regression: expected non-409 success): %v", err)
	}

	if len(syncRes.Conflicts) != 0 {
		t.Fatalf("expected 0 conflicts for a local edit on a single machine, got: %v", syncRes.Conflicts)
	}

	got, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("failed to read index.md after sync: %v", err)
	}
	if string(got) != edited {
		t.Fatalf("local edit was reverted by sync:\n got %q\nwant %q", string(got), edited)
	}

	if _, err := os.Stat(filepath.Join(dir, "index.conflict-local.md")); !os.IsNotExist(err) {
		t.Fatalf("index.conflict-local.md must not exist for a clean single-machine edit")
	}
}

func TestEngine_SyncStateFile_CorruptStateFallsBackCleanly(t *testing.T) {
	server := sync.NewServer("")
	ctx := context.Background()
	vaultID := "test-vault-corrupt-state"
	vaultKey := testVaultKey(13)

	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "index.md"), []byte("# Index\n"), 0o644)

	author := vault.CommitAuthor{ClientID: "client-1", Agent: "agent"}
	engine1 := sync.NewEngine(newTestClient(server.Handler(), "token-1"), vaultID, vaultKey, dir)
	if _, err := engine1.Push(ctx, author, "Initial push"); err != nil {
		t.Fatalf("first engine push error: %v", err)
	}

	statePath := filepath.Join(dir, ".okf", "sync-state.json")
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("expected persisted sync state at %s: %v", statePath, err)
	}

	// Corrupt the state file: a new engine must not crash and must degrade to
	// empty-state behavior.
	if err := os.WriteFile(statePath, []byte("{not json"), 0o644); err != nil {
		t.Fatalf("failed to corrupt state file: %v", err)
	}

	engine2 := sync.NewEngine(newTestClient(server.Handler(), "token-2"), vaultID, vaultKey, dir)
	// Pull from an up-to-date hub reports the current head and must succeed.
	pullRes, err := engine2.Pull(ctx)
	if err != nil {
		t.Fatalf("pull with corrupt state failed: %v", err)
	}
	if pullRes.CommitHash == "" {
		t.Fatalf("expected non-empty commit hash from pull with corrupt state")
	}
}

func TestEngine_SyncStateFile_ForeignVaultIgnored(t *testing.T) {
	server := sync.NewServer("")
	ctx := context.Background()
	vaultKey := testVaultKey(17)

	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "index.md"), []byte("# Index\n"), 0o644)

	author := vault.CommitAuthor{ClientID: "client-1", Agent: "agent"}
	engineA := sync.NewEngine(newTestClient(server.Handler(), "token-a"), "vault-a", vaultKey, dir)
	if _, err := engineA.Push(ctx, author, "Initial push to vault-a"); err != nil {
		t.Fatalf("push to vault-a error: %v", err)
	}

	// A state file written for vault-a must not be adopted by an engine that
	// targets a different vault on the same bundle directory.
	engineB := sync.NewEngine(newTestClient(server.Handler(), "token-b"), "vault-b", vaultKey, dir)
	if _, err := engineB.Push(ctx, author, "Initial push to vault-b"); err != nil {
		t.Fatalf("push to vault-b failed: %v", err)
	}
}
