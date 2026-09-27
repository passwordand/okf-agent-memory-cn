package sync

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/okf-memory/okf-agent-memory/pkg/vault"
)

// Server implements an embedded blind CAS and Atomic Head pointer server.
// It can run completely in-memory (for tests/Wasm/local execution) or with disk persistence.
type Server struct {
	storageDir string
	mu         sync.RWMutex
	heads      map[string]string            // vaultID -> head_commit
	headTimes  map[string]time.Time         // vaultID -> updated_at
	blobs      map[string]map[string][]byte // vaultID -> blobHash -> data
}

// NewServer creates a new Server instance. If storageDir is empty, it operates purely in memory.
func NewServer(storageDir string) *Server {
	return &Server{
		storageDir: storageDir,
		heads:      make(map[string]string),
		headTimes:  make(map[string]time.Time),
		blobs:      make(map[string]map[string][]byte),
	}
}

// Handler returns an http.Handler implementing the OKF Memory Hub REST API specification.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/vaults/", s.handleVaults)
	return mux
}

func (s *Server) handleVaults(w http.ResponseWriter, r *http.Request) {
	// Path format: /api/v1/vaults/{vault_id}/...
	prefix := "/api/v1/vaults/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		http.NotFound(w, r)
		return
	}

	rest := r.URL.Path[len(prefix):]
	parts := strings.Split(rest, "/")
	if len(parts) < 2 {
		http.NotFound(w, r)
		return
	}

	vaultID := parts[0]
	endpoint := parts[1]

	switch endpoint {
	case "head":
		s.handleHead(w, r, vaultID)
	case "commit":
		s.handleCommit(w, r, vaultID)
	case "blobs":
		if len(parts) == 3 && parts[2] == "check-missing" && r.Method == http.MethodPost {
			s.handleCheckMissing(w, r, vaultID)
			return
		}
		if len(parts) == 3 {
			blobHash := parts[2]
			s.handleBlob(w, r, vaultID, blobHash)
			return
		}
		http.NotFound(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleHead(w http.ResponseWriter, r *http.Request, vaultID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.RLock()
	head := s.heads[vaultID]
	updatedAt := s.headTimes[vaultID]
	s.mu.RUnlock()

	timeStr := ""
	if !updatedAt.IsZero() {
		timeStr = updatedAt.Format(time.RFC3339)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(HeadResponse{
		VaultID:    vaultID,
		HeadCommit: head,
		UpdatedAt:  timeStr,
	})
}

func (s *Server) handleCommit(w http.ResponseWriter, r *http.Request, vaultID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		NewCommitHash        string  `json:"new_commit_hash"`
		ExpectedPreviousHead *string `json:"expected_previous_head"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request: invalid json", http.StatusBadRequest)
		return
	}

	expected := ""
	if req.ExpectedPreviousHead != nil {
		expected = *req.ExpectedPreviousHead
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	current := s.heads[vaultID]
	if expected != current {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error":        "head_mismatch",
			"current_head": current,
		})
		return
	}

	s.heads[vaultID] = req.NewCommitHash
	s.headTimes[vaultID] = time.Now().UTC()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(CommitResponse{
		Status: "accepted",
		Head:   req.NewCommitHash,
	})
}

func (s *Server) handleCheckMissing(w http.ResponseWriter, r *http.Request, vaultID string) {
	var req struct {
		Hashes []string `json:"hashes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request: invalid json", http.StatusBadRequest)
		return
	}

	s.mu.RLock()
	vaultBlobs := s.blobs[vaultID]
	var missing []string
	for _, h := range req.Hashes {
		if vaultBlobs == nil || vaultBlobs[h] == nil {
			missing = append(missing, h)
		}
	}
	s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"missing": missing,
	})
}

func (s *Server) handleBlob(w http.ResponseWriter, r *http.Request, vaultID, blobHash string) {
	switch r.Method {
	case http.MethodPut:
		data, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}

		// Verify that blob_hash matches sha256 of data
		actualHash := vault.HashBlob(data)
		if actualHash != blobHash {
			http.Error(w, "Bad Request: blob hash mismatch", http.StatusBadRequest)
			return
		}

		s.mu.Lock()
		if s.blobs[vaultID] == nil {
			s.blobs[vaultID] = make(map[string][]byte)
		}
		s.blobs[vaultID][blobHash] = data
		s.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"hash": blobHash,
			"size": len(data),
		})

	case http.MethodGet:
		s.mu.RLock()
		vaultBlobs := s.blobs[vaultID]
		var data []byte
		if vaultBlobs != nil {
			data = vaultBlobs[blobHash]
		}
		s.mu.RUnlock()

		if data == nil {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(data)

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}
