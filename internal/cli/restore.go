package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/okf-memory/okf-agent-memory/pkg/lock"
	"github.com/okf-memory/okf-agent-memory/pkg/okf"
	"github.com/okf-memory/okf-agent-memory/pkg/registry"
)

func printRestoreUsage() {
	fmt.Println(`Usage:
  okf restore [flags]

Restores all vendor bundles specified in okf.lock.

Flags:
  --registry <url>   Override canonical registry endpoint (default: https://registry.okf-memory.dev)
  --force            Reinstall all bundles even if already present

Examples:
  okf restore
  okf restore --force`)
}

func cmdRestore(args []string) {
	fs := flag.NewFlagSet("restore", flag.ExitOnError)
	registryURL := fs.String("registry", registry.DefaultRegistryURL, "Registry endpoint")
	force := fs.Bool("force", false, "Reinstall existing vendor bundles")
	fs.Usage = printRestoreUsage

	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	if _, err := os.Stat("okf.lock"); os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "Error: okf.lock not found. Run 'okf pull <bundle-id>' to install a bundle.")
		exitFunc(1)
		return
	}

	runRestore(*registryURL, *force)
}

func runRestore(registryURL string, force bool) {
	lf, err := lock.ReadLockfile("okf.lock")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading okf.lock: %v\n", err)
		exitFunc(1)
		return
	}

	if lf == nil || len(lf.Bundles) == 0 {
		fmt.Println("No vendor bundles specified in okf.lock.")
		return
	}

	client := newRegistryClient(registryURL)
	restoredCount := 0
	skippedCount := 0

	for _, entry := range lf.Bundles {
		vendorDir := filepath.Join(".okf", "vendor", filepath.FromSlash(entry.ID))
		indexFile := filepath.Join(vendorDir, "index.md")

		if _, err := os.Stat(indexFile); err == nil && !force {
			fmt.Printf("Bundle %q is already installed in %s (skipping)\n", entry.ID, vendorDir)
			skippedCount++
			continue
		}

		source := entry.Source
		if source == "" {
			source = entry.ID
		}
		if entry.Version != "" && !strings.Contains(source, "@") && entry.Version != "main" {
			source = fmt.Sprintf("%s@%s", source, entry.Version)
		}

		fmt.Printf("Resolving %s from %s...\n", entry.ID, client.BaseURL)
		manifest, err := client.Resolve(source)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error resolving %s: %v\n", entry.ID, err)
			exitFunc(1)
			return
		}

		if entry.Hash != "" {
			manifest.Hash = entry.Hash
		}

		fmt.Printf("Downloading %s (version: %s)...\n", manifest.ID, manifest.Version)
		if err := client.DownloadAndExtract(manifest, vendorDir); err != nil {
			fmt.Fprintf(os.Stderr, "Download error for %s: %v\n", manifest.ID, err)
			exitFunc(1)
			return
		}

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

		fmt.Printf("Successfully restored %s in %s\n", manifest.ID, vendorDir)
		restoredCount++
	}

	fmt.Printf("Restore complete: %d restored, %d skipped.\n", restoredCount, skippedCount)
}
