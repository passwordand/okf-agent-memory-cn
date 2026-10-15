package cli

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/okf-memory/okf-agent-memory/pkg/lock"
	"github.com/okf-memory/okf-agent-memory/pkg/registry"
)

func createTestBundleTarGz(indexTitle string) ([]byte, string) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	content := fmt.Sprintf("# %s\n\n---\ntype: Fact\ntitle: %s\n---\n", indexTitle, indexTitle)
	_ = tw.WriteHeader(&tar.Header{Name: "index.md", Mode: 0o644, Size: int64(len(content))})
	_, _ = tw.Write([]byte(content))
	_ = tw.Close()
	_ = gw.Close()
	data := buf.Bytes()
	hash := fmt.Sprintf("sha256:%x", sha256.Sum256(data))
	return data, hash
}

func TestCmdVendor_RemovePrefixSafeguard(t *testing.T) {
	workDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(workDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Create scoped bundle dir .okf/vendor/sknr/okf-bundle-example/index.md
	bundleDir := filepath.Join(workDir, ".okf", "vendor", "sknr", "okf-bundle-example")
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		t.Fatal(err)
	}
	indexFile := filepath.Join(bundleDir, "index.md")
	if err := os.WriteFile(indexFile, []byte("# Demo Bundle\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create okf.lock
	lf := &lock.Lockfile{Version: 1}
	lf.Upsert(lock.BundleEntry{
		ID:          "sknr/okf-bundle-example",
		Source:      "github.com/sknr/okf-bundle-example@v1.0.0",
		Version:     "v1.0.0",
		Hash:        "sha256:123456",
		InstalledAt: time.Now().UTC(),
	})
	if err := lock.WriteLockfile("okf.lock", lf); err != nil {
		t.Fatal(err)
	}

	vendorCmd, ok := FindCommand("vendor")
	if !ok {
		t.Fatalf("vendor command not found")
	}

	var exitCode int
	origExit := exitFunc
	defer func() { exitFunc = origExit }()
	exitFunc = func(code int) {
		exitCode = code
	}

	// Capture stderr
	r, w, _ := os.Pipe()
	origStderr := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = origStderr }()

	// Try removing with just prefix "sknr"
	vendorCmd.Run([]string{"remove", "sknr"})
	_ = w.Close()

	var errBuf bytes.Buffer
	_, _ = io.Copy(&errBuf, r)
	errOutput := errBuf.String()

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(errOutput, "vendor bundle \"sknr\" is not installed. Did you mean \"sknr/okf-bundle-example\"?") {
		t.Errorf("expected helpful suggestion in error output, got: %s", errOutput)
	}

	// Verify bundle file was NOT deleted
	if _, err := os.Stat(indexFile); err != nil {
		t.Errorf("expected bundle index.md to still exist after prefix remove attempt: %v", err)
	}

	// Verify okf.lock still contains entry
	readLf, err := lock.ReadLockfile("okf.lock")
	if err != nil || len(readLf.Bundles) != 1 {
		t.Errorf("expected okf.lock to remain unchanged with 1 bundle, got: %v", readLf)
	}

	// Now remove with full exact ID
	exitCode = 0
	vendorCmd.Run([]string{"remove", "sknr/okf-bundle-example"})

	if _, err := os.Stat(bundleDir); !os.IsNotExist(err) {
		t.Errorf("expected bundle directory to be deleted")
	}
	// Parent dir "sknr" should also be cleaned up
	parentDir := filepath.Join(workDir, ".okf", "vendor", "sknr")
	if _, err := os.Stat(parentDir); !os.IsNotExist(err) {
		t.Errorf("expected parent directory 'sknr' to be cleaned up")
	}
	// okf.lock should be cleaned up because 0 bundles remain
	if _, err := os.Stat("okf.lock"); !os.IsNotExist(err) {
		t.Errorf("expected okf.lock to be deleted when empty")
	}
}

func TestCmdRestore_SuccessAndSkip(t *testing.T) {
	remoteData, remoteHash := createTestBundleTarGz("Remote Rules")

	mockURL := "https://mock.registry.okf-memory.dev"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/bundles/acme/remote-rules.json", "/bundles/acme/remote-rules-1.0.0.json":
			_, _ = fmt.Fprintf(w, `{"id":"acme/remote-rules","version":"1.0.0","hash":%q,"download_url":"/bundle.tar.gz"}`, remoteHash)
		case "/bundle.tar.gz":
			_, _ = w.Write(remoteData)
		default:
			http.NotFound(w, r)
		}
	})

	origFactory := newRegistryClient
	defer func() { newRegistryClient = origFactory }()
	newRegistryClient = func(baseURL string) *registry.Client {
		c := registry.NewClient(baseURL)
		c.HTTPClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			return rec.Result(), nil
		})
		return c
	}

	workDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(workDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Pre-install "local/existing"
	existingDir := filepath.Join(workDir, ".okf", "vendor", "local", "existing")
	if err := os.MkdirAll(existingDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(existingDir, "index.md"), []byte("# Existing\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create okf.lock with both
	lf := &lock.Lockfile{Version: 1}
	lf.Upsert(lock.BundleEntry{
		ID:          "local/existing",
		Source:      "local/existing",
		Version:     "1.0.0",
		Hash:        "sha256:dummy",
		InstalledAt: time.Now().UTC(),
	})
	lf.Upsert(lock.BundleEntry{
		ID:          "acme/remote-rules",
		Source:      "acme/remote-rules",
		Version:     "1.0.0",
		Hash:        remoteHash,
		InstalledAt: time.Now().UTC(),
	})
	if err := lock.WriteLockfile("okf.lock", lf); err != nil {
		t.Fatal(err)
	}

	restoreCmd, ok := FindCommand("restore")
	if !ok {
		t.Fatalf("restore command not found")
	}

	// Capture stdout
	r, w, _ := os.Pipe()
	origStdout := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = origStdout }()

	restoreCmd.Run([]string{"--registry", mockURL})
	_ = w.Close()

	var outBuf bytes.Buffer
	_, _ = io.Copy(&outBuf, r)
	output := outBuf.String()

	// Windows 输出本机路径分隔符，先归一化再断言恢复行为。
	if !strings.Contains(filepath.ToSlash(output), "Bundle \"local/existing\" is already installed in .okf/vendor/local/existing (skipping)") {
		t.Errorf("expected skip message for existing bundle, got: %s", output)
	}
	if !strings.Contains(output, "Successfully restored acme/remote-rules") {
		t.Errorf("expected success message for restored bundle, got: %s", output)
	}

	// Verify acme/remote-rules now exists on disk
	restoredIndex := filepath.Join(workDir, ".okf", "vendor", "acme", "remote-rules", "index.md")
	if _, err := os.Stat(restoredIndex); err != nil {
		t.Errorf("expected restored bundle file at %s: %v", restoredIndex, err)
	}
}

func TestCmdRestore_NoLock(t *testing.T) {
	workDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(workDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	restoreCmd, ok := FindCommand("restore")
	if !ok {
		t.Fatalf("restore command not found")
	}

	var exitCode int
	origExit := exitFunc
	defer func() { exitFunc = origExit }()
	exitFunc = func(code int) {
		exitCode = code
	}

	r, w, _ := os.Pipe()
	origStderr := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = origStderr }()

	restoreCmd.Run([]string{})
	_ = w.Close()

	var errBuf bytes.Buffer
	_, _ = io.Copy(&errBuf, r)
	errOutput := errBuf.String()

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(errOutput, "okf.lock not found") {
		t.Errorf("expected error about missing okf.lock, got: %s", errOutput)
	}
}

func TestCmdPull_ArglessRestoresFromLock(t *testing.T) {
	remoteData, remoteHash := createTestBundleTarGz("Pull Restored Rules")

	mockURL := "https://mock.registry.okf-memory.dev"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "acme/pull-rules") {
			_, _ = fmt.Fprintf(w, `{"id":"acme/pull-rules","version":"1.0.0","hash":%q,"download_url":"/bundle.tar.gz"}`, remoteHash)
		} else if r.URL.Path == "/bundle.tar.gz" {
			_, _ = w.Write(remoteData)
		} else {
			http.NotFound(w, r)
		}
	})

	origFactory := newRegistryClient
	defer func() { newRegistryClient = origFactory }()
	newRegistryClient = func(baseURL string) *registry.Client {
		c := registry.NewClient(baseURL)
		c.HTTPClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			return rec.Result(), nil
		})
		return c
	}

	workDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(workDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Create okf.lock
	lf := &lock.Lockfile{Version: 1}
	lf.Upsert(lock.BundleEntry{
		ID:          "acme/pull-rules",
		Source:      "acme/pull-rules",
		Version:     "1.0.0",
		Hash:        remoteHash,
		InstalledAt: time.Now().UTC(),
	})
	if err := lock.WriteLockfile("okf.lock", lf); err != nil {
		t.Fatal(err)
	}

	pullCmd, ok := FindCommand("pull")
	if !ok {
		t.Fatalf("pull command not found")
	}

	// Capture stdout
	r, w, _ := os.Pipe()
	origStdout := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = origStdout }()

	// Run okf pull with NO positional arguments
	pullCmd.Run([]string{"--registry", mockURL})
	_ = w.Close()

	var outBuf bytes.Buffer
	_, _ = io.Copy(&outBuf, r)
	output := outBuf.String()

	if !strings.Contains(output, "Successfully restored acme/pull-rules") {
		t.Errorf("expected success message from argless pull, got: %s", output)
	}

	// Verify bundle exists
	restoredIndex := filepath.Join(workDir, ".okf", "vendor", "acme", "pull-rules", "index.md")
	if _, err := os.Stat(restoredIndex); err != nil {
		t.Errorf("expected pulled bundle file at %s: %v", restoredIndex, err)
	}
}

func TestCmdPull_ArglessNoLock(t *testing.T) {
	workDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(workDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	pullCmd, ok := FindCommand("pull")
	if !ok {
		t.Fatalf("pull command not found")
	}

	var exitCode int
	origExit := exitFunc
	defer func() { exitFunc = origExit }()
	exitFunc = func(code int) {
		exitCode = code
	}

	r, w, _ := os.Pipe()
	origStderr := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = origStderr }()

	pullCmd.Run([]string{})
	_ = w.Close()

	var errBuf bytes.Buffer
	_, _ = io.Copy(&errBuf, r)
	errOutput := errBuf.String()

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(errOutput, "okf.lock not found") {
		t.Errorf("expected error about missing okf.lock, got: %s", errOutput)
	}
}
