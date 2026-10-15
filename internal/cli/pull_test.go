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

	"github.com/okf-memory/okf-agent-memory/pkg/registry"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestCmdPull_ScopedSuccess(t *testing.T) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	content := "# Django Rules"
	_ = tw.WriteHeader(&tar.Header{Name: "index.md", Mode: 0o644, Size: int64(len(content))})
	_, _ = tw.Write([]byte(content))
	_ = tw.Close()
	_ = gw.Close()
	data := buf.Bytes()
	hash := fmt.Sprintf("sha256:%x", sha256.Sum256(data))

	mockURL := "https://mock.registry.okf-memory.dev"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/bundles/peter/django-5-rules.json":
			_, _ = fmt.Fprintf(w, `{"id":"peter/django-5-rules","version":"1.0.0","hash":%q,"download_url":"/bundle.tar.gz"}`, hash)
		case "/bundle.tar.gz":
			_, _ = w.Write(data)
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

	cmd, ok := FindCommand("pull")
	if !ok {
		t.Fatalf("pull command not found in registry")
	}

	cmd.Run([]string{"--registry", mockURL, "peter/django-5-rules"})

	targetFile := filepath.Join(workDir, ".okf", "vendor", "peter", "django-5-rules", "index.md")
	if _, err := os.Stat(targetFile); err != nil {
		t.Fatalf("expected pulled bundle file at %s: %v", targetFile, err)
	}

	lockFile := filepath.Join(workDir, "okf.lock")
	if _, err := os.Stat(lockFile); err != nil {
		t.Fatalf("expected okf.lock at %s", lockFile)
	}

	// Test vendor list
	vendorCmd, ok := FindCommand("vendor")
	if !ok {
		t.Fatalf("vendor command not found in registry")
	}
	vendorCmd.Run([]string{"list"})

	// Test vendor remove
	vendorCmd.Run([]string{"remove", "peter/django-5-rules"})
	if _, err := os.Stat(targetFile); !os.IsNotExist(err) {
		t.Errorf("expected vendor file to be deleted: %v", err)
	}
}

func TestCmdVendor_RemoveNonExistent(t *testing.T) {
	workDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(workDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

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

	r, w, _ := os.Pipe()
	origStderr := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = origStderr }()

	vendorCmd.Run([]string{"remove", "non-existent/bundle"})
	_ = w.Close()

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	errOutput := buf.String()

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(errOutput, "vendor bundle \"non-existent/bundle\" is not installed") {
		t.Errorf("expected error message about bundle not installed, got: %s", errOutput)
	}
}

func TestCmdPull_InvalidBundleRollback(t *testing.T) {
	// Archive with NO index.md (invalid OKF bundle)
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	content := "some random file"
	_ = tw.WriteHeader(&tar.Header{Name: "random.txt", Mode: 0o644, Size: int64(len(content))})
	_, _ = tw.Write([]byte(content))
	_ = tw.Close()
	_ = gw.Close()
	data := buf.Bytes()
	hash := fmt.Sprintf("sha256:%x", sha256.Sum256(data))

	mockURL := "https://mock.registry.okf-memory.dev"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/bundles/bad/bundle.json":
			_, _ = fmt.Fprintf(w, `{"id":"bad/bundle","version":"1.0.0","hash":%q,"download_url":"/bundle.tgz"}`, hash)
		case "/bundle.tgz":
			_, _ = w.Write(data)
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

	cmd, ok := FindCommand("pull")
	if !ok {
		t.Fatalf("pull command not found in registry")
	}

	cmd.Run([]string{"--registry", mockURL, "bad/bundle"})
	_ = w.Close()

	var errBuf bytes.Buffer
	_, _ = io.Copy(&errBuf, r)
	errOutput := errBuf.String()

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(errOutput, "is not a valid OKF v0.2 bundle") {
		t.Errorf("expected validation rollback error message, got: %s", errOutput)
	}

	// Verify rollback: vendor dir must not exist
	vendorDir := filepath.Join(workDir, ".okf", "vendor", "bad", "bundle")
	if _, err := os.Stat(vendorDir); !os.IsNotExist(err) {
		t.Errorf("expected vendor directory to be rolled back/deleted, but found: %s", vendorDir)
	}

	// okf.lock must not contain bad/bundle
	lockFile := filepath.Join(workDir, "okf.lock")
	if _, err := os.Stat(lockFile); err == nil {
		content, _ := os.ReadFile(lockFile)
		if strings.Contains(string(content), "bad/bundle") {
			t.Errorf("expected okf.lock to not contain bad/bundle")
		}
	}
}
