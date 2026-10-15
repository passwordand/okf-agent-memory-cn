package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/okf-memory/okf-agent-memory/pkg/lock"
)

var exitFunc = os.Exit

func printVendorUsage() {
	fmt.Println(`Usage:
  okf vendor <command>

Commands:
  list     List installed vendor bundles
  remove   Uninstall a vendor bundle (okf vendor remove <bundle-id>)

Examples:
  okf vendor list
  okf vendor remove nextjs-15
  okf vendor remove @peter/django-5-rules`)
}

func cmdVendor(args []string) {
	if len(args) == 0 {
		printVendorUsage()
		return
	}

	switch args[0] {
	case "list":
		lf, err := lock.ReadLockfile("okf.lock")
		if err != nil || len(lf.Bundles) == 0 {
			fmt.Println("No vendor bundles installed.")
			return
		}
		fmt.Printf("Installed Vendor Bundles (%d):\n", len(lf.Bundles))
		for _, b := range lf.Bundles {
			fmt.Printf("  - %-24s %-10s (%s)\n", b.ID, b.Version, b.Source)
		}

	case "remove":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: okf vendor remove <bundle-id>")
			exitFunc(1)
			return
		}
		bundleID := strings.TrimPrefix(args[1], "@")
		vendorDir := filepath.Join(".okf", "vendor", filepath.FromSlash(bundleID))

		lf, err := lock.ReadLockfile("okf.lock")
		lockHasBundle := false
		if err == nil && lf != nil {
			for _, b := range lf.Bundles {
				if b.ID == bundleID {
					lockHasBundle = true
					break
				}
			}
		}

		// A directory is only considered an installed bundle if it contains index.md.
		// Subdirectories without index.md (e.g. parent namespace or organization dirs)
		// are not valid bundles and must not be removed by partial matches.
		isBundleDir := false
		if info, err := os.Stat(vendorDir); err == nil && info.IsDir() {
			if _, err := os.Stat(filepath.Join(vendorDir, "index.md")); err == nil {
				isBundleDir = true
			}
		}

		if !isBundleDir && !lockHasBundle {
			var suggestions []string
			if lf != nil {
				for _, b := range lf.Bundles {
					if strings.HasPrefix(b.ID, bundleID+"/") || strings.Contains(b.ID, bundleID) {
						suggestions = append(suggestions, b.ID)
					}
				}
			}
			if len(suggestions) > 0 {
				fmt.Fprintf(os.Stderr, "Error: vendor bundle %q is not installed. Did you mean %q?\n", bundleID, suggestions[0])
			} else {
				fmt.Fprintf(os.Stderr, "Error: vendor bundle %q is not installed\n", bundleID)
			}
			exitFunc(1)
			return
		}

		if isBundleDir {
			_ = os.RemoveAll(vendorDir)
			if strings.Contains(bundleID, "/") {
				parentDir := filepath.Dir(vendorDir)
				if entries, err := os.ReadDir(parentDir); err == nil && len(entries) == 0 {
					_ = os.Remove(parentDir)
				}
			}
		}

		if lockHasBundle && lf != nil {
			if lf.Remove(bundleID) {
				if len(lf.Bundles) == 0 {
					_ = os.Remove("okf.lock")
				} else {
					if err := lock.WriteLockfile("okf.lock", lf); err != nil {
						fmt.Fprintf(os.Stderr, "Warning: failed to update okf.lock: %v\n", err)
					}
				}
			}
		}
		fmt.Printf("Removed vendor bundle %s\n", bundleID)

	default:
		printVendorUsage()
	}
}
