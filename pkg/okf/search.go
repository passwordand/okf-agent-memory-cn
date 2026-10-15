package okf

import (
	"math"
	"path"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/go-ego/gse"
)

// SearchResult represents a scored concept match.
type SearchResult struct {
	ConceptID   string   `json:"concept_id"`
	Title       string   `json:"title"`
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Governance  string   `json:"governance,omitempty"`
	CodeRefs    []string `json:"code_refs,omitempty"`
	Score       float64  `json:"score"`
	MatchedOn   []string `json:"matched_on"`
	Tags        []string `json:"tags,omitempty"`
	Inbound     []string `json:"inbound,omitempty"`
	Outbound    []string `json:"outbound,omitempty"`
	Scope       Scope    `json:"scope,omitempty"`
	Priority    int      `json:"priority,omitempty"`
	Origin      string   `json:"origin,omitempty"`
}

var (
	segmenter     gse.Segmenter
	segmenterOnce sync.Once
)

// tokenize 保留英文 / 数字检索，同时对中文使用 gse 搜索模式分词。
// 分词器只初始化一次，避免每次搜索重复加载词典。
func tokenize(s string) []string {
	segmenterOnce.Do(func() {
		// 使用 gse 编译时嵌入的简体中文词典，保证发布后的 exe 不依赖模块缓存路径。
		_ = segmenter.LoadDictEmbed()
	})

	var out []string
	for _, token := range segmenter.CutSearch(strings.ToLower(s), true) {
		token = strings.TrimSpace(token)
		if token == "" || !hasSearchRune(token) {
			continue
		}
		// 标点不进入索引；保留单字中文，支持用户查询短词。
		if hasLetterOrDigit(token) {
			out = append(out, token)
		}
	}
	return out
}

func hasSearchRune(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

func hasLetterOrDigit(s string) bool {
	return hasSearchRune(s)
}

// MaxSearchLimit defines the maximum allowable search results returned to prevent resource exhaustion.
const MaxSearchLimit = 100

// MaxQueryLength defines the maximum query string length in runes/bytes evaluated to prevent resource exhaustion.
const MaxQueryLength = 1000

// SearchOptions controls search parameters including query text, path filtering,
// YAML frontmatter metadata filters, and staleness horizons.
type SearchOptions struct {
	Query       string
	TargetPath  string
	Limit       int
	Filter      string
	StaleWithin time.Duration
	BaseTime    time.Time
}

// Search queries the bundle using in-memory BM25/TF-IDF token scoring over frontmatter and body.
func (b *Bundle) Search(query string, limit int) []SearchResult {
	results, _ := b.SearchAdvanced(SearchOptions{
		Query: query,
		Limit: limit,
	})
	return results
}

// SearchAdvanced executes an advanced query combining full-text search, code path references,
// frontmatter filter predicates, and staleness horizon checks.
func (b *Bundle) SearchAdvanced(opts SearchOptions) ([]SearchResult, error) {
	if opts.Filter != "" {
		if _, err := parseFilterClauses(opts.Filter); err != nil {
			return nil, err
		}
	}

	baseTime := opts.BaseTime
	if baseTime.IsZero() {
		baseTime = time.Now().UTC()
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = 10
	} else if limit > MaxSearchLimit {
		limit = MaxSearchLimit
	}

	matchesCriteria := func(c *Concept) bool {
		if opts.Filter != "" {
			m, _ := c.MatchesFilter(opts.Filter)
			if !m {
				return false
			}
		}
		if opts.StaleWithin > 0 {
			if !c.IsStaleWithin(baseTime, opts.StaleWithin) {
				return false
			}
		}
		return true
	}

	targetPath := strings.TrimSpace(opts.TargetPath)
	query := strings.TrimSpace(opts.Query)

	// Sub-case A: Search by targetPath via code_refs
	if targetPath != "" {
		type candidate struct {
			concept    *Concept
			exactMatch bool
			matchedRef string
		}
		var matchedCandidates []candidate

		for _, c := range b.Concepts {
			if !matchesCriteria(c) {
				continue
			}
			for _, ref := range c.CodeRefs {
				if matchCodeRef(ref, targetPath) {
					exact := strings.ReplaceAll(strings.TrimPrefix(ref, "./"), "\\", "/") == strings.ReplaceAll(strings.TrimPrefix(targetPath, "./"), "\\", "/")
					matchedCandidates = append(matchedCandidates, candidate{
						concept:    c,
						exactMatch: exact,
						matchedRef: ref,
					})
					break
				}
			}
		}

		if len(matchedCandidates) == 0 {
			return []SearchResult{}, nil
		}

		textScores := make(map[string]SearchResult)
		if query != "" {
			allSearchResults := b.Search(query, len(b.Concepts))
			for _, r := range allSearchResults {
				textScores[r.ConceptID] = r
			}
		}

		var results []SearchResult
		for _, cand := range matchedCandidates {
			c := cand.concept
			gov := c.EffectiveGovernance()

			score := float64(governanceRank(gov) * 10)
			matchedOn := []string{"code_refs"}
			if opts.Filter != "" {
				matchedOn = append(matchedOn, "filter")
			}
			if opts.StaleWithin > 0 {
				matchedOn = append(matchedOn, "stale_within")
			}

			if cand.exactMatch {
				score += 2.0
			}

			if tr, exists := textScores[c.ID]; exists {
				score += tr.Score
				matchedOn = append(matchedOn, tr.MatchedOn...)
			}

			results = append(results, SearchResult{
				ConceptID:   c.ID,
				Title:       c.Title,
				Type:        c.Type,
				Description: c.Description,
				Governance:  gov,
				CodeRefs:    c.CodeRefs,
				Score:       math.Round(score*100) / 100,
				MatchedOn:   matchedOn,
				Tags:        c.Tags,
				Inbound:     b.InboundGraph[c.ID],
				Outbound:    b.Graph[c.ID],
			})
		}

		sort.Slice(results, func(i, j int) bool {
			rankI := governanceRank(results[i].Governance)
			rankJ := governanceRank(results[j].Governance)
			if rankI != rankJ {
				return rankI > rankJ
			}
			if results[i].Score != results[j].Score {
				return results[i].Score > results[j].Score
			}
			return results[i].ConceptID < results[j].ConceptID
		})

		if len(results) > limit {
			results = results[:limit]
		}
		return results, nil
	}

	// Sub-case B: Text query with BM25
	if query != "" {
		if len(query) > MaxQueryLength {
			runes := []rune(query)
			if len(runes) > MaxQueryLength {
				query = string(runes[:MaxQueryLength])
			}
		}

		qTokens := tokenize(query)
		if len(qTokens) == 0 {
			return []SearchResult{}, nil
		}
		if len(qTokens) > 50 {
			qTokens = qTokens[:50]
		}

		N := float64(len(b.Concepts))
		if N == 0 {
			return []SearchResult{}, nil
		}

		type parsedConceptTokens struct {
			id          string
			concept     *Concept
			titleTokens []string
			tagTokens   []string
			descTokens  []string
			idTokens    []string
			bodyTokens  []string
		}

		conceptTokenList := make([]parsedConceptTokens, 0, len(b.Concepts))
		for id, c := range b.Concepts {
			conceptTokenList = append(conceptTokenList, parsedConceptTokens{
				id:          id,
				concept:     c,
				titleTokens: tokenize(c.Title),
				tagTokens:   tokenize(strings.Join(c.Tags, " ")),
				descTokens:  tokenize(c.Description),
				idTokens:    tokenize(c.ID),
				bodyTokens:  tokenize(c.Body),
			})
		}

		tokenMatches := func(token, term string) bool {
			if utf8.RuneCountInString(term) < 4 {
				return token == term
			}
			return token == term || strings.HasPrefix(token, term)
		}

		countOccurrences := func(tokens []string, term string) float64 {
			cnt := 0.0
			for _, t := range tokens {
				if tokenMatches(t, term) {
					cnt++
				}
			}
			return cnt
		}

		// Compute Document Frequency for each query term using identical tokenization and matching logic
		df := make(map[string]float64, len(qTokens))
		for _, term := range qTokens {
			if _, exists := df[term]; exists {
				continue
			}
			count := 0.0
			for i := range conceptTokenList {
				tc := &conceptTokenList[i]
				if countOccurrences(tc.titleTokens, term) > 0 ||
					countOccurrences(tc.tagTokens, term) > 0 ||
					countOccurrences(tc.descTokens, term) > 0 ||
					countOccurrences(tc.idTokens, term) > 0 ||
					countOccurrences(tc.bodyTokens, term) > 0 {
					count++
				}
			}
			df[term] = count
		}

		var results []SearchResult

		for i := range conceptTokenList {
			tc := &conceptTokenList[i]
			c := tc.concept
			if !matchesCriteria(c) {
				continue
			}

			score := 0.0
			var matchedOn []string
			hasMatch := false

			for _, term := range qTokens {
				docFreq := df[term]
				if docFreq == 0 {
					continue
				}

				idf := math.Log(1.0 + (N-docFreq+0.5)/(docFreq+0.5))

				tfTitle := countOccurrences(tc.titleTokens, term)
				tfTags := countOccurrences(tc.tagTokens, term)
				tfDesc := countOccurrences(tc.descTokens, term)
				tfID := countOccurrences(tc.idTokens, term)
				tfBody := countOccurrences(tc.bodyTokens, term)

				termScore := 0.0
				if tfTitle > 0 {
					termScore += math.Min(tfTitle, 2.0) * 4.0
					matchedOn = append(matchedOn, "title")
				}
				if tfTags > 0 {
					termScore += math.Min(tfTags, 2.0) * 3.5
					matchedOn = append(matchedOn, "tags")
				}
				if tfDesc > 0 {
					termScore += math.Min(tfDesc, 3.0) * 2.5
					matchedOn = append(matchedOn, "description")
				}
				if tfID > 0 {
					termScore += math.Min(tfID, 2.0) * 2.0
					matchedOn = append(matchedOn, "id")
				}
				if tfBody > 0 {
					termScore += math.Min(tfBody, 5.0) * 1.0
					matchedOn = append(matchedOn, "body")
				}

				if termScore > 0 {
					hasMatch = true
					score += termScore * idf
				}
			}

			if hasMatch && score > 0 {
				dedupMap := make(map[string]bool)
				var dedupMatched []string
				for _, m := range matchedOn {
					if !dedupMap[m] {
						dedupMap[m] = true
						dedupMatched = append(dedupMatched, m)
					}
				}
				if opts.Filter != "" {
					dedupMatched = append(dedupMatched, "filter")
				}
				if opts.StaleWithin > 0 {
					dedupMatched = append(dedupMatched, "stale_within")
				}

				results = append(results, SearchResult{
					ConceptID:   tc.id,
					Title:       c.Title,
					Type:        c.Type,
					Description: c.Description,
					Governance:  c.EffectiveGovernance(),
					CodeRefs:    c.CodeRefs,
					Score:       math.Round(score*100) / 100,
					MatchedOn:   dedupMatched,
					Tags:        c.Tags,
					Inbound:     b.InboundGraph[tc.id],
					Outbound:    b.Graph[tc.id],
				})
			}
		}

		sort.Slice(results, func(i, j int) bool {
			if results[i].Score == results[j].Score {
				return results[i].ConceptID < results[j].ConceptID
			}
			return results[i].Score > results[j].Score
		})

		if len(results) > limit {
			results = results[:limit]
		}
		return results, nil
	}

	// Sub-case C: Pure Filter / Staleness search (query == "" && targetPath == "")
	var results []SearchResult
	for id, c := range b.Concepts {
		if !matchesCriteria(c) {
			continue
		}
		gov := c.EffectiveGovernance()
		score := float64(governanceRank(gov) * 10)
		var matchedOn []string
		if opts.Filter != "" {
			matchedOn = append(matchedOn, "filter")
		}
		if opts.StaleWithin > 0 {
			matchedOn = append(matchedOn, "stale_within")
		}

		results = append(results, SearchResult{
			ConceptID:   id,
			Title:       c.Title,
			Type:        c.Type,
			Description: c.Description,
			Governance:  gov,
			CodeRefs:    c.CodeRefs,
			Score:       score,
			MatchedOn:   matchedOn,
			Tags:        c.Tags,
			Inbound:     b.InboundGraph[id],
			Outbound:    b.Graph[id],
		})
	}

	sort.Slice(results, func(i, j int) bool {
		rankI := governanceRank(results[i].Governance)
		rankJ := governanceRank(results[j].Governance)
		if rankI != rankJ {
			return rankI > rankJ
		}
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		return results[i].ConceptID < results[j].ConceptID
	})

	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// matchCodeRef tests whether target matches a code reference pattern.
// Supports exact paths, directory prefixes, standard globs (path.Match), and recursive ** wildcards.
// It also seamlessly handles absolute paths (e.g. /workspace/pkg/okf/types.go).
func matchCodeRef(ref, target string) bool {
	ref = strings.ReplaceAll(strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(ref), "."), "/"), "\\", "/")
	target = strings.ReplaceAll(strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(target), "."), "/"), "\\", "/")
	if ref == "" || target == "" {
		return false
	}
	if ref == target || strings.HasSuffix(target, "/"+ref) {
		return true
	}
	// Directory prefix: e.g. "pkg/okf" matches "pkg/okf/types.go" or "/app/pkg/okf/types.go"
	cleanRef := strings.TrimSuffix(ref, "/")
	if strings.HasPrefix(target, cleanRef+"/") || strings.Contains(target, "/"+cleanRef+"/") {
		return true
	}
	// Standard path.Match glob
	if matched, err := path.Match(ref, target); err == nil && matched {
		return true
	}
	// Recursive glob: e.g. "pkg/**/*.go" or "**/*.go"
	if strings.Contains(ref, "**") {
		parts := strings.Split(ref, "**")
		if len(parts) == 2 {
			prefix := strings.TrimSuffix(parts[0], "/")
			suffix := strings.TrimPrefix(parts[1], "/")
			hasPrefixMatch := prefix == "" || strings.HasPrefix(target, prefix+"/") || target == prefix || strings.Contains(target, "/"+prefix+"/")
			if !hasPrefixMatch {
				return false
			}
			if suffix == "" {
				return true
			}
			if strings.Contains(suffix, "*") {
				matched, err := path.Match(suffix, path.Base(target))
				return err == nil && matched
			}
			return strings.HasSuffix(target, "/"+suffix) || target == suffix
		}
	}
	return false
}

func governanceRank(gov string) int {
	switch strings.ToLower(gov) {
	case GovernanceHold:
		return 3
	case GovernanceConstraint:
		return 2
	default:
		return 1
	}
}

// SearchForPath finds concepts governing targetPath via code_refs.
// Matches are ranked first by governance authority (hold > constraint > context),
// then by query relevance score (or match specificity if query is empty), and finally by ConceptID.
func (b *Bundle) SearchForPath(targetPath, query string, limit int) []SearchResult {
	results, _ := b.SearchAdvanced(SearchOptions{
		TargetPath: targetPath,
		Query:      query,
		Limit:      limit,
	})
	return results
}
