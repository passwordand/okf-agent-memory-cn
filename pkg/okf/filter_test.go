package okf

import (
	"testing"
	"time"
)

func TestParseRelativeDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{"14d", 14 * 24 * time.Hour, false},
		{"2w", 14 * 24 * time.Hour, false},
		{"1w", 7 * 24 * time.Hour, false},
		{"1m", 30 * 24 * time.Hour, false},
		{"3m", 90 * 24 * time.Hour, false},
		{"1y", 365 * 24 * time.Hour, false},
		{"48h", 48 * time.Hour, false},
		{"", 0, true},
		{"invalid", 0, true},
		{"-5d", 0, true},
	}

	for _, tt := range tests {
		got, err := ParseRelativeDuration(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseRelativeDuration(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.expected {
			t.Errorf("ParseRelativeDuration(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestConceptMatchesFilter(t *testing.T) {
	c := &Concept{
		ID:          "auth/oauth2",
		Type:        "Decision",
		Title:       "OAuth2 Implementation",
		Description: "OAuth2 authentication details",
		Status:      "stable",
		Governance:  "constraint",
		Tags:        []string{"auth", "security", "jwt"},
		StaleAfter:  "2026-10-15",
		Generated: &Generated{
			By: "agent/claude",
			At: "2026-09-01T12:00:00Z",
		},
		Verified: []Verified{
			{By: "human:reviewer", At: "2026-09-05T10:00:00Z"},
		},
		Extra: map[string]any{
			"team": "core-infra",
		},
	}

	tests := []struct {
		filter string
		match  bool
	}{
		// Basic exact matches
		{"type=Decision", true},
		{"type=Fact", false},
		{"status=stable", true},
		{"status=deprecated", false},
		{"governance=constraint", true},
		{"id=auth/oauth2", true},

		// Inequality matches
		{"type!=Fact", true},
		{"type!=Decision", false},
		{"status!=deprecated", true},

		// Tag array matching
		{"tags=security", true},
		{"tags=database", false},
		{"tags!=database", true},

		// Nested fields: verified
		{"verified.by=human:reviewer", true},
		{"verified.by=bot", false},
		{"verified.by!=null", true},
		{"verified.by!=nil", true},
		{"verified!=null", true},

		// Nested fields: generated
		{"generated.by=agent/claude", true},
		{"generated.by!=null", true},

		// Extra fields
		{"team=core-infra", true},
		{"team!=security", true},

		// Empty / null checks for unset fields
		{"nonexistent=null", true},
		{"nonexistent!=null", false},

		// Multiple comma-separated conditions
		{"type=Decision,governance=constraint", true},
		{"type=Decision,status=deprecated", false},
	}

	for _, tt := range tests {
		got, err := c.MatchesFilter(tt.filter)
		if err != nil {
			t.Errorf("MatchesFilter(%q) returned unexpected error: %v", tt.filter, err)
			continue
		}
		if got != tt.match {
			t.Errorf("MatchesFilter(%q) = %v, want %v", tt.filter, got, tt.match)
		}
	}
}

func TestConceptMatchesFilterCustomListFields(t *testing.T) {
	c := &Concept{
		ID: "search/ranking",
		Extra: map[string]any{
			"topics": []any{"retrieval", "Ranking"},
			"counts": []any{float64(1), float64(2)},
			"repos":  []string{"contextopia", "easygov"},
			"empty":  []any{},
			"state":  "decided",
		},
	}

	tests := []struct {
		filter string
		match  bool
	}{
		{"topics=retrieval", true},
		{"topics=RETRIEVAL", true},
		{"topics=ranking", true},
		{"topics=indexing", false},
		{"topics!=indexing", true},
		{"topics!=retrieval", false},

		{"counts=2", true},
		{"counts=3", false},

		{"repos=easygov", true},
		{"repos=other", false},

		{"topics=null", false},
		{"topics!=null", true},
		{"empty=null", true},
		{"empty!=null", false},

		{"state=decided", true},
		{"state=open", false},

		{"missing=retrieval", false},
		{"missing!=retrieval", true},

		{"topics=retrieval,state=decided", true},
		{"topics=retrieval,state=open", false},
	}

	for _, tt := range tests {
		got, err := c.MatchesFilter(tt.filter)
		if err != nil {
			t.Errorf("MatchesFilter(%q) returned unexpected error: %v", tt.filter, err)
			continue
		}
		if got != tt.match {
			t.Errorf("MatchesFilter(%q) = %v, want %v", tt.filter, got, tt.match)
		}
	}
}

func TestConceptMatchesFilterUsesFrontmatterKeyNames(t *testing.T) {
	builtIn := &Concept{
		ID:          "auth/jwt",
		Description: "JWT validation",
		Tags:        []string{"security"},
		CodeRefs:    []string{"pkg/auth/*.go"},
	}
	for _, filter := range []string{"tag=security", "code_ref=pkg/auth/*.go", "desc=JWT validation"} {
		if ok, _ := builtIn.MatchesFilter(filter); ok {
			t.Errorf("MatchesFilter(%q) must not alias a frontmatter key", filter)
		}
	}
	if ok, _ := builtIn.MatchesFilter("description=JWT validation"); !ok {
		t.Error("MatchesFilter(\"description=...\") should match the description")
	}

	custom := &Concept{
		ID:    "notes/solo",
		Extra: map[string]any{"tag": "solo", "code_ref": "x", "desc": "short"},
	}
	for _, filter := range []string{"tag=solo", "code_ref=x", "desc=short"} {
		if ok, _ := custom.MatchesFilter(filter); !ok {
			t.Errorf("MatchesFilter(%q) should match the custom field of that name", filter)
		}
	}
}

func TestConceptMatchesFilterMultipleTags(t *testing.T) {
	both := &Concept{ID: "a", Tags: []string{"ci", "ui"}}
	onlyCI := &Concept{ID: "b", Tags: []string{"ci"}}

	// Clauses are ANDed, so repeating the key requires every tag.
	const allOf = "tags=ci,tags=ui"
	if ok, err := both.MatchesFilter(allOf); err != nil || !ok {
		t.Errorf("MatchesFilter(%q) on both tags = %v, %v; want true, nil", allOf, ok, err)
	}
	if ok, err := onlyCI.MatchesFilter(allOf); err != nil || ok {
		t.Errorf("MatchesFilter(%q) on one tag = %v, %v; want false, nil", allOf, ok, err)
	}

	// A comma starts a new clause; a bare value is not a clause.
	if _, err := both.MatchesFilter("tags=ci,ui"); err == nil {
		t.Error("MatchesFilter(\"tags=ci,ui\") must report the clause without an operator")
	}
}

func TestConceptMatchesFilterVerifiedByHuman(t *testing.T) {
	verifiedBy := func(by string) *Concept {
		return &Concept{ID: "note", Verified: []Verified{{By: by, At: "2026-01-01"}}}
	}
	tests := []struct {
		by    string
		match bool
	}{
		{"human:lead", true},
		{"human/lead", true},
		{"Human:Lead", true},
		{"agent/claude", false},
		{"humanoid-bot", false},
	}
	for _, tt := range tests {
		got, err := verifiedBy(tt.by).MatchesFilter("verified.by=human")
		if err != nil {
			t.Fatalf("MatchesFilter returned error: %v", err)
		}
		if got != tt.match {
			t.Errorf("verified.by=human on %q = %v, want %v", tt.by, got, tt.match)
		}
	}
}

func TestConceptMatchesFilterParsedCustomLists(t *testing.T) {
	for name, fields := range map[string]string{
		"flow list":  "topics: [retrieval, ranking]",
		"block list": "topics:\n  - retrieval\n  - ranking",
	} {
		t.Run(name, func(t *testing.T) {
			c, err := ParseConcept("note.md", "---\ntype: Fact\n"+fields+"\n---\n\n# Body\n")
			if err != nil {
				t.Fatalf("ParseConcept failed: %v", err)
			}
			if ok, _ := c.MatchesFilter("topics=retrieval"); !ok {
				t.Errorf("topics=retrieval should match parsed list %#v", c.Extra["topics"])
			}
			if ok, _ := c.MatchesFilter("topics=indexing"); ok {
				t.Errorf("topics=indexing must not match parsed list %#v", c.Extra["topics"])
			}
		})
	}
}

func TestConceptIsStaleWithin(t *testing.T) {
	// Base date: 2026-09-25
	baseDate := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		staleAfter  string
		within      time.Duration
		expectMatch bool
	}{
		{
			name:        "Stale 5 days ago (already stale)",
			staleAfter:  "2026-09-20",
			within:      14 * 24 * time.Hour,
			expectMatch: true,
		},
		{
			name:        "Stale in 10 days (within 14d horizon)",
			staleAfter:  "2026-10-05",
			within:      14 * 24 * time.Hour,
			expectMatch: true,
		},
		{
			name:        "Stale in 20 days (outside 14d horizon)",
			staleAfter:  "2026-10-15",
			within:      14 * 24 * time.Hour,
			expectMatch: false,
		},
		{
			name:        "No stale_after declared",
			staleAfter:  "",
			within:      14 * 24 * time.Hour,
			expectMatch: false,
		},
	}

	for _, tt := range tests {
		c := &Concept{
			ID:         "test/doc",
			StaleAfter: tt.staleAfter,
		}
		got := c.IsStaleWithin(baseDate, tt.within)
		if got != tt.expectMatch {
			t.Errorf("%s: IsStaleWithin(..., %v) = %v, want %v", tt.name, tt.within, got, tt.expectMatch)
		}
	}
}
