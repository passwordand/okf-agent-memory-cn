package okf_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/okf-memory/okf-agent-memory/pkg/okf"
)

func conceptWithFrontmatter(fields string) string {
	return "---\ntype: Fact\n" + fields + "\n---\n\n# Body\n"
}

func TestParseConcept_CustomListFields(t *testing.T) {
	tests := []struct {
		name   string
		fields string
		key    string
		want   []any
	}{
		{"flow list", "repos: [contextopia, easygov]", "repos", []any{"contextopia", "easygov"}},
		{"flow list single quotes", "repos: ['a, b', c]", "repos", []any{"a, b", "c"}},
		{"flow list double quotes", `repos: ["a, b", c]`, "repos", []any{"a, b", "c"}},
		{"flow list scalar types", "values: [1, true, text]", "values", []any{float64(1), true, "text"}},
		{"empty flow list", "items: []", "items", []any{}},
		{"block list", "commands:\n  - git commit", "commands", []any{"git commit"}},
		{"block list without indent", "commands:\n- git commit\n- git push", "commands", []any{"git commit", "git push"}},
		{"block list quoted item with colon", "commands:\n  - \"a: b\"\n  - 'c'", "commands", []any{"a: b", "c"}},
		{"block list scalar types", "values:\n  - 2\n  - false\n  - text", "values", []any{float64(2), false, "text"}},
		{"block list url is a scalar", "links:\n  - https://example.com/a", "links", []any{"https://example.com/a"}},
		{"block list negative number", "values:\n  - -5", "values", []any{float64(-5)}},
		{"nested flow list", "matrix: [[a, b], c]", "matrix", []any{[]any{"a", "b"}, "c"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := okf.ParseConcept("note.md", conceptWithFrontmatter(tt.fields))
			if err != nil {
				t.Fatalf("ParseConcept failed: %v", err)
			}
			if got := c.Extra[tt.key]; !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("Extra[%q] = %#v, want %#v", tt.key, got, tt.want)
			}

			reparsed, err := okf.ParseConcept("note.md", okf.SerializeConcept(c))
			if err != nil {
				t.Fatalf("ParseConcept after SerializeConcept failed: %v", err)
			}
			if got := reparsed.Extra[tt.key]; !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("after round trip Extra[%q] = %#v, want %#v", tt.key, got, tt.want)
			}
		})
	}
}

func TestParseConcept_NestedBlockValuesStayVerbatim(t *testing.T) {
	tests := []struct {
		name   string
		fields string
		want   string
	}{
		{"list of mappings", "people:\n  - name: ada\n    role: dev", "people:\n  - name: ada\n    role: dev\n"},
		{"nested mapping", "owner:\n  name: ada\n  team: core", "owner:\n  name: ada\n  team: core\n"},
		{"nested list", "groups:\n  - - a\n    - b", "groups:\n  - - a\n    - b\n"},
		{"uneven indentation", "groups:\n  - a\n    - b", "groups:\n  - a\n    - b\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := okf.ParseConcept("note.md", conceptWithFrontmatter(tt.fields))
			if err != nil {
				t.Fatalf("ParseConcept failed: %v", err)
			}
			if got := okf.SerializeConcept(c); !strings.Contains(got, tt.want) {
				t.Fatalf("serialized concept lost its block structure:\n%s", got)
			}
		})
	}
}

func TestUpdateKeepsCustomListFieldsAsLists(t *testing.T) {
	dir := t.TempDir()
	index := "---\nokf_version: \"0.2\"\n---\n\n# Index\n"
	if err := os.WriteFile(filepath.Join(dir, "index.md"), []byte(index), 0o644); err != nil {
		t.Fatal(err)
	}
	fields := "title: Note\ndescription: original\ntags: [git]\ncommands:\n  - git commit\nrepos: [contextopia, easygov]"
	if err := os.WriteFile(filepath.Join(dir, "note.md"), []byte(conceptWithFrontmatter(fields)), 0o644); err != nil {
		t.Fatal(err)
	}

	b, err := okf.LoadBundle(dir)
	if err != nil {
		t.Fatalf("LoadBundle failed: %v", err)
	}
	c := b.Concepts["note"]
	c.Description = "updated"
	if err := okf.SaveConcept(dir, c, okf.SaveOptions{Actor: "agent/test"}); err != nil {
		t.Fatalf("SaveConcept failed: %v", err)
	}

	reloaded, err := okf.LoadBundle(dir)
	if err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	got := reloaded.Concepts["note"]
	if want := []any{"contextopia", "easygov"}; !reflect.DeepEqual(got.Extra["repos"], want) {
		t.Errorf("repos = %#v, want %#v", got.Extra["repos"], want)
	}
	if want := []any{"git commit"}; !reflect.DeepEqual(got.Extra["commands"], want) {
		t.Errorf("commands = %#v, want %#v", got.Extra["commands"], want)
	}
	if want := []string{"git"}; !reflect.DeepEqual(got.Tags, want) {
		t.Errorf("tags = %#v, want %#v", got.Tags, want)
	}
}
