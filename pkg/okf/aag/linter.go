package aag

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// RFC 2119 valid prefix keywords for invariants
var validModalPrefixes = []string{
	"MUST", "MUST NOT", "NEVER", "PREFER", "ALWAYS", "SHOULD", "MAY", "!",
}

// LintFile reads a file and lints it according to AAG rules.
func LintFile(path string, opts LinterOptions) (*LintResult, error) {
	cleanPath := filepath.Clean(path)
	// #nosec G304 -- cleanPath is explicitly provided by user/caller for CLI linting
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", cleanPath, err)
	}
	res := LintContent(cleanPath, data, opts)
	return res, nil
}

// LintContent performs AAG linting on raw markdown bytes.
func LintContent(filename string, content []byte, opts LinterOptions) *LintResult {
	if opts.BudgetLimit <= 0 {
		opts.BudgetLimit = DefaultBudgetLimit
	}

	res := &LintResult{
		Path:     filename,
		Findings: make([]Finding, 0),
	}

	rawText := string(content)
	totalTokens := EstimateTokens(rawText)
	aagBlockTokens := EstimateTokens(ExtractAAGBlock(rawText))

	res.TokenStats = TokenStats{
		EstimatedTokens: aagBlockTokens,
		BudgetLimit:     opts.BudgetLimit,
		BudgetExceeded:  aagBlockTokens > opts.BudgetLimit,
	}

	// Rule AAG-005: Token Budget Gate
	if res.TokenStats.BudgetExceeded {
		res.Findings = append(res.Findings, Finding{
			RuleID:   RuleTokenBudgetGate,
			Severity: SeverityError,
			Line:     1,
			Message:  fmt.Sprintf("AAG working memory token count (%d) exceeds budget limit (%d tokens; total file: %d)", aagBlockTokens, opts.BudgetLimit, totalTokens),
		})
	}

	scanner := bufio.NewScanner(bytes.NewReader(content))
	lineNum := 0
	currentSection := ""
	hasSectionZero := false
	hasMermaidAssertion := false

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Check for Section headers
		if strings.HasPrefix(trimmed, "#") {
			headerText := strings.TrimLeft(trimmed, "# ")
			currentSection = headerText
			if strings.HasPrefix(headerText, "0.") || strings.Contains(strings.ToLower(headerText), "domain codex") {
				hasSectionZero = true
			}
		}

		// Rule AAG-001: ASCII Only Invariance
		checkAsciiOnly(lineNum, line, res)

		// Check Section 0 Domain Codex for Mermaid assertion
		if strings.HasPrefix(currentSection, "0.") || strings.Contains(strings.ToLower(currentSection), "domain codex") {
			lower := strings.ToLower(line)
			if strings.Contains(lower, "mermaid") {
				hasMermaidAssertion = true
			}
		}

		// Rule AAG-002: Explicit RFC 2119 Modal Verbs in Invariants
		if isInvariantSection(currentSection) && strings.HasPrefix(trimmed, "- ") {
			ruleText := strings.TrimSpace(trimmed[2:])
			checkExplicitModal(lineNum, ruleText, res)
		}

		// Rule AAG-003: Tool Signature Integrity
		checkToolSignatures(lineNum, line, res)
	}

	// Rule AAG-004: Mermaid Assertion Check in Domain Codex
	if hasSectionZero && !hasMermaidAssertion {
		res.Findings = append(res.Findings, Finding{
			RuleID:   RuleMermaidFormat,
			Severity: SeverityError,
			Line:     1,
			Message:  "Section 0 (Domain Codex) must specify Mermaid diagram syntax assertion ('FORMAT: diagrams => ASSERT(syntax == mermaid)')",
		})
	}

	// Calculate error and warning counts
	for _, f := range res.Findings {
		if f.Severity == SeverityError || opts.Strict {
			res.ErrorCount++
		} else {
			res.WarnCount++
		}
	}

	res.Passed = res.ErrorCount == 0
	return res
}

func checkAsciiOnly(lineNum int, line string, res *LintResult) {
	for _, r := range line {
		if r > unicode.MaxASCII {
			res.Findings = append(res.Findings, Finding{
				RuleID:   RuleAsciiOnly,
				Severity: SeverityError,
				Line:     lineNum,
				Message:  fmt.Sprintf("Non-ASCII / multi-byte character '%c' (U+%04X) detected; use standard ASCII operators", r, r),
				Snippet:  strings.TrimSpace(line),
			})
			return
		}
	}
}

func isInvariantSection(section string) bool {
	lower := strings.ToLower(section)
	return strings.Contains(lower, "constraints") ||
		strings.Contains(lower, "invariants") ||
		strings.Contains(lower, "rfc 2119") ||
		strings.HasPrefix(lower, "1.")
}

func checkExplicitModal(lineNum int, ruleText string, res *LintResult) {
	upper := strings.ToUpper(ruleText)
	hasValidModal := false
	for _, modal := range validModalPrefixes {
		if strings.HasPrefix(upper, modal+" ") || upper == modal || strings.HasPrefix(upper, "`"+modal) {
			hasValidModal = true
			break
		}
	}

	if !hasValidModal {
		res.Findings = append(res.Findings, Finding{
			RuleID:   RuleExplicitModal,
			Severity: SeverityError,
			Line:     lineNum,
			Message:  fmt.Sprintf("Rule must begin with explicit RFC 2119 modal verb (MUST, NEVER, PREFER, etc.): %q", ruleText),
			Snippet:  ruleText,
		})
	}
}

func checkToolSignatures(lineNum int, line string, res *LintResult) {
	// Check for unclosed tool invocations (e.g. `okf_search(` without matching closing paren on the line)
	if strings.Contains(line, "okf_") || strings.Contains(line, "tool_") {
		openParen := strings.Count(line, "(")
		closeParen := strings.Count(line, ")")
		if openParen > closeParen {
			res.Findings = append(res.Findings, Finding{
				RuleID:   RuleToolSignature,
				Severity: SeverityError,
				Line:     lineNum,
				Message:  "Malformed tool invocation with unclosed parenthesis",
				Snippet:  strings.TrimSpace(line),
			})
		}
	}
}

// EstimateTokens calculates an approximate BPE token count for text.
func EstimateTokens(text string) int {
	trimmed := strings.TrimSpace(text)
	if len(trimmed) == 0 {
		return 0
	}
	charEstimate := (len(trimmed) + 3) / 4
	words := len(strings.Fields(trimmed))
	wordEstimate := int(float64(words) * 1.2)

	if charEstimate > wordEstimate {
		return charEstimate
	}
	return wordEstimate
}

// ExtractAAGBlock extracts the operational AAG instruction block between delimiters if present.
func ExtractAAGBlock(content string) string {
	beginIdx := strings.Index(content, "<!-- BEGIN OKF AGENT MEMORY -->")
	endIdx := strings.Index(content, "<!-- END OKF AGENT MEMORY -->")
	if beginIdx != -1 && endIdx != -1 && endIdx > beginIdx {
		return content[beginIdx+len("<!-- BEGIN OKF AGENT MEMORY -->") : endIdx]
	}
	return content
}
