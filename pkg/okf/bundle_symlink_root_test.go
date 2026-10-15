package okf_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/okf-memory/okf-agent-memory/pkg/okf"
)

const symlinkTestConcept = "---\ntype: Fact\ntitle: Note\ndescription: Reachable through a symlink\n---\n\n# Note\n"

func writeSymlinkTestBundle(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	index := "---\nokf_version: \"0.2\"\n---\n\n# Index\n"
	if err := os.WriteFile(filepath.Join(dir, "index.md"), []byte(index), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "note.md"), []byte(symlinkTestConcept), 0o644); err != nil {
		t.Fatal(err)
	}
}

func symlinkOrSkip(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks not supported on this platform: %v", err)
	}
}

func requireNote(t *testing.T, b *okf.Bundle) {
	t.Helper()
	if _, ok := b.Concepts["note"]; !ok {
		t.Fatalf("expected concept %q to be loaded, got %d concepts", "note", len(b.Concepts))
	}
}

func TestLoadBundle_SymlinkedRootWithinParent(t *testing.T) {
	base := t.TempDir()
	writeSymlinkTestBundle(t, filepath.Join(base, "real"))
	link := filepath.Join(base, "link")
	symlinkOrSkip(t, filepath.Join(base, "real"), link)

	for name, root := range map[string]string{
		"plain":          link,
		"trailing slash": link + string(filepath.Separator),
	} {
		t.Run(name, func(t *testing.T) {
			b, err := okf.LoadBundle(root)
			if err != nil {
				t.Fatalf("LoadBundle(%q) failed: %v", root, err)
			}
			requireNote(t, b)
		})
	}
}

func TestLoadBundle_SymlinkedRootResolvesKnowledgeSubdir(t *testing.T) {
	base := t.TempDir()
	writeSymlinkTestBundle(t, filepath.Join(base, "project", "knowledge"))
	link := filepath.Join(base, "link")
	symlinkOrSkip(t, filepath.Join(base, "project"), link)

	b, err := okf.LoadBundle(link)
	if err != nil {
		t.Fatalf("LoadBundle failed: %v", err)
	}
	requireNote(t, b)
}

func TestLoadBundle_SymlinkedRootOutsideParentIsRejected(t *testing.T) {
	outside := t.TempDir()
	writeSymlinkTestBundle(t, outside)
	project := t.TempDir()
	link := filepath.Join(project, "knowledge")
	symlinkOrSkip(t, outside, link)

	for name, root := range map[string]string{
		"plain":          link,
		"trailing slash": link + string(filepath.Separator),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := okf.LoadBundle(root)
			if err == nil {
				t.Fatalf("LoadBundle(%q) must reject a root symlink escaping its parent", root)
			}
			if !strings.Contains(err.Error(), "escapes bundle directory") {
				t.Errorf("expected containment error, got: %v", err)
			}
		})
	}
}

func TestLoadBundle_KnowledgeSubdirSymlinkOutsideIsRejected(t *testing.T) {
	outside := t.TempDir()
	writeSymlinkTestBundle(t, outside)
	project := t.TempDir()
	symlinkOrSkip(t, outside, filepath.Join(project, "knowledge"))

	_, err := okf.LoadBundle(project)
	if err == nil {
		t.Fatal("LoadBundle must reject a knowledge/ symlink escaping the project")
	}
	if !strings.Contains(err.Error(), "escapes bundle directory") {
		t.Errorf("expected containment error, got: %v", err)
	}
}

func TestSearchLayered_TrustedScopeRootMayBeSymlinkAnywhere(t *testing.T) {
	scopes := []struct {
		name string
		opts func(root string) okf.LayeredSearchOptions
	}{
		{"user", func(root string) okf.LayeredSearchOptions {
			return okf.LayeredSearchOptions{Scope: "user", UserDir: root}
		}},
		{"system", func(root string) okf.LayeredSearchOptions {
			return okf.LayeredSearchOptions{Scope: "system", SystemDir: root}
		}},
	}
	for _, sc := range scopes {
		t.Run(sc.name, func(t *testing.T) {
			real := filepath.Join(t.TempDir(), "elsewhere")
			writeSymlinkTestBundle(t, real)
			link := filepath.Join(t.TempDir(), "okf")
			symlinkOrSkip(t, real, link)

			opts := sc.opts(link)
			opts.SearchOpts = okf.SearchOptions{Query: "Reachable", Limit: 5}
			results, err := okf.SearchLayered(opts)
			if err != nil {
				t.Fatalf("SearchLayered failed: %v", err)
			}
			if len(results) != 1 {
				t.Fatalf("expected 1 result through symlinked %s root, got %d", sc.name, len(results))
			}
		})
	}
}

func TestResolveScopedConcept_TrustedScopeRootMayBeSymlinkAnywhere(t *testing.T) {
	real := filepath.Join(t.TempDir(), "elsewhere")
	writeSymlinkTestBundle(t, real)
	link := filepath.Join(t.TempDir(), "okf")
	symlinkOrSkip(t, real, link)

	res, err := okf.ResolveScopedConcept("user:note", "", "", link, "")
	if err != nil {
		t.Fatalf("ResolveScopedConcept failed: %v", err)
	}
	if res.Concept.Title != "Note" {
		t.Errorf("unexpected concept: %+v", res.Concept)
	}
}

func TestSearchLayered_ProjectRootSymlinkOutsideIsNotSearched(t *testing.T) {
	outside := t.TempDir()
	writeSymlinkTestBundle(t, outside)
	project := t.TempDir()
	link := filepath.Join(project, "knowledge")
	symlinkOrSkip(t, outside, link)

	_, err := okf.SearchLayered(okf.LayeredSearchOptions{
		Scope:      "project",
		BundleDir:  link,
		SearchOpts: okf.SearchOptions{Query: "Reachable", Limit: 5},
	})
	if err == nil {
		t.Fatal("project scope must not load a bundle root symlinked outside the project")
	}
}
