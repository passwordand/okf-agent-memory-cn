package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/okf-memory/okf-agent-memory/pkg/sync"
	"github.com/okf-memory/okf-agent-memory/pkg/vault"
)

type localRoundTripper struct {
	handler http.Handler
}

func (l *localRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rec := httptest.NewRecorder()
	l.handler.ServeHTTP(rec, req)
	return rec.Result(), nil
}

func newTestHubServer() *sync.Server {
	return sync.NewServer("")
}

func newTestClientFromHandler(handler http.Handler, token string) *sync.Client {
	c := sync.NewClient("http://hub.local", token)
	c.HTTPClient.Transport = &localRoundTripper{handler: handler}
	return c
}

func TestCmdHub_InitVault(t *testing.T) {
	var buf bytes.Buffer
	dir := t.TempDir()

	err := runHubInitVault(&buf, dir, "http://127.0.0.1:8080", "")
	if err != nil {
		t.Fatalf("runHubInitVault error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "EMERGENCY KIT") {
		t.Fatalf("output missing EMERGENCY KIT header:\n%s", out)
	}
	if !strings.Contains(out, "Vault ID:") {
		t.Fatalf("output missing Vault ID:\n%s", out)
	}
	if !strings.Contains(out, "Secret Key:") {
		t.Fatalf("output missing Secret Key:\n%s", out)
	}

	// Verify local config file was created
	cfgPath := filepath.Join(dir, ".okf-vault.json")
	if _, err := os.Stat(cfgPath); err != nil {
		t.Fatalf("expected config file %s to exist", cfgPath)
	}
}

func TestCmdHub_PushPullSyncWithServer(t *testing.T) {
	// Initialize a vault in dirA
	dirA := t.TempDir()
	_ = os.WriteFile(filepath.Join(dirA, "index.md"), []byte("# Knowledge Index\nokf_version: 0.2\n"), 0o644)

	var initBuf bytes.Buffer
	if err := runHubInitVault(&initBuf, dirA, "http://127.0.0.1:8080", ""); err != nil {
		t.Fatalf("init error: %v", err)
	}

	// Read generated config
	cfgA, err := loadVaultConfig(dirA)
	if err != nil {
		t.Fatalf("load config error: %v", err)
	}

	secretKey, err := vault.GenerateSecretKey()
	if err != nil {
		t.Fatalf("GenerateSecretKey error: %v", err)
	}
	password := "master-pass-123"

	// Create test in-memory server handler
	server := newTestHubServer()
	clientA := newTestClientFromHandler(server.Handler(), "token-a")

	// 1. Push from A
	var pushBuf bytes.Buffer
	err = executeHubPush(&pushBuf, dirA, clientA, cfgA.VaultID, password, secretKey, "Initial push")
	if err != nil {
		t.Fatalf("executeHubPush error: %v", err)
	}
	if !strings.Contains(pushBuf.String(), "Push completed") {
		t.Fatalf("expected push completion message, got:\n%s", pushBuf.String())
	}

	// 2. Pull into dirB
	dirB := t.TempDir()
	clientB := newTestClientFromHandler(server.Handler(), "token-b")
	var pullBuf bytes.Buffer
	err = executeHubPull(&pullBuf, dirB, clientB, cfgA.VaultID, password, secretKey)
	if err != nil {
		t.Fatalf("executeHubPull error: %v", err)
	}

	pulledIndex, err := os.ReadFile(filepath.Join(dirB, "index.md"))
	if err != nil {
		t.Fatalf("failed to read pulled index.md: %v", err)
	}
	if string(pulledIndex) != "# Knowledge Index\nokf_version: 0.2\n" {
		t.Fatalf("unexpected content in pulled index.md: %s", string(pulledIndex))
	}
}

func TestResolveToken(t *testing.T) {
	cfg := &VaultConfigFile{
		VaultID:   "v_test",
		HubURL:    "http://127.0.0.1:8080",
		AuthToken: "cfg-token",
	}

	// 1. Flag priority
	t.Setenv("OKF_HUB_TOKEN", "env-token")

	if tok := resolveToken("flag-token", cfg); tok != "flag-token" {
		t.Fatalf("expected flag-token, got %s", tok)
	}

	// 2. Env priority over config
	if tok := resolveToken("", cfg); tok != "env-token" {
		t.Fatalf("expected env-token, got %s", tok)
	}

	// 3. Config fallback
	t.Setenv("OKF_HUB_TOKEN", "")
	if tok := resolveToken("", cfg); tok != "cfg-token" {
		t.Fatalf("expected cfg-token, got %s", tok)
	}

	// 4. Empty
	if tok := resolveToken("", nil); tok != "" {
		t.Fatalf("expected empty string, got %s", tok)
	}
}

func TestCmdHub_BearerAuthProtection(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "index.md"), []byte("# Index\n"), 0o644)
	var initBuf bytes.Buffer
	if err := runHubInitVault(&initBuf, dir, "http://127.0.0.1:8080", "secret-token"); err != nil {
		t.Fatalf("init error: %v", err)
	}
	cfg, _ := loadVaultConfig(dir)
	secretKey, _ := vault.GenerateSecretKey()

	// Handler that requires Bearer secret-token
	authHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer secret-token" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		// Forward to memory server
		newTestHubServer().Handler().ServeHTTP(w, r)
	})

	// Client with valid token
	clientValid := newTestClientFromHandler(authHandler, "secret-token")
	var pushBuf bytes.Buffer
	err := executeHubPush(&pushBuf, dir, clientValid, cfg.VaultID, "pass", secretKey, "msg")
	if err != nil {
		t.Fatalf("expected push to succeed with valid token, got: %v", err)
	}

	// Client with wrong token
	clientInvalid := newTestClientFromHandler(authHandler, "wrong-token")
	err = executeHubPush(&pushBuf, dir, clientInvalid, cfg.VaultID, "pass", secretKey, "msg")
	if err == nil || !strings.Contains(err.Error(), "unauthorized") {
		t.Fatalf("expected unauthorized error with wrong token, got: %v", err)
	}
}

func TestResolveHubURL(t *testing.T) {
	cfg := &VaultConfigFile{
		VaultID: "v_test",
		HubURL:  "https://hub.example.com",
	}

	// Flag priority
	if u := resolveHubURL("http://override.local", cfg); u != "http://override.local" {
		t.Fatalf("expected override URL, got %s", u)
	}

	// Config fallback
	if u := resolveHubURL("", cfg); u != "https://hub.example.com" {
		t.Fatalf("expected config URL, got %s", u)
	}

	// Default fallback
	if u := resolveHubURL("", nil); u != "http://127.0.0.1:8080" {
		t.Fatalf("expected default URL, got %s", u)
	}
}

func TestCmdHub_Serve_E2E_Socket(t *testing.T) {
	// 1. Start real embedded HTTP server on dynamic local port (as in "okf hub serve")
	srv := sync.NewServer("")
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	password := "master-pass-e2e"
	secretKey, err := vault.GenerateSecretKey()
	if err != nil {
		t.Fatalf("GenerateSecretKey error: %v", err)
	}

	// 2. Initialize Bundle A on real HTTP server URL
	dirA := t.TempDir()
	_ = os.WriteFile(filepath.Join(dirA, "index.md"), []byte("# Root Index\nokf_version: 0.2\n"), 0o644)
	subDirA := filepath.Join(dirA, "concepts")
	_ = os.MkdirAll(subDirA, 0o755)
	_ = os.WriteFile(filepath.Join(subDirA, "architecture.md"), []byte("# Architecture\nSocket E2E test\n"), 0o644)

	var initBuf bytes.Buffer
	if err := runHubInitVault(&initBuf, dirA, ts.URL, "secret-token-e2e"); err != nil {
		t.Fatalf("init-vault error: %v", err)
	}

	cfgA, err := loadVaultConfig(dirA)
	if err != nil {
		t.Fatalf("load config error: %v", err)
	}
	if cfgA.HubURL != ts.URL {
		t.Fatalf("expected config hub_url %s, got %s", ts.URL, cfgA.HubURL)
	}

	// 3. Client A: Real HTTP Push
	clientA := sync.NewClient(ts.URL, "secret-token-e2e")
	var pushBuf bytes.Buffer
	err = executeHubPush(&pushBuf, dirA, clientA, cfgA.VaultID, password, secretKey, "Initial socket push")
	if err != nil {
		t.Fatalf("real socket push failed: %v", err)
	}
	if !strings.Contains(pushBuf.String(), "Push completed") {
		t.Fatalf("expected push completed, got:\n%s", pushBuf.String())
	}

	// 4. Verify remote head over real HTTP
	headResp, err := clientA.GetHead(context.Background(), cfgA.VaultID)
	if err != nil {
		t.Fatalf("GetHead over socket failed: %v", err)
	}
	if headResp.HeadCommit == "" {
		t.Fatalf("expected non-empty head commit on remote server")
	}

	// 5. Client B: Clone / Pull into separate Bundle B over real HTTP
	dirB := t.TempDir()
	clientB := sync.NewClient(ts.URL, "secret-token-e2e")
	var pullBuf bytes.Buffer
	err = executeHubPull(&pullBuf, dirB, clientB, cfgA.VaultID, password, secretKey)
	if err != nil {
		t.Fatalf("real socket pull failed: %v", err)
	}

	// 6. Verify bit-exact file matching over socket
	pulledIndex, err := os.ReadFile(filepath.Join(dirB, "index.md"))
	if err != nil {
		t.Fatalf("failed to read pulled index.md: %v", err)
	}
	if string(pulledIndex) != "# Root Index\nokf_version: 0.2\n" {
		t.Fatalf("mismatch in pulled index.md: %s", string(pulledIndex))
	}

	pulledArch, err := os.ReadFile(filepath.Join(dirB, "concepts", "architecture.md"))
	if err != nil {
		t.Fatalf("failed to read pulled architecture.md: %v", err)
	}
	if string(pulledArch) != "# Architecture\nSocket E2E test\n" {
		t.Fatalf("mismatch in pulled architecture.md: %s", string(pulledArch))
	}

	// 7. Verify Concurrent Reconcile Sync over real HTTP socket (disjoint addition)
	noteBPath := filepath.Join(dirB, "concepts", "note_b.md")
	_ = os.WriteFile(noteBPath, []byte("# Note B\nCreated on Device B\n"), 0o644)
	var syncBuf bytes.Buffer
	err = executeHubSync(&syncBuf, dirB, clientB, cfgA.VaultID, password, secretKey, "Sync note B from device B")
	if err != nil {
		t.Fatalf("real socket sync failed: %v", err)
	}
	if !strings.Contains(syncBuf.String(), "Sync completed") {
		t.Fatalf("expected sync completed, got:\n%s", syncBuf.String())
	}
	if strings.Contains(syncBuf.String(), "Collision detected") {
		t.Fatalf("expected no collision for disjoint add, got:\n%s", syncBuf.String())
	}

	// Device A pulls the merged state
	var pullBuf2 bytes.Buffer
	err = executeHubPull(&pullBuf2, dirA, clientA, cfgA.VaultID, password, secretKey)
	if err != nil {
		t.Fatalf("device A pull failed: %v", err)
	}

	pulledNoteB, err := os.ReadFile(filepath.Join(dirA, "concepts", "note_b.md"))
	if err != nil {
		t.Fatalf("device A failed to receive note_b.md: %v", err)
	}
	if string(pulledNoteB) != "# Note B\nCreated on Device B\n" {
		t.Fatalf("content mismatch for note_b.md on device A: %s", string(pulledNoteB))
	}

	// 8. Verify Collision Failsafe over real HTTP socket
	// Device B modifies note_b.md. In stateless CLI execution, conflicting edits fork safely into .conflict-local.md
	_ = os.WriteFile(noteBPath, []byte("# Note B Conflicting Local Edit\n"), 0o644)
	var syncBufConflict bytes.Buffer
	err = executeHubSync(&syncBufConflict, dirB, clientB, cfgA.VaultID, password, secretKey, "Device B sync conflicting edit")
	if err != nil {
		t.Fatalf("device B sync with conflict failed: %v", err)
	}
	if !strings.Contains(syncBufConflict.String(), "Collision detected at concepts/note_b.md") {
		t.Fatalf("expected collision detected warning, got:\n%s", syncBufConflict.String())
	}

	// Verify conflict-local file created with zero data loss
	conflictForkPath := filepath.Join(dirB, "concepts", "note_b.conflict-local.md")
	conflictContent, err := os.ReadFile(conflictForkPath)
	if err != nil {
		t.Fatalf("conflict fork file %s was not created: %v", conflictForkPath, err)
	}
	if string(conflictContent) != "# Note B Conflicting Local Edit\n" {
		t.Fatalf("unexpected content in conflict fork: %s", string(conflictContent))
	}
}
