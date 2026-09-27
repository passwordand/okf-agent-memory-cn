package sync_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/okf-memory/okf-agent-memory/pkg/sync"
)

type localRoundTripper struct {
	handler http.Handler
}

func (l *localRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rec := httptest.NewRecorder()
	l.handler.ServeHTTP(rec, req)
	return rec.Result(), nil
}

func newTestClient(handler http.Handler, token string) *sync.Client {
	client := sync.NewClient("http://hub.local", token)
	client.HTTPClient.Transport = &localRoundTripper{handler: handler}
	return client
}

func TestClient_GetHead(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/vaults/v123/head" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer secret-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"vault_id":    "v123",
			"head_commit": "commit_hash_1",
			"updated_at":  "2026-09-14T12:00:00Z",
		})
	})

	client := newTestClient(handler, "secret-token")
	head, err := client.GetHead(context.Background(), "v123")
	if err != nil {
		t.Fatalf("unexpected GetHead error: %v", err)
	}

	if head.VaultID != "v123" || head.HeadCommit != "commit_hash_1" {
		t.Fatalf("unexpected head response: %+v", head)
	}
}

func TestClient_Unauthorized(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})

	client := newTestClient(handler, "bad-token")
	_, err := client.GetHead(context.Background(), "v123")
	if !errors.Is(err, sync.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got: %v", err)
	}

	expected := "c1"
	_, err = client.Commit(context.Background(), "v123", "c2", &expected)
	if !errors.Is(err, sync.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got: %v", err)
	}

	_, err = client.CheckMissingBlobs(context.Background(), "v123", []string{"h1"})
	if !errors.Is(err, sync.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got: %v", err)
	}

	err = client.PutBlob(context.Background(), "v123", "h1", []byte("data"))
	if !errors.Is(err, sync.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got: %v", err)
	}

	_, err = client.GetBlob(context.Background(), "v123", "h1")
	if !errors.Is(err, sync.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got: %v", err)
	}
}

func TestClient_Commit_SuccessAndConflict(t *testing.T) {
	currentHead := "c1"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/vaults/v123/commit" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}

		var req struct {
			NewCommitHash        string  `json:"new_commit_hash"`
			ExpectedPreviousHead *string `json:"expected_previous_head"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		expected := ""
		if req.ExpectedPreviousHead != nil {
			expected = *req.ExpectedPreviousHead
		}

		if expected != currentHead {
			w.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error":        "head_mismatch",
				"current_head": currentHead,
			})
			return
		}

		currentHead = req.NewCommitHash
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "accepted",
			"head":   currentHead,
		})
	})

	client := newTestClient(handler, "token")

	// 1. Success commit
	expectedHead := "c1"
	resp, err := client.Commit(context.Background(), "v123", "c2", &expectedHead)
	if err != nil {
		t.Fatalf("unexpected commit error: %v", err)
	}
	if resp.Head != "c2" {
		t.Fatalf("expected head c2, got %s", resp.Head)
	}

	// 2. Conflict commit (submitting with old expectedHead "c1", but server head is now "c2")
	_, err = client.Commit(context.Background(), "v123", "c3", &expectedHead)
	if err == nil {
		t.Fatalf("expected 409 conflict error, got nil")
	}

	var conflictErr *sync.HeadConflictError
	if !errors.As(err, &conflictErr) {
		t.Fatalf("expected *sync.HeadConflictError, got %T: %v", err, err)
	}
	if conflictErr.CurrentHead != "c2" {
		t.Fatalf("expected current_head 'c2', got %q", conflictErr.CurrentHead)
	}
}

func TestClient_CheckMissingBlobs(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/vaults/v123/blobs/check-missing" {
			http.NotFound(w, r)
			return
		}

		var req struct {
			Hashes []string `json:"hashes"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)

		// Pretend "blob1" exists, so "blob2" is missing
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"missing": []string{"blob2"},
		})
	})

	client := newTestClient(handler, "token")
	missing, err := client.CheckMissingBlobs(context.Background(), "v123", []string{"blob1", "blob2"})
	if err != nil {
		t.Fatalf("unexpected CheckMissingBlobs error: %v", err)
	}

	if len(missing) != 1 || missing[0] != "blob2" {
		t.Fatalf("expected missing ['blob2'], got %v", missing)
	}
}

func TestClient_PutAndGetBlob(t *testing.T) {
	store := make(map[string][]byte)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hash := r.URL.Path[len("/api/v1/vaults/v123/blobs/"):]
		if r.Method == http.MethodPut {
			data, _ := io.ReadAll(r.Body)
			store[hash] = data
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"hash": hash,
				"size": len(data),
			})
			return
		}
		if r.Method == http.MethodGet {
			data, ok := store[hash]
			if !ok {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write(data)
			return
		}
		http.NotFound(w, r)
	})

	client := newTestClient(handler, "token")
	payload := []byte("binary-envelope-bytes")
	hash := "fakehash123"

	// PUT blob
	if err := client.PutBlob(context.Background(), "v123", hash, payload); err != nil {
		t.Fatalf("PutBlob error: %v", err)
	}

	// GET blob
	fetched, err := client.GetBlob(context.Background(), "v123", hash)
	if err != nil {
		t.Fatalf("GetBlob error: %v", err)
	}

	if !bytes.Equal(fetched, payload) {
		t.Fatalf("fetched data mismatch")
	}

	// GET non-existent blob -> ErrBlobNotFound
	_, err = client.GetBlob(context.Background(), "v123", "missing")
	if !errors.Is(err, sync.ErrBlobNotFound) {
		t.Fatalf("expected ErrBlobNotFound, got: %v", err)
	}
}
