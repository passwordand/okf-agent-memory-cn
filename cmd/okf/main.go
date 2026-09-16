package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/okf-memory/okf-agent-memory/pkg/okf"
)

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func init() {
	if Version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok {
			if info.Main.Version != "" && info.Main.Version != "(devel)" {
				Version = info.Main.Version
			}
			for _, setting := range info.Settings {
				if setting.Key == "vcs.revision" && Commit == "none" {
					if len(setting.Value) > 7 {
						Commit = setting.Value[:7]
					} else {
						Commit = setting.Value
					}
				}
				if setting.Key == "vcs.time" && Date == "unknown" {
					Date = setting.Value
				}
			}
		}
	}
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "validate":
		cmdValidate(args)
	case "search":
		cmdSearch(args)
	case "show":
		cmdShow(args)
	case "create":
		cmdCreate(args)
	case "update":
		cmdUpdate(args)
	case "relate":
		cmdRelate(args)
	case "init":
		cmdInit(args)
	case "bootstrap":
		cmdBootstrap(args)
	case "agents":
		cmdAgents(args)
	case "mcp":
		cmdMCP(args)
	case "version", "--version", "-v":
		fmt.Printf("okf version %s (OKF v0.2 specification)\n", Version)
	case "help", "--help", "-h":
		if len(os.Args) > 2 {
			sub := os.Args[2]
			switch sub {
			case "validate":
				printValidateUsage()
			case "search":
				printSearchUsage()
			case "show":
				printShowUsage()
			case "create":
				printCreateUsage()
			case "update":
				printUpdateUsage()
			case "relate":
				printRelateUsage()
			case "init":
				printInitUsage()
			case "bootstrap":
				printBootstrapUsage()
			case "agents":
				printAgentsUsage()
			case "mcp":
				printMCPUsage()
			default:
				printUsage()
			}
			return
		}
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command '%s'\n\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func isHelpArg(arg string) bool {
	return arg == "--help" || arg == "-h" || arg == "help"
}

func hasHelpFlag(args []string) bool {
	for _, a := range args {
		if isHelpArg(a) {
			return true
		}
	}
	return false
}

func printValidateUsage() {
	fmt.Printf(`Validate an OKF knowledge bundle and agent workspace for conformance and graph health.

Usage:
  okf validate [bundle] [flags]

Arguments:
  [bundle]               Path to OKF bundle directory or workspace root (default: 'knowledge' or '.')

Flags:
  --strict               Gate connectivity warnings, broken links, orphans, and trust gaps as errors
  --drift                Check descriptions and code_refs for drift between index.md and concepts
  --stale                Gate expired review dates (stale_after) as errors
  --agents               Validate AGENTS.md against AAG rules (AAG-001 to AAG-005) and SSoT tool symlinks
  --json                 Output validation results as structured JSON

Examples:
  okf validate knowledge --strict --drift
  okf validate --agents --strict .
  okf validate knowledge --json
`)
}

func printSearchUsage() {
	fmt.Printf(`Search concepts using in-memory BM25 scoring or filter by code path.

Usage:
  okf search <query> [bundle] [flags]
  okf search --for-path <path> [bundle] [flags]

Arguments:
  <query>                Search terms to match against concept titles, descriptions, and bodies
  [bundle]               Path to OKF bundle directory (default: 'knowledge' or '.')

Flags:
  --for-path <path>      Discover concepts governing a specific file path via code_refs
  --limit <N>            Maximum number of search results to return (default: 10)
  --json                 Output results as machine-readable JSON array with BM25 scores

Examples:
  okf search "authentication jwt" knowledge --limit 3
  okf search --for-path pkg/auth/service.go knowledge --json
  okf search "database connection" --limit 5
`)
}

func printShowUsage() {
	fmt.Printf(`Display concept details, metadata, relationships, and body content.

Usage:
  okf show <concept-id> [bundle] [flags]

Arguments:
  <concept-id>           Unique concept identifier (e.g. 'architecture/database' or 'decisions/adr-001')
  [bundle]               Path to OKF bundle directory (default: 'knowledge' or '.')

Flags:
  --raw                  Output verbatim raw markdown file content
  --json                 Output parsed concept metadata and body as JSON

Examples:
  okf show architecture/database knowledge
  okf show decisions/adr-001 --raw
  okf show architecture/auth --json
`)
}

func printCreateUsage() {
	fmt.Printf(`Create a new concept with automated frontmatter, timestamps, and log bookkeeping.

Usage:
  okf create <concept-id> [bundle] [flags]

Arguments:
  <concept-id>           Unique identifier for the concept (e.g. 'decisions/adr-001' or 'architecture/caching')
  [bundle]               Path to OKF bundle directory (default: 'knowledge' or '.')

Flags:
  --type <type>          Semantic concept type (e.g. 'Decision', 'Architecture', 'Fact', 'Requirement', 'Bug') [default: Fact]
  --title <title>        Human-readable title (defaults to concept basename)
  --desc <desc>          One-sentence summary of the concept
  --body <body>          Markdown body content
  --tags <tags>          Comma-separated list of searchable tags (e.g. 'auth,security,jwt')
  --actor <actor>        Author provenance identifier (default: 'agent/cli')
  --no-log               Skip appending an entry to knowledge/log.md
  --no-index             Skip updating immediate parent directory index.md
  --json                 Emit machine-readable JSON result

Examples:
  okf create decisions/adr-001 knowledge --type Decision --title "Database Architecture" --desc "Use PostgreSQL with connection pooling."
  okf create architecture/caching --type Architecture --desc "Redis cluster configuration." --json
`)
}

func printUpdateUsage() {
	fmt.Printf(`Update an existing concept's metadata, description, or body with automated bookkeeping.

Usage:
  okf update <concept-id> [bundle] [flags]

Arguments:
  <concept-id>           Unique identifier of existing concept to mutate
  [bundle]               Path to OKF bundle directory (default: 'knowledge' or '.')

Flags:
  --desc <desc>          Update the one-sentence description
  --title <title>        Update the title
  --body <body>          Update the markdown body content
  --type <type>          Update the concept type
  --status <status>      Set concept status (e.g. 'active', 'deprecated', 'draft')
  --tags <tags>          Replace tags with comma-separated list
  --actor <actor>        Author provenance identifier (default: 'agent/cli')
  --no-log               Skip appending to knowledge/log.md
  --no-index             Skip updating parent directory index.md
  --json                 Emit machine-readable JSON result

Examples:
  okf update architecture/database knowledge --desc "Migrated from SQLite to PostgreSQL 16 on RDS."
  okf update decisions/adr-001 --status deprecated --desc "Superseded by adr-008." --json
`)
}

func printRelateUsage() {
	fmt.Printf(`Connect two concepts with a relative link and semantic context.

Usage:
  okf relate <source-id> <target-id> [bundle] --desc <context> [flags]

Arguments:
  <source-id>            Source concept identifier
  <target-id>            Target concept identifier to link to
  [bundle]               Path to OKF bundle directory (default: 'knowledge' or '.')

Flags:
  --desc <prose>         Semantic description of the relationship (required)
  --actor <actor>        Author provenance identifier (default: 'agent/cli')
  --no-log               Skip appending to knowledge/log.md
  --json                 Emit machine-readable JSON result

Examples:
  okf relate decisions/adr-008 architecture/database knowledge --desc "implements connection pooling strategy"
`)
}

func printInitUsage() {
	fmt.Printf(`Initialize a new OKF v0.2 bundle with root index.md and log.md.

Usage:
  okf init [path]

Arguments:
  [path]                 Target directory to initialize (default: 'knowledge' or '.')

Examples:
  okf init knowledge
  okf init .
`)
}

func printBootstrapUsage() {
	fmt.Printf(`Scaffold a complete OKF Agent Memory stack in a project repository.

Usage:
  okf bootstrap [target-dir] [flags]

Arguments:
  [target-dir]           Target repository root directory (default: '.')

Flags:
  --name <name>          Project name (defaults to target directory name)
  --no-skill             Skip installing .agents/skills/okf-memory/
  --no-agents-md         Skip installing canonical AGENTS.md
  --overwrite-agents-md  Overwrite existing AGENTS.md instead of smart append
  --no-makefile          Skip installing Makefile
  --no-bundle            Skip installing knowledge/ scaffold (index.md, log.md)

Examples:
  okf bootstrap .
  okf bootstrap /path/to/project --name "My Project"
`)
}

func printMCPUsage() {
	fmt.Printf(`Run OKF Agent Memory as a Model Context Protocol (MCP) server over stdio.

Usage:
  okf mcp [bundle]

Arguments:
  [bundle]               Path to OKF bundle directory (default: 'knowledge' or '.')

Examples:
  okf mcp knowledge
  okf mcp .
`)
}

func printUsage() {
	fmt.Printf(`OKF Agent Memory CLI (v%s)

Usage:
  okf <command> [arguments] [flags]

Commands:
  validate [bundle]      Validate an OKF bundle for conformance and graph health
  search <query> [bundle] Search concepts using in-memory BM25 scoring
  show <concept-id>      Display full concept details, frontmatter, and links
  create <id> [bundle]   Create a new concept with automated bookkeeping
  update <id> [bundle]   Update an existing concept
  relate <src> <tgt>     Connect two concepts with a relative link and context
  init [path]            Initialize a new OKF v0.2 bundle (index.md, log.md)
  bootstrap [target-dir] Scaffold complete memory stack (skill, AGENTS.md, knowledge, Makefile)
  agents <subcommand>    Manage AGENTS.md, lint AAG rules, and maintain SSoT tool symlinks
  mcp [bundle]           Run as a Model Context Protocol (MCP) server over stdio
  version                Print version information
  help                   Show this help message

Flags:
  --for-path <path>      Filter concepts governing a file path via code_refs (search)
  --json                 Emit machine-readable JSON output
  --strict               Gate connectivity warnings and trust gaps as errors in validate
  --drift                Check descriptions and code_refs for drift in validate
  --stale                Gate expired review dates (stale_after) as errors in validate

`, Version)
}

func defaultBundle(args []string) (string, []string) {
	// Look for ./knowledge or default to current directory.
	fallback := "."
	if info, err := os.Stat("knowledge"); err == nil && info.IsDir() {
		fallback = "knowledge"
	}
	return splitOptionalPath(args, fallback)
}

// splitOptionalPath consumes an optional positional path only when it is the
// first argument. Everything else belongs to the command's FlagSet, including
// values for flags such as "--limit 3" and "--type Fact".
func splitOptionalPath(args []string, fallback string) (string, []string) {
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		return args[0], args[1:]
	}
	return fallback, args
}

func cmdValidate(args []string) {
	if hasHelpFlag(args) {
		printValidateUsage()
		return
	}
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	strict := fs.Bool("strict", false, "Fail on broken links, orphans, and provenance gaps")
	drift := fs.Bool("drift", false, "Check for drift between index.md and concept descriptions")
	stale := fs.Bool("stale", false, "Fail if any concepts are stale (past stale_after)")
	agents := fs.Bool("agents", false, "Validate AGENTS.md against AAG rules and SSoT tool symlinks")
	jsonOut := fs.Bool("json", false, "Output results as JSON")

	bundleDir, flagArgs := defaultBundle(args)
	_ = fs.Parse(flagArgs)
	if len(fs.Args()) > 0 {
		bundleDir = fs.Args()[0]
	}

	if *agents {
		agentsRoot := filepath.Clean(bundleDir)
		// #nosec G703 -- agentsRoot is sanitized and checked for existence of AGENTS.md
		if info, err := os.Stat(filepath.Join(agentsRoot, "AGENTS.md")); err != nil || !info.Mode().IsRegular() {
			parent := filepath.Dir(agentsRoot)
			// #nosec G703 -- parent is derived from sanitized path
			if pInfo, pErr := os.Stat(filepath.Join(parent, "AGENTS.md")); pErr == nil && pInfo.Mode().IsRegular() {
				agentsRoot = parent
			}
		}
		if err := runAgentsCheck([]string{"--root", agentsRoot, fmt.Sprintf("--strict=%t", *strict)}); err != nil {
			fmt.Fprintf(os.Stderr, "\nAgents governance check failed: %v\n", err)
			os.Exit(1)
		}
	}

	b, err := okf.LoadBundle(bundleDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading bundle: %v\n", err)
		os.Exit(2)
	}

	res := okf.Validate(b, okf.ValidateOptions{Strict: *strict, Drift: *drift, Stale: *stale})

	if *jsonOut {
		data, _ := json.MarshalIndent(res, "", "  ")
		fmt.Println(string(data))
		if !res.GatePassed {
			os.Exit(1)
		}
		return
	}

	for _, w := range res.Warnings {
		fmt.Printf("warn  %s\n", w)
	}
	for _, g := range res.GateFindings {
		prefix := "warn "
		if *strict {
			prefix = "gate "
		}
		fmt.Printf("%s %s\n", prefix, g)
	}
	for _, bl := range res.BrokenLinks {
		prefix := "warn "
		if *strict {
			prefix = "gate "
		}
		fmt.Printf("%s %s: broken concept link -> %s (%s)\n", prefix, bl.SourceConcept, bl.TargetHref, bl.Reason)
	}
	for _, o := range res.Orphans {
		prefix := "warn "
		if *strict {
			prefix = "gate "
		}
		fmt.Printf("%s %s.md: orphan (no concept links in or out)\n", prefix, o)
	}
	for _, e := range res.Errors {
		fmt.Printf("error %s\n", e)
	}

	verStr := "no declared version"
	if res.DeclaredVer != "" {
		verStr = "v" + res.DeclaredVer
	}

	flagSummary := ""
	if *strict {
		flagSummary += " [--strict]"
	}
	if *stale {
		flagSummary += " [--stale]"
	}

	fmt.Printf("\nOKF v0.2 check of \"%s\" (%s): %d concept(s), %d error(s), %d warning(s); %d broken link(s), %d orphan(s), %d stale%s. ",
		bundleDir, verStr, res.ConceptCount, len(res.Errors), len(res.Warnings)+len(res.GateFindings), len(res.BrokenLinks), len(res.Orphans), res.StaleCount, flagSummary)

	if !res.IsConformant {
		fmt.Println("NOT conformant.")
		os.Exit(1)
	} else if !res.GatePassed {
		fmt.Println("Conformant, but the producer gate failed.")
		os.Exit(1)
	} else {
		fmt.Println("Conformant.")
	}
}

func cmdSearch(args []string) {
	if hasHelpFlag(args) {
		printSearchUsage()
		return
	}
	fs := flag.NewFlagSet("search", flag.ExitOnError)
	limit := fs.Int("limit", 10, "Maximum number of search results")
	forPath := fs.String("for-path", "", "Filter concepts governing a specific file path via code_refs")
	jsonOut := fs.Bool("json", false, "Output results as JSON")

	// Separate flags from positional arguments
	var flagArgs []string
	var positional []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flagArgs = append(flagArgs, arg)
			// If flag takes a value separated by space, consume it
			name := strings.TrimLeft(arg, "-")
			if (name == "limit" || name == "for-path") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				flagArgs = append(flagArgs, args[i])
			}
		} else {
			positional = append(positional, arg)
		}
	}

	_ = fs.Parse(flagArgs)

	fallback := "."
	if info, err := os.Stat("knowledge"); err == nil && info.IsDir() {
		fallback = "knowledge"
	}
	bundleDir := fallback

	var query string
	if *forPath != "" {
		if len(positional) == 1 {
			cleanCandidate := filepath.Clean(positional[0])
			// #nosec G703 -- CLI argument used for bundle directory existence check
			if info, err := os.Stat(cleanCandidate); err == nil && info.IsDir() {
				bundleDir = cleanCandidate
			} else {
				query = positional[0]
			}
		} else if len(positional) >= 2 {
			query = positional[0]
			bundleDir = positional[1]
		}
	} else {
		if len(positional) >= 1 {
			query = positional[0]
		}
		if len(positional) >= 2 {
			bundleDir = positional[1]
		}
	}

	if query == "" && *forPath == "" {
		fmt.Fprintln(os.Stderr, "Usage: okf search <query> [bundle] [--for-path <path>] [--limit N] [--json]")
		os.Exit(1)
	}

	b, err := okf.LoadBundle(bundleDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading bundle: %v\n", err)
		os.Exit(2)
	}

	var results []okf.SearchResult
	if *forPath != "" {
		results = b.SearchForPath(*forPath, query, *limit)
	} else {
		results = b.Search(query, *limit)
	}

	if *jsonOut {
		data, _ := json.MarshalIndent(results, "", "  ")
		fmt.Println(string(data))
		return
	}

	if len(results) == 0 {
		if *forPath != "" && query != "" {
			fmt.Printf("No matching concepts found for path '%s' and query: '%s'\n", *forPath, query)
		} else if *forPath != "" {
			fmt.Printf("No matching concepts found governing path: '%s'\n", *forPath)
		} else {
			fmt.Printf("No matching concepts found for query: '%s'\n", query)
		}
		return
	}

	if *forPath != "" {
		fmt.Printf("Found %d matching concept(s) governing '%s' in '%s':\n\n", len(results), *forPath, bundleDir)
	} else {
		fmt.Printf("Found %d matching concept(s) in '%s':\n\n", len(results), bundleDir)
	}

	for i, r := range results {
		govBadge := fmt.Sprintf("[%s]", r.Governance)
		fmt.Printf("%2d. %-12s [%.2f] %s (%s)\n    %s\n    Matches: %s\n\n",
			i+1, govBadge, r.Score, r.ConceptID, r.Type, r.Description, strings.Join(r.MatchedOn, ", "))
	}
}

func cmdShow(args []string) {
	if len(args) == 0 || hasHelpFlag(args) {
		printShowUsage()
		if len(args) == 0 {
			os.Exit(1)
		}
		return
	}

	rawID := strings.TrimSpace(args[0])
	if err := okf.ValidateConceptID(rawID); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	conceptID := strings.TrimSuffix(rawID, ".md")
	var subArgs []string
	if len(args) > 1 {
		subArgs = args[1:]
	}

	fs := flag.NewFlagSet("show", flag.ExitOnError)
	jsonOut := fs.Bool("json", false, "Output concept as JSON")
	rawOut := fs.Bool("raw", false, "Output raw markdown file content")

	bundleDir, flagArgs := defaultBundle(subArgs)
	_ = fs.Parse(flagArgs)

	b, err := okf.LoadBundle(bundleDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading bundle: %v\n", err)
		os.Exit(2)
	}

	c, ok := b.Concepts[conceptID]
	if !ok {
		fmt.Fprintf(os.Stderr, "Concept '%s' not found in '%s'\n", conceptID, bundleDir)
		os.Exit(1)
	}

	if *rawOut {
		fmt.Print(c.RawContent)
		return
	}

	if *jsonOut {
		data, _ := json.MarshalIndent(c, "", "  ")
		fmt.Println(string(data))
		return
	}

	fmt.Printf("ID:          %s\n", c.ID)
	fmt.Printf("Type:        %s\n", c.Type)
	fmt.Printf("Title:       %s\n", c.Title)
	fmt.Printf("Description: %s\n", c.Description)
	if len(c.Tags) > 0 {
		fmt.Printf("Tags:        %s\n", strings.Join(c.Tags, ", "))
	}
	if c.Generated != nil {
		fmt.Printf("Generated:   %s by %s\n", c.Generated.At, c.Generated.By)
	}
	if len(b.InboundGraph[c.ID]) > 0 {
		fmt.Printf("Inbound:     %s\n", strings.Join(b.InboundGraph[c.ID], ", "))
	}
	if len(b.Graph[c.ID]) > 0 {
		fmt.Printf("Outbound:    %s\n", strings.Join(b.Graph[c.ID], ", "))
	}
	fmt.Println("\n--- Body ---")
	fmt.Println(strings.TrimSpace(c.Body))
}

func cmdCreate(args []string) {
	if len(args) == 0 || hasHelpFlag(args) {
		printCreateUsage()
		if len(args) == 0 {
			os.Exit(1)
		}
		return
	}

	rawID := strings.TrimSpace(args[0])
	if err := okf.ValidateConceptID(rawID); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	conceptID := strings.TrimSuffix(rawID, ".md")
	var subArgs []string
	if len(args) > 1 {
		subArgs = args[1:]
	}

	fs := flag.NewFlagSet("create", flag.ExitOnError)
	cType := fs.String("type", "Fact", "Concept type (required)")
	title := fs.String("title", "", "Concept title")
	desc := fs.String("desc", "", "Concept description (one sentence)")
	body := fs.String("body", "", "Concept body content")
	tagsStr := fs.String("tags", "", "Comma-separated tags")
	actor := fs.String("actor", "agent/cli", "Author actor string")
	noLog := fs.Bool("no-log", false, "Skip appending to log.md")
	noIndex := fs.Bool("no-index", false, "Skip updating parent index.md")
	jsonOut := fs.Bool("json", false, "Emit JSON result")

	bundleDir, flagArgs := defaultBundle(subArgs)
	_ = fs.Parse(flagArgs)

	titleVal := *title
	if strings.TrimSpace(titleVal) == "" {
		titleVal = filepath.Base(conceptID)
	}

	var tags []string
	if *tagsStr != "" {
		for _, t := range strings.Split(*tagsStr, ",") {
			trimmed := strings.TrimSpace(t)
			if trimmed != "" {
				tags = append(tags, trimmed)
			}
		}
	}

	c := &okf.Concept{
		ID:          conceptID,
		Path:        conceptID + ".md",
		Type:        *cType,
		Title:       titleVal,
		Description: *desc,
		Tags:        tags,
		Body:        *body,
	}

	err := okf.SaveConcept(bundleDir, c, true, !*noLog, !*noIndex, *actor)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if *jsonOut {
		data, _ := json.Marshal(map[string]string{
			"status":     "success",
			"concept_id": conceptID,
			"path":       c.Path,
		})
		fmt.Println(string(data))
	} else {
		fmt.Printf("Created concept '%s' (%s) in '%s'\n", c.Path, c.Title, bundleDir)
	}
}

func cmdUpdate(args []string) {
	if len(args) == 0 || hasHelpFlag(args) {
		printUpdateUsage()
		if len(args) == 0 {
			os.Exit(1)
		}
		return
	}

	rawID := strings.TrimSpace(args[0])
	if err := okf.ValidateConceptID(rawID); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	conceptID := strings.TrimSuffix(rawID, ".md")
	var subArgs []string
	if len(args) > 1 {
		subArgs = args[1:]
	}

	fs := flag.NewFlagSet("update", flag.ExitOnError)
	title := fs.String("title", "", "Updated concept title")
	desc := fs.String("desc", "", "Updated description")
	body := fs.String("body", "", "Updated body content")
	actor := fs.String("actor", "agent/cli", "Author actor string")
	noLog := fs.Bool("no-log", false, "Skip appending to log.md")
	noIndex := fs.Bool("no-index", false, "Skip updating parent index.md")
	jsonOut := fs.Bool("json", false, "Emit JSON result")

	bundleDir, flagArgs := defaultBundle(subArgs)
	_ = fs.Parse(flagArgs)

	b, err := okf.LoadBundle(bundleDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading bundle: %v\n", err)
		os.Exit(2)
	}

	c, ok := b.Concepts[conceptID]
	if !ok {
		fmt.Fprintf(os.Stderr, "Concept '%s' not found\n", conceptID)
		os.Exit(1)
	}

	isPassed := func(name string) bool {
		found := false
		fs.Visit(func(f *flag.Flag) {
			if f.Name == name {
				found = true
			}
		})
		return found
	}

	if isPassed("title") {
		c.Title = *title
	}
	if isPassed("desc") {
		c.Description = *desc
	}
	if isPassed("body") {
		c.Body = *body
	}

	err = okf.SaveConcept(bundleDir, c, false, !*noLog, !*noIndex, *actor)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if *jsonOut {
		data, _ := json.Marshal(map[string]string{
			"status":     "success",
			"concept_id": conceptID,
			"path":       c.Path,
		})
		fmt.Println(string(data))
	} else {
		fmt.Printf("Updated concept '%s' in '%s'\n", c.Path, bundleDir)
	}
}

func cmdRelate(args []string) {
	if len(args) < 2 || hasHelpFlag(args) {
		printRelateUsage()
		if len(args) < 2 {
			os.Exit(1)
		}
		return
	}

	sourceID := args[0]
	targetID := args[1]

	var subArgs []string
	if len(args) > 2 {
		subArgs = args[2:]
	}

	fs := flag.NewFlagSet("relate", flag.ExitOnError)
	desc := fs.String("desc", "", "Description explaining the relationship")
	actor := fs.String("actor", "agent/cli", "Author actor string")
	jsonOut := fs.Bool("json", false, "Emit JSON result")

	bundleDir, flagArgs := defaultBundle(subArgs)
	_ = fs.Parse(flagArgs)

	err := okf.RelateConcepts(bundleDir, sourceID, targetID, *desc, *actor)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if *jsonOut {
		data, _ := json.Marshal(map[string]string{
			"status": "success",
			"source": strings.TrimSpace(strings.TrimSuffix(sourceID, ".md")),
			"target": strings.TrimSpace(strings.TrimSuffix(targetID, ".md")),
		})
		fmt.Println(string(data))
	} else {
		fmt.Printf("Linked '%s' -> '%s' in '%s'\n", strings.TrimSpace(strings.TrimSuffix(sourceID, ".md")), strings.TrimSpace(strings.TrimSuffix(targetID, ".md")), bundleDir)
	}
}

func cmdInit(args []string) {
	if hasHelpFlag(args) {
		printInitUsage()
		return
	}
	bundleDir, _ := defaultBundle(args)
	if err := okf.InitBundle(bundleDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing bundle: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Initialized OKF v0.2 bundle in '%s'\n", bundleDir)
}

func cmdBootstrap(args []string) {
	if hasHelpFlag(args) {
		printBootstrapUsage()
		return
	}
	targetDir, subArgs := splitOptionalPath(args, ".")

	fs := flag.NewFlagSet("bootstrap", flag.ExitOnError)
	name := fs.String("name", "", "Project name (defaults to target directory name)")
	noSkill := fs.Bool("no-skill", false, "Skip installing .agents/skills/okf-memory")
	noAgentsMD := fs.Bool("no-agents-md", false, "Skip installing AGENTS.md")
	overwriteAgentsMD := fs.Bool("overwrite-agents-md", false, "Overwrite existing AGENTS.md instead of smart append")
	noMakefile := fs.Bool("no-makefile", false, "Skip installing Makefile")
	noBundle := fs.Bool("no-bundle", false, "Skip installing knowledge/ scaffold")
	_ = fs.Parse(subArgs)

	opts := okf.BootstrapOptions{
		ProjectName:       *name,
		InstallSkill:      !*noSkill,
		InstallAgentsMD:   !*noAgentsMD,
		OverwriteAgentsMD: *overwriteAgentsMD,
		InstallMakefile:   !*noMakefile,
		InstallBundle:     !*noBundle,
	}

	if err := okf.Bootstrap(targetDir, opts); err != nil {
		fmt.Fprintf(os.Stderr, "Bootstrap error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully bootstrapped OKF Agent Memory in '%s'!\n", targetDir)
	fmt.Println("Created:")
	if opts.InstallBundle {
		fmt.Println("  - knowledge/ (index.md, log.md)")
	}
	if opts.InstallSkill {
		fmt.Println("  - .agents/skills/okf-memory/ (SKILL.md, discovery, update, etc.)")
	}
	if opts.InstallAgentsMD {
		fmt.Println("  - AGENTS.md")
	}
	if opts.InstallMakefile {
		fmt.Println("  - Makefile")
	}
	fmt.Println("\nRun 'okf validate knowledge' or 'make validate' to verify.")
}

func cmdMCP(args []string) {
	if hasHelpFlag(args) {
		printMCPUsage()
		return
	}
	bundleDir, _ := defaultBundle(args)
	if err := RunMCPServer(bundleDir); err != nil {
		fmt.Fprintf(os.Stderr, "MCP server error: %v\n", err)
		os.Exit(1)
	}
}
