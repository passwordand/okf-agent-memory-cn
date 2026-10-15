package sync

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/okf-memory/okf-agent-memory/pkg/vault"
)

const (
	// syncStateDirName is the hidden directory inside a bundle that holds the
	// persisted sync state. Engine.ScanBundle skips hidden directories, so the
	// state file is never treated as bundle content and never pushed to the hub.
	syncStateDirName = ".okf"
	// syncStateFileName is the name of the persisted sync state file.
	syncStateFileName = "sync-state.json"
	// currentSyncStateVersion is the schema version of the sync state file.
	// Engines refuse to load files with any other version.
	currentSyncStateVersion = 1
)

// syncState is the on-disk representation of the engine's cached sync
// position: the last synced head commit and its tree manifest. It lets a new
// engine instance (i.e. a new `okf hub push|pull|sync` invocation) resume from
// the previous position instead of starting from an empty state.
type syncState struct {
	Version int             `json:"version"`
	VaultID string          `json:"vault_id"`
	Head    string          `json:"head"`
	Tree    json.RawMessage `json:"tree"`
	SavedAt string          `json:"saved_at,omitempty"`
}

// syncStatePath returns the location of the sync state file inside the bundle.
func (e *Engine) syncStatePath() string {
	return filepath.Join(e.BundlePath, syncStateDirName, syncStateFileName)
}

// loadState restores the cached head and tree from the bundle's sync state
// file so the engine continues from the last synced position. Expected heads
// are then sent on commit (avoiding spurious 409s) and the restored tree is
// used as the three-way merge base (avoiding false collisions).
//
// Any problem (missing file, corrupt JSON, unknown schema version, state that
// belongs to a different vault, undecodable tree) is silently ignored: the
// engine falls back to the empty-state behavior and treats the next push as
// first-of-vault, exactly as before persistence existed.
func (e *Engine) loadState() {
	// #nosec G304 -- path is derived from the user-provided bundle root
	data, err := os.ReadFile(e.syncStatePath())
	if err != nil {
		return
	}

	var st syncState
	if err := json.Unmarshal(data, &st); err != nil {
		return
	}
	if st.Version != currentSyncStateVersion || st.Head == "" || len(st.Tree) == 0 {
		return
	}
	if st.VaultID != e.VaultID {
		return
	}

	tree, err := vault.ParseTree(st.Tree)
	if err != nil {
		return
	}

	e.cachedHead = st.Head
	e.cachedTree = tree
}

// saveState atomically persists the cached head and tree into the bundle so
// the next engine instance can resume from this position. It must be called
// after every successful push/pull/sync that advances the cached state.
func (e *Engine) saveState() error {
	if e.cachedHead == "" || e.cachedTree == nil {
		return nil
	}

	treeJSON, err := e.cachedTree.Serialize()
	if err != nil {
		return fmt.Errorf("sync: failed to serialize tree for sync state: %w", err)
	}

	data, err := json.Marshal(syncState{
		Version: currentSyncStateVersion,
		VaultID: e.VaultID,
		Head:    e.cachedHead,
		Tree:    treeJSON,
		SavedAt: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		return fmt.Errorf("sync: failed to encode sync state: %w", err)
	}

	dir := filepath.Join(e.BundlePath, syncStateDirName)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("sync: failed to create sync state directory: %w", err)
	}

	// Atomic write: temp file inside the hidden state directory, then rename.
	// ScanBundle skips both the hidden directory and "*.tmp" files, so the
	// temp file can never leak into a push while being written.
	tmp, err := os.CreateTemp(dir, syncStateFileName+".*.tmp")
	if err != nil {
		return fmt.Errorf("sync: failed to create temp sync state file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }() // no-op once renamed into place

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync: failed to write sync state: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync: failed to flush sync state: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("sync: failed to close sync state: %w", err)
	}
	if err := os.Rename(tmpName, e.syncStatePath()); err != nil {
		return fmt.Errorf("sync: failed to persist sync state: %w", err)
	}

	return nil
}
