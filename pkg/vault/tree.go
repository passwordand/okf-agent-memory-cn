package vault

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
)

// CurrentTreeVersion represents version 1 of the vault tree manifest schema.
const CurrentTreeVersion = 1

// ErrUnsupportedTreeVersion indicates a tree manifest version is not supported.
var ErrUnsupportedTreeVersion = errors.New("vault: unsupported tree manifest version")

// TreeEntry describes a single file version stored in the vault tree.
type TreeEntry struct {
	// BlobHash is the SHA-256 hash of the encrypted binary envelope in the CAS store.
	BlobHash string `json:"blob_hash"`
	// Size is the file size in bytes.
	Size int64 `json:"size"`
	// PlaintextHash is the SHA-256 hash of the unencrypted file content.
	// Used for client-side change detection without needing to re-encrypt.
	PlaintextHash string `json:"plaintext_hash"`
	// ModifiedAt is the ISO 8601 (RFC3339) timestamp of last modification.
	ModifiedAt string `json:"modified_at,omitempty"`
}

// Tree represents the manifest of all files tracked within a vault commit snapshot.
type Tree struct {
	Version int                  `json:"version"`
	Entries map[string]TreeEntry `json:"entries"`
}

// NewTree creates an empty, initialized Tree at the current schema version.
func NewTree() *Tree {
	return &Tree{
		Version: CurrentTreeVersion,
		Entries: make(map[string]TreeEntry),
	}
}

// Serialize encodes the Tree as canonical JSON bytes.
// Go's encoding/json deterministically sorts map keys alphabetically.
func (t *Tree) Serialize() ([]byte, error) {
	data, err := json.Marshal(t)
	if err != nil {
		return nil, fmt.Errorf("vault: failed to marshal tree manifest: %w", err)
	}
	return data, nil
}

// Hash returns the deterministic SHA-256 hex digest of the serialized Tree manifest.
// This is the tree_hash referenced in Commit objects.
func (t *Tree) Hash() (string, error) {
	data, err := t.Serialize()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// ParseTree deserializes and validates a Tree manifest from JSON bytes.
func ParseTree(data []byte) (*Tree, error) {
	var t Tree
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("vault: failed to parse tree manifest: %w", err)
	}

	if t.Version != CurrentTreeVersion {
		return nil, ErrUnsupportedTreeVersion
	}

	if t.Entries == nil {
		t.Entries = make(map[string]TreeEntry)
	}

	return &t, nil
}

// HasChanged checks whether a file at path with the given plaintext SHA-256 hash
// differs from what is currently recorded in this Tree manifest.
func (t *Tree) HasChanged(path, plaintextHash string) bool {
	entry, ok := t.Entries[path]
	if !ok {
		return true
	}
	return entry.PlaintextHash != plaintextHash
}

// TreeDiff categorizes differences between a base Tree and a target Tree.
type TreeDiff struct {
	Added     []string
	Modified  []string
	Deleted   []string
	Unchanged []string
}

// DiffTrees computes the file-level differences between base and target trees.
func DiffTrees(base, target *Tree) TreeDiff {
	var diff TreeDiff

	baseEntries := make(map[string]TreeEntry)
	if base != nil && base.Entries != nil {
		baseEntries = base.Entries
	}

	targetEntries := make(map[string]TreeEntry)
	if target != nil && target.Entries != nil {
		targetEntries = target.Entries
	}

	// Check entries in target against base
	for path, targetEntry := range targetEntries {
		baseEntry, exists := baseEntries[path]
		if !exists {
			diff.Added = append(diff.Added, path)
		} else if baseEntry.PlaintextHash != targetEntry.PlaintextHash || baseEntry.BlobHash != targetEntry.BlobHash {
			diff.Modified = append(diff.Modified, path)
		} else {
			diff.Unchanged = append(diff.Unchanged, path)
		}
	}

	// Check entries in base that do not exist in target
	for path := range baseEntries {
		if _, exists := targetEntries[path]; !exists {
			diff.Deleted = append(diff.Deleted, path)
		}
	}

	sort.Strings(diff.Added)
	sort.Strings(diff.Modified)
	sort.Strings(diff.Deleted)
	sort.Strings(diff.Unchanged)

	return diff
}
