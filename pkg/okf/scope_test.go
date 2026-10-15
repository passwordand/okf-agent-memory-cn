package okf_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/okf-memory/okf-agent-memory/pkg/okf"
)

func TestScope_ResolveURI(t *testing.T) {
	// 1. Scoped bundle with okf://@: okf://@peter/django-5-rules/decisions/auth
	uri := "okf://@peter/django-5-rules/decisions/auth"
	scope, bundleID, conceptID, err := okf.ParseURI(uri)
	if err != nil {
		t.Fatalf("ParseURI failed: %v", err)
	}
	if scope != okf.ScopeVendor {
		t.Errorf("expected ScopeVendor, got %s", scope)
	}
	if bundleID != "peter/django-5-rules" {
		t.Errorf("expected peter/django-5-rules, got %s", bundleID)
	}
	if conceptID != "decisions/auth" {
		t.Errorf("expected decisions/auth, got %s", conceptID)
	}

	// 2. Direct @-reference: @peter/django-5-rules/decisions/auth
	scopeBare, bundleIDBare, conceptIDBare, err := okf.ParseURI("@peter/django-5-rules/decisions/auth")
	if err != nil {
		t.Fatalf("ParseURI bare @ failed: %v", err)
	}
	if scopeBare != okf.ScopeVendor || bundleIDBare != "peter/django-5-rules" || conceptIDBare != "decisions/auth" {
		t.Errorf("unexpected bare @ parse result: %s, %s, %s", scopeBare, bundleIDBare, conceptIDBare)
	}

	// 3. Top-level vendor bundle with okf://@: okf://@nextjs-15/decisions/routing
	uriTop := "okf://@nextjs-15/decisions/routing"
	scopeTop, bundleIDTop, conceptIDTop, err := okf.ParseURI(uriTop)
	if err != nil {
		t.Fatalf("ParseURI top-level failed: %v", err)
	}
	if scopeTop != okf.ScopeVendor {
		t.Errorf("expected ScopeVendor, got %s", scopeTop)
	}
	if bundleIDTop != "nextjs-15" {
		t.Errorf("expected nextjs-15, got %s", bundleIDTop)
	}
	if conceptIDTop != "decisions/routing" {
		t.Errorf("expected decisions/routing, got %s", conceptIDTop)
	}

	// 4. Direct @-reference top-level: @nextjs-15/decisions/routing
	scopeTopBare, bundleIDTopBare, conceptIDTopBare, err := okf.ParseURI("@nextjs-15/decisions/routing")
	if err != nil {
		t.Fatalf("ParseURI bare @ top failed: %v", err)
	}
	if scopeTopBare != okf.ScopeVendor || bundleIDTopBare != "nextjs-15" || conceptIDTopBare != "decisions/routing" {
		t.Errorf("unexpected bare @ top parse result: %s, %s, %s", scopeTopBare, bundleIDTopBare, conceptIDTopBare)
	}

	// 5. Unscoped target without @ (e.g. nextjs-15/decisions/routing) must reject ParseURI
	// so caller treats it as a local bundle concept!
	if _, _, _, err := okf.ParseURI("nextjs-15/decisions/routing"); err == nil {
		t.Errorf("expected error for unscoped target without @, but succeeded")
	}

	// 6. User scope: okf://user/preferences
	uriUser := "okf://user/preferences"
	scopeUser, _, conceptIDUser, err := okf.ParseURI(uriUser)
	if err != nil {
		t.Fatalf("ParseURI user failed: %v", err)
	}
	if scopeUser != okf.ScopeUser || conceptIDUser != "preferences" {
		t.Errorf("unexpected user scope parse result")
	}

	// 7. Markdown link normalization: @org/repo/path.md -> okf://@org/repo/path
	shorthand := "@peter/django-5-rules/decisions/auth.md"
	normURI := okf.NormalizeVendorLink(shorthand)
	if normURI != "okf://@peter/django-5-rules/decisions/auth" {
		t.Errorf("expected normalized URI, got %s", normURI)
	}

	shorthandTop := "@nextjs-15/decisions/routing.md"
	normURITop := okf.NormalizeVendorLink(shorthandTop)
	if normURITop != "okf://@nextjs-15/decisions/routing" {
		t.Errorf("expected normalized top URI, got %s", normURITop)
	}

	// 8. User and System shorthand: user:preferences/style.md -> okf://user/preferences/style
	normUser := okf.NormalizeVendorLink("user:preferences/style.md")
	if normUser != "okf://user/preferences/style" {
		t.Errorf("expected normalized user URI, got %s", normUser)
	}
	sUser, _, cUser, err := okf.ParseURI("user:preferences/style")
	if err != nil || sUser != okf.ScopeUser || cUser != "preferences/style" {
		t.Errorf("unexpected parse result for user: shorthand: %s, %s, %v", sUser, cUser, err)
	}

	normSys := okf.NormalizeVendorLink("system:compliance/soc2.md")
	if normSys != "okf://system/compliance/soc2" {
		t.Errorf("expected normalized system URI, got %s", normSys)
	}
	sSys, _, cSys, err := okf.ParseURI("system:compliance/soc2")
	if err != nil || sSys != okf.ScopeSystem || cSys != "compliance/soc2" {
		t.Errorf("unexpected parse result for system: shorthand: %s, %s, %v", sSys, cSys, err)
	}
}

func TestIsExternalLink(t *testing.T) {
	cases := []struct {
		href     string
		expected bool
	}{
		{"decisions/routing.md", false},
		{"../architecture/database.md", false},
		{"index.md", false},
		{"@nextjs-15/decisions/routing.md", true},
		{"@nextjs-15/decisions/routing", true},
		{"@peter/django-rules/auth.md", true},
		{"user:preferences/style.md", true},
		{"system:corp/policies.md", true},
		{"okf://user/preferences", true},
		{"okf://system/policy", true},
		{"okf://@nextjs-15/routing", true},
		{"https://example.com/docs", true},
		{"http://localhost:8080", true},
		{"mailto:admin@example.com", true},
		{"", false},
	}

	for _, tc := range cases {
		actual := okf.IsExternalLink(tc.href)
		if actual != tc.expected {
			t.Errorf("IsExternalLink(%q) = %v; want %v", tc.href, actual, tc.expected)
		}
	}
}

func TestValidate_ExternalLinksNeverFail(t *testing.T) {
	tmpDir := t.TempDir()

	indexContent := `---
okf_version: "0.2"
title: "Test Bundle"
---
# Test Bundle
- [Architecture](architecture/index.md) - Architecture docs
`
	if err := os.WriteFile(filepath.Join(tmpDir, "index.md"), []byte(indexContent), 0o600); err != nil {
		t.Fatal(err)
	}

	archDir := filepath.Join(tmpDir, "architecture")
	if err := os.MkdirAll(archDir, 0o750); err != nil {
		t.Fatal(err)
	}

	archIndex := `# Architecture
- [Core](core.md) - Core architecture
`
	if err := os.WriteFile(filepath.Join(archDir, "index.md"), []byte(archIndex), 0o600); err != nil {
		t.Fatal(err)
	}

	coreContent := `---
type: Architecture
title: Core Architecture
description: Core architecture
status: stable
---
# Core Architecture

We build upon vendor standards and external rules:
- Vendor package: [@nextjs-15/decisions/routing](@nextjs-15/decisions/routing.md)
- Scoped vendor package: [@peter/django-rules/auth](@peter/django-rules/auth.md)
- User settings: [User Style](user:preferences/style.md)
- System compliance: [SOC2 Policy](system:compliance/soc2.md)
- Canonical URI: [Remote Policy](okf://system/compliance/soc2.md)
- Web reference: [Next.js Docs](https://nextjs.org/docs)
`
	if err := os.WriteFile(filepath.Join(archDir, "core.md"), []byte(coreContent), 0o600); err != nil {
		t.Fatal(err)
	}

	b, err := okf.LoadBundle(tmpDir)
	if err != nil {
		t.Fatalf("LoadBundle failed: %v", err)
	}

	res := okf.Validate(b, okf.ValidateOptions{
		Strict: true,
		Drift:  true,
	})

	if len(res.Errors) > 0 {
		t.Errorf("expected 0 errors, got: %v", res.Errors)
	}
	if len(b.BrokenLinks) > 0 {
		t.Errorf("expected 0 broken links for external references, got: %v", b.BrokenLinks)
	}
	if !res.GatePassed {
		t.Errorf("expected GatePassed=true under Strict mode with external links, but gate failed")
	}
	if !res.IsConformant {
		t.Errorf("expected IsConformant=true, but failed")
	}
}
