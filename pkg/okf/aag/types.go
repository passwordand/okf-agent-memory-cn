package aag

// RuleID represents standardized AAG validation rule codes.
type RuleID string

const (
	RuleAsciiOnly       RuleID = "AAG-001" // Flag non-ASCII or multi-byte unicode characters
	RuleExplicitModal   RuleID = "AAG-002" // Invariant rules must use explicit RFC 2119 modal verbs
	RuleToolSignature   RuleID = "AAG-003" // Tool calls must follow valid identifier(arg=val) signatures
	RuleMermaidFormat   RuleID = "AAG-004" // Domain codex must specify Mermaid diagram assertions
	RuleTokenBudgetGate RuleID = "AAG-005" // AGENTS.md working memory must stay within token budget
)

// Severity represents finding severity level.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// Finding represents a single linting violation or diagnostic issue.
type Finding struct {
	RuleID   RuleID   `json:"rule_id"`
	Severity Severity `json:"severity"`
	Line     int      `json:"line"`
	Message  string   `json:"message"`
	Snippet  string   `json:"snippet,omitempty"`
}

// TokenStats tracks token estimation and budget consumption.
type TokenStats struct {
	EstimatedTokens int  `json:"estimated_tokens"`
	BudgetLimit     int  `json:"budget_limit"`
	BudgetExceeded  bool `json:"budget_exceeded"`
}

// LintResult contains the complete output of an AAG lint run.
type LintResult struct {
	Path       string     `json:"path"`
	Passed     bool       `json:"passed"`
	ErrorCount int        `json:"error_count"`
	WarnCount  int        `json:"warn_count"`
	Findings   []Finding  `json:"findings"`
	TokenStats TokenStats `json:"token_stats"`
}

// DefaultBudgetLimit is the default token limit for full AGENTS.md files.
const DefaultBudgetLimit = 400

// LinterOptions configures the AAG linter behavior.
type LinterOptions struct {
	BudgetLimit int  // Max token budget (default: 400, or 150 for strict micro-syntax)
	Strict      bool // Treat warnings as errors
}
