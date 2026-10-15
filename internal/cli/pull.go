package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/okf-memory/okf-agent-memory/pkg/lock"
	"github.com/okf-memory/okf-agent-memory/pkg/okf"
	"github.com/okf-memory/okf-agent-memory/pkg/registry"
)

var newRegistryClient = func(baseURL string) *registry.Client {
	return registry.NewClient(baseURL)
}

func printPullUsage() {
	fmt.Println(`Usage:
  okf pull [<bundle-id|git-url>] [flags]

Pulls and installs an external knowledge bundle into .okf/vendor/ and updates okf.lock.
If no bundle is specified, restores all bundles declared in okf.lock.

Flags:
  --registry <url>   Override canonical registry endpoint (default: https://registry.okf-memory.dev)
  --force            Overwrite existing vendor installation if present

Examples:
  okf pull
  okf pull nextjs-15
  okf pull @peter/django-5-rules
  okf pull peter/django-5-rules@1.0.0
  okf pull github.com/acme/agent-rules@v1.0.0`)
}

func cmdPull(args []string) {
	fs := flag.NewFlagSet("pull", flag.ExitOnError)
	registryURL := fs.String("registry", registry.DefaultRegistryURL, "Registry endpoint")
	force := fs.Bool("force", false, "Overwrite existing vendor installation")
	fs.Usage = printPullUsage

	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	if fs.NArg() < 1 {
		if _, err := os.Stat("okf.lock"); os.IsNotExist(err) {
			fmt.Fprintln(os.Stderr, "Error: okf.lock not found. Specify a bundle to pull (e.g. 'okf pull <bundle-id>') or run in a workspace with an existing okf.lock.")
			exitFunc(1)
			return
		}
		runRestore(*registryURL, *force)
		return
	}

	target := fs.Arg(0)
	client := newRegistryClient(*registryURL)

	fmt.Printf("Resolving %s from %s...\n", target, client.BaseURL)
	manifest, err := client.Resolve(target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		exitFunc(1)
		return
	}

	if manifest.Version == "main" && (strings.HasPrefix(target, "github.com/") || strings.HasPrefix(target, "https://")) {
		fmt.Fprintf(os.Stderr, "Warning: Pulling floating branch 'main'. Pin to @vX.Y.Z for reproducible builds.\n")
	}

	vendorDir := filepath.Join(".okf", "vendor", filepath.FromSlash(manifest.ID))
	if _, err := os.Stat(vendorDir); err == nil && !*force {
		fmt.Printf("Bundle %q is already installed in %s. Use --force to reinstall.\n", manifest.ID, vendorDir)
		return
	}

	fmt.Printf("Downloading %s (version: %s)...\n", manifest.ID, manifest.Version)
	if err := client.DownloadAndExtract(manifest, vendorDir); err != nil {
		fmt.Fprintf(os.Stderr, "Download error: %v\n", err)
		exitFunc(1)
		return
	}

	// Post-install validation: ensure installed bundle conforms to OKF v0.2
	rollback := func() {
		_ = os.RemoveAll(vendorDir)
		if strings.Contains(manifest.ID, "/") {
			parentDir := filepath.Dir(vendorDir)
			if entries, err := os.ReadDir(parentDir); err == nil && len(entries) == 0 {
				_ = os.Remove(parentDir)
			}
		}
	}

	if _, err := os.Stat(filepath.Join(vendorDir, "index.md")); os.IsNotExist(err) {
		rollback()
		fmt.Fprintf(os.Stderr, "Error: bundle %q is not a valid OKF v0.2 bundle: missing index.md (installation rolled back)\n", manifest.ID)
		exitFunc(1)
		return
	}

	vb, err := okf.LoadBundle(vendorDir)
	if err != nil {
		rollback()
		fmt.Fprintf(os.Stderr, "Error: bundle %q failed to load: %v (installation rolled back)\n", manifest.ID, err)
		exitFunc(1)
		return
	}

	diag := okf.Validate(vb, okf.ValidateOptions{Strict: true})
	if !diag.IsConformant {
		rollback()
		fmt.Fprintf(os.Stderr, "Error: bundle %q failed strict validation (%d errors, %d broken links); installation rolled back\n",
			manifest.ID, len(diag.Errors), len(diag.BrokenLinks))
		exitFunc(1)
		return
	}

	lockPath := "okf.lock"
	lf, err := lock.ReadLockfile(lockPath)
	if err != nil {
		lf = &lock.Lockfile{Version: 1}
	}
	lf.Upsert(lock.BundleEntry{
		ID:          manifest.ID,
		Source:      target,
		Version:     manifest.Version,
		Hash:        manifest.Hash,
		InstalledAt: time.Now().UTC(),
	})
	if err := lock.WriteLockfile(lockPath, lf); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to update okf.lock: %v\n", err)
	}

	fmt.Printf("Successfully installed and verified %s in %s\n", manifest.ID, vendorDir)
}
