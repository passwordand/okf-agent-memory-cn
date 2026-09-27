package sync_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/okf-memory/okf-agent-memory/pkg/sync"
	"github.com/okf-memory/okf-agent-memory/pkg/vault"
)

func TestServer_LifecycleAndCAS(t *testing.T) {
	server := sync.NewServer("") // In-memory server
	client := newTestClient(server.Handler(), "test-token")
	ctx := context.Background()
	vaultID := "test-vault-1"

	// 1. Initial head is empty
	head, err := client.GetHead(ctx, vaultID)
	if err != nil {
		t.Fatalf("GetHead error: %v", err)
	}
	if head.HeadCommit != "" {
		t.Fatalf("expected empty initial head, got %q", head.HeadCommit)
	}

	// 2. Put blobs
	data1 := []byte("blob content 1")
	hash1 := vault.HashBlob(data1)
	if err := client.PutBlob(ctx, vaultID, hash1, data1); err != nil {
		t.Fatalf("PutBlob error: %v", err)
	}

	// 3. Put blob with bad hash should fail with HTTP 400
	err = client.PutBlob(ctx, vaultID, "wronghash123", data1)
	if err == nil {
		t.Fatalf("expected error for mismatching blob hash, got nil")
	}

	// 4. Check missing blobs
	missing, err := client.CheckMissingBlobs(ctx, vaultID, []string{hash1, "nonexistent-hash"})
	if err != nil {
		t.Fatalf("CheckMissingBlobs error: %v", err)
	}
	if len(missing) != 1 || missing[0] != "nonexistent-hash" {
		t.Fatalf("expected missing ['nonexistent-hash'], got %v", missing)
	}

	// 5. Get blob
	fetched, err := client.GetBlob(ctx, vaultID, hash1)
	if err != nil {
		t.Fatalf("GetBlob error: %v", err)
	}
	if !bytes.Equal(fetched, data1) {
		t.Fatalf("fetched blob mismatch")
	}

	// 6. Commit initial advance (expectedHead = nil / "")
	commit1Hash := "commit-1-hash"
	resp, err := client.Commit(ctx, vaultID, commit1Hash, nil)
	if err != nil {
		t.Fatalf("Commit initial error: %v", err)
	}
	if resp.Head != commit1Hash {
		t.Fatalf("expected head %q, got %q", commit1Hash, resp.Head)
	}

	// 7. Verify GetHead returns commit1
	head, err = client.GetHead(ctx, vaultID)
	if err != nil {
		t.Fatalf("GetHead error: %v", err)
	}
	if head.HeadCommit != commit1Hash {
		t.Fatalf("expected head %q, got %q", commit1Hash, head.HeadCommit)
	}

	// 8. Successful second commit
	commit2Hash := "commit-2-hash"
	resp, err = client.Commit(ctx, vaultID, commit2Hash, &commit1Hash)
	if err != nil {
		t.Fatalf("Commit second error: %v", err)
	}
	if resp.Head != commit2Hash {
		t.Fatalf("expected head %q, got %q", commit2Hash, resp.Head)
	}

	// 9. Conflict: trying to advance from commit1 again
	var conflictErr *sync.HeadConflictError
	_, err = client.Commit(ctx, vaultID, "commit-3-alt", &commit1Hash)
	if err == nil {
		t.Fatalf("expected 409 conflict, got nil")
	}
	if !errors.As(err, &conflictErr) {
		t.Fatalf("expected *sync.HeadConflictError, got: %v", err)
	}
	if conflictErr.CurrentHead != commit2Hash {
		t.Fatalf("expected current_head %q, got %q", commit2Hash, conflictErr.CurrentHead)
	}
}
