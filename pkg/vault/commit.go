package vault

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// CurrentCommitVersion is the version identifier for Commit schema.
const CurrentCommitVersion = 1

var (
	// ErrEmptyTreeHash indicates that the commit does not reference a tree manifest.
	ErrEmptyTreeHash = errors.New("vault: tree_hash cannot be empty in commit")
	// ErrUnsupportedCommitVersion indicates an unsupported commit schema version.
	ErrUnsupportedCommitVersion = errors.New("vault: unsupported commit schema version")
)

// CommitAuthor records client device identification and agent producer metadata.
type CommitAuthor struct {
	ClientID string `json:"client_id"`
	Agent    string `json:"agent"`
}

// Commit represents an immutable snapshot node in the vault version history.
// It is serialized to JSON, encrypted into an Envelope, and stored in the blind CAS store.
type Commit struct {
	Version      int          `json:"version"`
	ParentCommit *string      `json:"parent_commit"` // nil/null for the repository's initial commit
	TreeHash     string       `json:"tree_hash"`
	Timestamp    string       `json:"timestamp"` // ISO 8601 / RFC3339 UTC
	Author       CommitAuthor `json:"author"`
	Message      string       `json:"message"`
}

// NewCommit creates an initialized Commit with the current timestamp in UTC RFC3339 format.
func NewCommit(parentCommit *string, treeHash string, author CommitAuthor, message string) *Commit {
	return &Commit{
		Version:      CurrentCommitVersion,
		ParentCommit: parentCommit,
		TreeHash:     treeHash,
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		Author:       author,
		Message:      message,
	}
}

// Serialize serializes the commit into canonical JSON bytes.
func (c *Commit) Serialize() ([]byte, error) {
	if c.TreeHash == "" {
		return nil, ErrEmptyTreeHash
	}

	data, err := json.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("vault: failed to marshal commit: %w", err)
	}
	return data, nil
}

// Hash returns the deterministic SHA-256 hex digest of the serialized Commit JSON.
func (c *Commit) Hash() (string, error) {
	data, err := c.Serialize()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// ParseCommit deserializes and validates a Commit object from JSON bytes.
func ParseCommit(data []byte) (*Commit, error) {
	var c Commit
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("vault: failed to parse commit: %w", err)
	}

	if c.Version != CurrentCommitVersion {
		return nil, ErrUnsupportedCommitVersion
	}

	if c.TreeHash == "" {
		return nil, ErrEmptyTreeHash
	}

	return &c, nil
}
