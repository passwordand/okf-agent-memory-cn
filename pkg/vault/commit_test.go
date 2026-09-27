package vault_test

import (
	"crypto/rand"
	"errors"
	"testing"
	"time"

	"github.com/okf-memory/okf-agent-memory/pkg/vault"
)

func TestCommitInitial(t *testing.T) {
	author := vault.CommitAuthor{
		ClientID: "skn-macbook-pro",
		Agent:    "antigravity/claude-code",
	}

	commit := vault.NewCommit(nil, "7d91a23e5b6c891f", author, "Initial commit")
	if commit.Version != 1 {
		t.Fatalf("expected version 1, got %d", commit.Version)
	}
	if commit.ParentCommit != nil {
		t.Fatalf("expected ParentCommit to be nil for initial commit")
	}

	data, err := commit.Serialize()
	if err != nil {
		t.Fatalf("unexpected Serialize error: %v", err)
	}

	parsed, err := vault.ParseCommit(data)
	if err != nil {
		t.Fatalf("unexpected ParseCommit error: %v", err)
	}

	if parsed.ParentCommit != nil {
		t.Fatalf("expected parsed ParentCommit to be nil, got %v", *parsed.ParentCommit)
	}
	if parsed.TreeHash != "7d91a23e5b6c891f" {
		t.Fatalf("expected tree hash '7d91a23e5b6c891f', got %q", parsed.TreeHash)
	}
	if parsed.Author.ClientID != "skn-macbook-pro" || parsed.Author.Agent != "antigravity/claude-code" {
		t.Fatalf("author mismatch: %+v", parsed.Author)
	}
	if parsed.Message != "Initial commit" {
		t.Fatalf("message mismatch: %q", parsed.Message)
	}
}

func TestCommitWithParent(t *testing.T) {
	parentHash := "e5b6c891f0123456"
	author := vault.CommitAuthor{
		ClientID: "skn-macbook-pro",
		Agent:    "antigravity/claude-code",
	}

	commit := vault.NewCommit(&parentHash, "tree-hash-2", author, "Second commit")
	if commit.ParentCommit == nil || *commit.ParentCommit != parentHash {
		t.Fatalf("expected parent commit %s, got %v", parentHash, commit.ParentCommit)
	}

	data, err := commit.Serialize()
	if err != nil {
		t.Fatalf("failed to serialize: %v", err)
	}

	parsed, err := vault.ParseCommit(data)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if parsed.ParentCommit == nil || *parsed.ParentCommit != parentHash {
		t.Fatalf("expected parsed parent commit %s, got %v", parentHash, parsed.ParentCommit)
	}
}

func TestCommitDeterministicHash(t *testing.T) {
	parent := "parent123"
	author := vault.CommitAuthor{ClientID: "client1", Agent: "agent1"}
	ts := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC).Format(time.RFC3339)

	c1 := vault.NewCommit(&parent, "tree123", author, "msg")
	c1.Timestamp = ts

	c2 := vault.NewCommit(&parent, "tree123", author, "msg")
	c2.Timestamp = ts

	h1, err := c1.Hash()
	if err != nil {
		t.Fatalf("c1 Hash error: %v", err)
	}

	h2, err := c2.Hash()
	if err != nil {
		t.Fatalf("c2 Hash error: %v", err)
	}

	if h1 != h2 {
		t.Fatalf("expected identical commit hash:\nh1: %s\nh2: %s", h1, h2)
	}
}

func TestCommitValidation(t *testing.T) {
	// Empty tree hash
	author := vault.CommitAuthor{ClientID: "c", Agent: "a"}
	c := vault.NewCommit(nil, "", author, "msg")
	_, err := c.Serialize()
	if !errors.Is(err, vault.ErrEmptyTreeHash) {
		t.Fatalf("expected ErrEmptyTreeHash on empty tree hash, got: %v", err)
	}

	// Invalid JSON
	_, err = vault.ParseCommit([]byte("{bad-json"))
	if err == nil {
		t.Fatalf("expected error on invalid JSON")
	}
}

func TestCommitEncryptAndCASRoundtrip(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	author := vault.CommitAuthor{ClientID: "mac", Agent: "test"}
	commit := vault.NewCommit(nil, "tree-hash-xyz", author, "Confidential commit")

	commitData, err := commit.Serialize()
	if err != nil {
		t.Fatalf("failed to serialize commit: %v", err)
	}

	// Encrypt commit into binary envelope
	envelopeBytes, err := vault.EncryptPayload(commitData, key)
	if err != nil {
		t.Fatalf("failed to encrypt commit: %v", err)
	}

	casBlobHash := vault.HashBlob(envelopeBytes)
	if len(casBlobHash) != 64 {
		t.Fatalf("expected 64-char sha256 hex hash, got %d chars", len(casBlobHash))
	}

	// Decrypt envelope
	decryptedBytes, err := vault.DecryptPayload(envelopeBytes, key)
	if err != nil {
		t.Fatalf("failed to decrypt commit: %v", err)
	}

	parsedCommit, err := vault.ParseCommit(decryptedBytes)
	if err != nil {
		t.Fatalf("failed to parse decrypted commit: %v", err)
	}

	if parsedCommit.TreeHash != commit.TreeHash || parsedCommit.Message != commit.Message {
		t.Fatalf("decrypted commit does not match original: %+v vs %+v", parsedCommit, commit)
	}
}
