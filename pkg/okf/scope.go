package okf

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Scope string

const (
	ScopeProject Scope = "project"
	ScopeVendor  Scope = "vendor"
	ScopeUser    Scope = "user"
	ScopeSystem  Scope = "system"
)

const (
	PriorityProject = 100
	PriorityVendor  = 70
	PriorityUser    = 50
	PrioritySystem  = 10
)

type LayeredSearchResult struct {
	ConceptID string  `json:"concept_id"`
	Scope     Scope   `json:"scope"`
	Priority  int     `json:"priority"`
	Score     float64 `json:"score"`
	Title     string  `json:"title"`
}

// NormalizeVendorLink normalizes a vendor link or markdown reference into an okf://@ URI.
// For example:
//
//	@peter/django-5-rules/decisions/auth.md -> okf://@peter/django-5-rules/decisions/auth
//	@nextjs-15/decisions/routing.md        -> okf://@nextjs-15/decisions/routing
func NormalizeVendorLink(link string) string {
	if strings.HasPrefix(link, "@") {
		rel := strings.TrimPrefix(link, "@")
		rel = strings.TrimSuffix(rel, ".md")
		return "okf://@" + rel
	}
	if strings.HasPrefix(link, "user:") {
		rel := strings.TrimPrefix(link, "user:")
		rel = strings.TrimPrefix(rel, "/")
		rel = strings.TrimSuffix(rel, ".md")
		return "okf://user/" + rel
	}
	if strings.HasPrefix(link, "system:") {
		rel := strings.TrimPrefix(link, "system:")
		rel = strings.TrimPrefix(rel, "/")
		rel = strings.TrimSuffix(rel, ".md")
		return "okf://system/" + rel
	}
	return link
}

// ParseVendorRef splits a bundle-relative target (e.g. "nextjs-15/decisions/routing" or
// "peter/django-5-rules/decisions/auth") into bundleID and conceptID.
// It checks vendorRoot (default .okf/vendor) for existing index.md files to disambiguate
// between 1-part and 2-part bundle names, with a deterministic fallback heuristic.
func ParseVendorRef(target, vendorRoot string) (string, string, error) {
	clean := strings.TrimPrefix(target, "@")
	clean = strings.Trim(clean, "/")
	parts := strings.Split(clean, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("malformed vendor reference (must contain bundle and concept): %s", target)
	}

	if vendorRoot == "" {
		vendorRoot = filepath.Join(".okf", "vendor")
	}

	// 1. Check disk: does 2-segment bundle exist (.okf/vendor/org/bundle/index.md)?
	if len(parts) >= 3 {
		if _, err := os.Stat(filepath.Join(vendorRoot, parts[0], parts[1], "index.md")); err == nil {
			return parts[0] + "/" + parts[1], strings.Join(parts[2:], "/"), nil
		}
	}

	// 2. Check disk: does 1-segment bundle exist (.okf/vendor/bundle/index.md)?
	if _, err := os.Stat(filepath.Join(vendorRoot, parts[0], "index.md")); err == nil {
		return parts[0], strings.Join(parts[1:], "/"), nil
	}

	// 3. Fallback heuristic: standard concept directories
	if len(parts) >= 3 && !isKnownConceptCategory(parts[1]) {
		return parts[0] + "/" + parts[1], strings.Join(parts[2:], "/"), nil
	}

	return parts[0], strings.Join(parts[1:], "/"), nil
}

func isKnownConceptCategory(s string) bool {
	switch strings.ToLower(s) {
	case "decisions", "architecture", "convention", "roadmap", "project",
		"requirements", "domain", "runbooks", "concepts", "facts", "entities",
		"guides", "playbooks", "specs", "adr", "rfc", "api":
		return true
	default:
		return false
	}
}

// ParseURI parses canonical okf:// URIs or @-prefixed vendor references.
// Examples:
//
//	okf://@peter/django-5-rules/decisions/auth -> (ScopeVendor, "peter/django-5-rules", "decisions/auth", nil)
//	okf://@nextjs-15/decisions/routing         -> (ScopeVendor, "nextjs-15", "decisions/routing", nil)
//	okf://user/preferences                     -> (ScopeUser, "", "preferences", nil)
//	okf://system/compliance                    -> (ScopeSystem, "", "compliance", nil)
//	@peter/django-5-rules/decisions/auth       -> (ScopeVendor, "peter/django-5-rules", "decisions/auth", nil)
func ParseURI(rawURI string) (Scope, string, string, error) {
	if strings.HasPrefix(rawURI, "@") {
		bundleID, conceptID, err := ParseVendorRef(rawURI, "")
		if err != nil {
			return "", "", "", err
		}
		return ScopeVendor, bundleID, conceptID, nil
	}

	if strings.HasPrefix(rawURI, "user:") {
		conceptID := strings.TrimPrefix(rawURI, "user:")
		conceptID = strings.TrimPrefix(conceptID, "/")
		conceptID = strings.TrimSuffix(conceptID, ".md")
		return ScopeUser, "", conceptID, nil
	}
	if strings.HasPrefix(rawURI, "system:") {
		conceptID := strings.TrimPrefix(rawURI, "system:")
		conceptID = strings.TrimPrefix(conceptID, "/")
		conceptID = strings.TrimSuffix(conceptID, ".md")
		return ScopeSystem, "", conceptID, nil
	}

	if !strings.HasPrefix(rawURI, "okf://") {
		return "", "", "", fmt.Errorf("not an okf:// URI or @vendor reference")
	}

	stripped := strings.TrimPrefix(rawURI, "okf://")
	if strings.HasPrefix(stripped, "@") {
		bundleID, conceptID, err := ParseVendorRef(stripped, "")
		if err != nil {
			return "", "", "", err
		}
		return ScopeVendor, bundleID, conceptID, nil
	}

	parts := strings.Split(stripped, "/")
	if len(parts) < 2 {
		return "", "", "", fmt.Errorf("malformed okf:// URI: %s", rawURI)
	}

	scope := Scope(parts[0])
	switch scope {
	case ScopeUser, ScopeProject, ScopeSystem:
		conceptID := strings.Join(parts[1:], "/")
		return scope, "", conceptID, nil
	default:
		return "", "", "", fmt.Errorf("unknown scope in URI: %s", scope)
	}
}

// IsExternalLink checks whether a link href points to an external or scoped target
// (vendor @bundle/..., user:..., system:..., or explicit scheme like okf://, https://, etc.)
// and therefore must never cause broken-link failures in local bundle validation.
func IsExternalLink(href string) bool {
	norm := strings.TrimSpace(href)
	if norm == "" {
		return false
	}
	return strings.Contains(norm, "://") ||
		strings.HasPrefix(norm, "@") ||
		strings.HasPrefix(norm, "user:") ||
		strings.HasPrefix(norm, "system:") ||
		strings.HasPrefix(norm, "mailto:")
}

// ResolveUserDir returns the user-level OKF directory (~/.okf or $OKF_USER_DIR).
func ResolveUserDir() string {
	if dir := os.Getenv("OKF_USER_DIR"); dir != "" {
		return dir
	}
	if homeDir, err := os.UserHomeDir(); err == nil && homeDir != "" {
		return filepath.Join(homeDir, ".okf")
	}
	return ""
}

// ResolveSystemDir returns the system-level OKF directory (/etc/okf or $OKF_SYSTEM_DIR).
func ResolveSystemDir() string {
	if dir := os.Getenv("OKF_SYSTEM_DIR"); dir != "" {
		return dir
	}
	return "/etc/okf"
}

// LayeredSearchOptions configures a multi-scope search across memory layers.
type LayeredSearchOptions struct {
	BundleDir  string
	VendorRoot string
	UserDir    string
	SystemDir  string
	Scope      string // "all", "project", "bundle", "vendor", "user", "system"
	SearchOpts SearchOptions
}

// SearchLayered executes search across multiple memory layers (Project, Vendor, User, System)
// with deterministic shadowing and priority ranking.
func SearchLayered(opts LayeredSearchOptions) ([]SearchResult, error) {
	targetScope := strings.ToLower(strings.TrimSpace(opts.Scope))
	if targetScope == "" {
		targetScope = "all"
	}
	switch targetScope {
	case "all", "project", "bundle", "vendor", "user", "system":
	default:
		return nil, fmt.Errorf("invalid scope %q (allowed: all, project, bundle, vendor, user, system)", opts.Scope)
	}

	searchProject := targetScope == "all" || targetScope == "project" || targetScope == "bundle"
	searchVendor := targetScope == "all" || targetScope == "vendor"
	searchUser := targetScope == "all" || targetScope == "user"
	searchSystem := targetScope == "all" || targetScope == "system"

	var results []SearchResult
	seenConcepts := make(map[string]bool)

	// 1. Search Project Layer (Priority 100)
	if searchProject {
		bDir := opts.BundleDir
		if bDir == "" {
			bDir = "knowledge"
			if info, err := os.Stat("knowledge"); err != nil || !info.IsDir() {
				bDir = "."
			}
		}
		b, err := LoadBundle(bDir)
		if err != nil {
			if targetScope != "all" {
				return nil, fmt.Errorf("error loading bundle: %w", err)
			}
		} else {
			pResults, err := b.SearchAdvanced(opts.SearchOpts)
			if err != nil {
				return nil, fmt.Errorf("search error: %w", err)
			}
			for i := range pResults {
				pResults[i].Scope = ScopeProject
				pResults[i].Priority = PriorityProject
				pResults[i].Origin = "local"
				seenConcepts[pResults[i].ConceptID] = true
				results = append(results, pResults[i])
			}
			for id := range b.Concepts {
				seenConcepts[id] = true
			}
		}
	}

	// 2. Search Vendor Layer (Priority 70)
	if searchVendor {
		vRoot := opts.VendorRoot
		if vRoot == "" {
			vRoot = filepath.Join(".okf", "vendor")
		}
		if info, err := os.Stat(vRoot); err == nil && info.IsDir() {
			_ = filepath.Walk(vRoot, func(p string, fi os.FileInfo, err error) error {
				if err != nil || !fi.IsDir() {
					return nil
				}
				if _, err := os.Stat(filepath.Join(p, "index.md")); err == nil {
					rel, _ := filepath.Rel(vRoot, p)
					bundleID := filepath.ToSlash(rel)
					vb, err := LoadBundle(p)
					if err == nil {
						vResults, _ := vb.SearchAdvanced(opts.SearchOpts)
						for _, vr := range vResults {
							rawID := vr.ConceptID
							if !seenConcepts[rawID] {
								vr.Scope = ScopeVendor
								vr.Priority = PriorityVendor
								vr.Origin = fmt.Sprintf("@%s", bundleID)
								vr.ConceptID = fmt.Sprintf("@%s/%s", bundleID, rawID)
								results = append(results, vr)
								seenConcepts[rawID] = true
							}
						}
						for id := range vb.Concepts {
							seenConcepts[id] = true
						}
					}
					return filepath.SkipDir
				}
				return nil
			})
		}
	}

	// 3. Search User Layer (Priority 50)
	if searchUser {
		uDir := opts.UserDir
		if uDir == "" {
			uDir = ResolveUserDir()
		}
		if uDir != "" {
			if info, err := os.Stat(uDir); err == nil && info.IsDir() {
				cand := uDir
				if kInfo, kErr := os.Stat(filepath.Join(uDir, "knowledge")); kErr == nil && kInfo.IsDir() {
					cand = filepath.Join(uDir, "knowledge")
				}
				if ub, err := loadTrustedBundle(cand); err == nil {
					uResults, _ := ub.SearchAdvanced(opts.SearchOpts)
					for _, ur := range uResults {
						rawID := ur.ConceptID
						if !seenConcepts[rawID] {
							ur.Scope = ScopeUser
							ur.Priority = PriorityUser
							ur.Origin = "user"
							ur.ConceptID = fmt.Sprintf("user:%s", rawID)
							results = append(results, ur)
							seenConcepts[rawID] = true
						}
					}
					for id := range ub.Concepts {
						seenConcepts[id] = true
					}
				}
			}
		}
	}

	// 4. Search System Layer (Priority 10)
	if searchSystem {
		sDir := opts.SystemDir
		if sDir == "" {
			sDir = ResolveSystemDir()
		}
		if sDir != "" {
			if info, err := os.Stat(sDir); err == nil && info.IsDir() {
				cand := sDir
				if kInfo, kErr := os.Stat(filepath.Join(sDir, "knowledge")); kErr == nil && kInfo.IsDir() {
					cand = filepath.Join(sDir, "knowledge")
				}
				if sb, err := loadTrustedBundle(cand); err == nil {
					sResults, _ := sb.SearchAdvanced(opts.SearchOpts)
					for _, sr := range sResults {
						rawID := sr.ConceptID
						if !seenConcepts[rawID] {
							sr.Scope = ScopeSystem
							sr.Priority = PrioritySystem
							sr.Origin = "system"
							sr.ConceptID = fmt.Sprintf("system:%s", rawID)
							results = append(results, sr)
							seenConcepts[rawID] = true
						}
					}
					for id := range sb.Concepts {
						seenConcepts[id] = true
					}
				}
			}
		}
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Priority != results[j].Priority {
			return results[i].Priority > results[j].Priority
		}
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		return results[i].ConceptID < results[j].ConceptID
	})

	limit := opts.SearchOpts.Limit
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// ScopedConceptResult represents the outcome of resolving a concept identifier across scopes.
type ScopedConceptResult struct {
	Concept   *Concept
	Bundle    *Bundle
	BundleDir string
	Scope     Scope
}

// ResolveScopedConcept resolves a concept identifier across project, vendor, user, and system scopes.
// It accepts local concept IDs (e.g. 'architecture/database'), vendor references (e.g. '@nextjs-15/decisions/routing'),
// user-scoped IDs ('user:preferences/style'), system-scoped IDs ('system:corp/policies'), or canonical URNs ('okf://...').
func ResolveScopedConcept(rawID, defaultBundleDir, vendorRoot, userDir, systemDir string) (*ScopedConceptResult, error) {
	trimmed := strings.TrimSpace(rawID)
	if trimmed == "" {
		return nil, fmt.Errorf("concept ID cannot be empty")
	}

	normID := NormalizeVendorLink(trimmed)
	showScope := ScopeProject
	var showBundleID string
	var conceptID string

	if strings.HasPrefix(normID, "okf://") {
		var parseErr error
		showScope, showBundleID, conceptID, parseErr = ParseURI(normID)
		if parseErr != nil {
			return nil, parseErr
		}
		if err := ValidateConceptID(conceptID); err != nil {
			return nil, err
		}
	} else {
		if err := ValidateConceptID(trimmed); err != nil {
			return nil, err
		}
		conceptID = strings.TrimSuffix(trimmed, ".md")
	}

	bundleDir := defaultBundleDir
	if bundleDir == "" {
		bundleDir = "knowledge"
		if info, err := os.Stat("knowledge"); err != nil || !info.IsDir() {
			bundleDir = "."
		}
	}

	switch showScope {
	case ScopeVendor:
		if vendorRoot == "" {
			vendorRoot = filepath.Join(".okf", "vendor")
		}
		bundleDir = filepath.Join(vendorRoot, filepath.FromSlash(showBundleID))
	case ScopeUser:
		if userDir == "" {
			userDir = ResolveUserDir()
		}
		bundleDir = userDir
		if kInfo, kErr := os.Stat(filepath.Join(bundleDir, "knowledge")); kErr == nil && kInfo.IsDir() {
			bundleDir = filepath.Join(bundleDir, "knowledge")
		}
	case ScopeSystem:
		if systemDir == "" {
			systemDir = ResolveSystemDir()
		}
		bundleDir = systemDir
		if kInfo, kErr := os.Stat(filepath.Join(bundleDir, "knowledge")); kErr == nil && kInfo.IsDir() {
			bundleDir = filepath.Join(bundleDir, "knowledge")
		}
	}

	load := LoadBundle
	if showScope == ScopeUser || showScope == ScopeSystem {
		load = loadTrustedBundle
	}
	b, err := load(bundleDir)
	if err != nil {
		return nil, fmt.Errorf("error loading bundle from %q: %w", bundleDir, err)
	}

	c, ok := b.Concepts[conceptID]
	if !ok {
		return nil, fmt.Errorf("concept '%s' not found in '%s'", conceptID, bundleDir)
	}

	return &ScopedConceptResult{
		Concept:   c,
		Bundle:    b,
		BundleDir: bundleDir,
		Scope:     showScope,
	}, nil
}
