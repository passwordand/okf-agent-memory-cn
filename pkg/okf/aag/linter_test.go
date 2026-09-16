package aag

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLintValidRepositories(t *testing.T) {
	// Paths relative to repository root
	validFiles := []string{
		"../../../AGENTS.md",
		"../../../examples/software/AGENTS.md",
		"../../../examples/books/AGENTS.md",
		"../../../examples/coaching/AGENTS.md",
		"../../../pkg/okf/assets/templates/AGENTS.md",
	}

	opts := LinterOptions{
		BudgetLimit: DefaultBudgetLimit,
		Strict:      true,
	}

	for _, relPath := range validFiles {
		absPath, err := filepath.Abs(relPath)
		if err != nil {
			t.Fatalf("failed to resolve path %s: %v", relPath, err)
		}
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			t.Logf("skipping nonexistent test path %s", absPath)
			continue
		}

		result, err := LintFile(absPath, opts)
		if err != nil {
			t.Fatalf("unexpected error linting %s: %v", relPath, err)
		}

		if !result.Passed {
			t.Errorf("expected %s to pass linting, got %d errors, %d warnings: %+v",
				relPath, result.ErrorCount, result.WarnCount, result.Findings)
		}

		if result.TokenStats.BudgetExceeded {
			t.Errorf("expected %s to be within budget %d, got %d tokens",
				relPath, opts.BudgetLimit, result.TokenStats.EstimatedTokens)
		}
	}
}

func TestLintRule_AAG001_AsciiOnly(t *testing.T) {
	invalidContent := `# AGENTS.md
## 0. Domain Codex
- FORMAT: diagrams ➔ ASSERT(syntax == mermaid)
- STATUS: 🚨 alerts active
`
	result := LintContent("test.md", []byte(invalidContent), LinterOptions{BudgetLimit: 150})
	if result.Passed {
		t.Fatalf("expected AAG-001 violation, but passed")
	}

	found := false
	for _, f := range result.Findings {
		if f.RuleID == RuleAsciiOnly {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected finding for RuleAsciiOnly (AAG-001), got: %+v", result.Findings)
	}
}

func TestLintRule_AAG002_ExplicitModal(t *testing.T) {
	invalidContent := `# AGENTS.md
## 1. Behavioral Invariants & Constraints (RFC 2119)
- Please make sure to search before editing files.
- Try to avoid using raw grep or list_dir if possible.
`
	result := LintContent("test.md", []byte(invalidContent), LinterOptions{BudgetLimit: 150})
	if result.Passed {
		t.Fatalf("expected AAG-002 violation, but passed")
	}

	foundCount := 0
	for _, f := range result.Findings {
		if f.RuleID == RuleExplicitModal {
			foundCount++
		}
	}
	if foundCount < 2 {
		t.Errorf("expected at least 2 AAG-002 findings, got %d: %+v", foundCount, result.Findings)
	}
}

func TestLintRule_AAG003_ToolSignature(t *testing.T) {
	validContent := `# AGENTS.md
## 2. Guard Clauses & Scope Governance
- ON edit(@pkg/):
    IF first_visit(@pkg/) => okf_search(for_path=@pkg/)
`
	validResult := LintContent("test.md", []byte(validContent), LinterOptions{BudgetLimit: 150})
	for _, f := range validResult.Findings {
		if f.RuleID == RuleToolSignature {
			t.Errorf("unexpected AAG-003 finding on valid content: %+v", f)
		}
	}

	invalidContent := `# AGENTS.md
## 2. Guard Clauses & Scope Governance
- ON edit(@pkg/):
    IF first_visit(@pkg/) => okf_search(unclosed parameter
`
	invalidResult := LintContent("test.md", []byte(invalidContent), LinterOptions{BudgetLimit: 150})
	found := false
	for _, f := range invalidResult.Findings {
		if f.RuleID == RuleToolSignature {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected AAG-003 finding for malformed tool signature, got: %+v", invalidResult.Findings)
	}
}

func TestLintRule_AAG004_MermaidFormat(t *testing.T) {
	missingMermaid := `# AGENTS.md
## 0. Project & Domain Codex
- TONE: style == direct_concise
- GOAL: maintain(high_integrity)
`
	result := LintContent("test.md", []byte(missingMermaid), LinterOptions{BudgetLimit: 150})
	found := false
	for _, f := range result.Findings {
		if f.RuleID == RuleMermaidFormat {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected AAG-004 finding for missing Mermaid assertion in Section 0, got: %+v", result.Findings)
	}
}

func TestLintRule_AAG005_TokenBudget(t *testing.T) {
	content := `# AGENTS.md
## 0. Project & Domain Codex
- TONE: style == direct_concise, zero_pleasantries
- FORMAT: diagrams => ASSERT(syntax == mermaid, ELSE=STOP("Mermaid required."))
- GOAL: maintain(high_factual_integrity, domain_neutrality)

## 1. Behavioral Invariants & Constraints (RFC 2119)
- MUST execute okf_search(query=keywords, limit=3) before proposing architecture.
- NEVER scan knowledge/ via list_dir, grep_search, find, or raw file readers.
`
	// Low budget to force failure
	result := LintContent("test.md", []byte(content), LinterOptions{BudgetLimit: 20})
	if result.Passed {
		t.Fatalf("expected AAG-005 failure due to budget limit, but passed")
	}
	if !result.TokenStats.BudgetExceeded {
		t.Errorf("expected BudgetExceeded to be true")
	}

	found := false
	for _, f := range result.Findings {
		if f.RuleID == RuleTokenBudgetGate {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected AAG-005 finding, got: %+v", result.Findings)
	}
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		input string
		min   int
		max   int
	}{
		{"", 0, 0},
		{"Hello world", 1, 4},
		{"- MUST execute `okf_search(query=keywords, limit=3)` before proposing architecture.", 10, 25},
	}

	for _, tt := range tests {
		tokens := EstimateTokens(tt.input)
		if tokens < tt.min || tokens > tt.max {
			t.Errorf("EstimateTokens(%q) = %d, expected between %d and %d", tt.input, tokens, tt.min, tt.max)
		}
	}
}
