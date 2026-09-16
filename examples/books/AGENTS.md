# AGENTS.md - DMAA Protocol v0.1 - Instructions for AI Agents in `Literature & Book Knowledge Base`

> Powered by [OKF Agent Memory](https://github.com/okf-memory/okf-agent-memory) -- Open Knowledge Format (OKF) v0.2 persistent project memory for AI agents.

## 0. Project & Domain Codex (Literature & Editorial Analysis)
- TONE: style == literary_analytical, objective_scholarly, nuanced
- FORMAT: diagrams => ASSERT(syntax == mermaid, ELSE=STOP("All diagrams MUST use Mermaid syntax; ASCII/text art prohibited."))
- INTEGRITY: maintain(character_consistency, canon_fidelity, timeline_accuracy)
- SPOILERS: explicit_warning => MUST flag(major_plot_twists, ending_revelations)
- CITATIONS: quotes => ASSERT(attribute_to_author, specify_edition_or_chapter)

<!-- BEGIN OKF AGENT MEMORY -->
## 1. Behavioral Invariants & Constraints (RFC 2119)
- MUST execute `okf_search(query=keywords, limit=3)` before analyzing authors, books, themes, or literary reviews.
- NEVER scan `knowledge/` via `list_dir`, `grep_search`, `find`, or raw file readers.
- NEVER forge human verification (`verified:` is human-only; declare `generated: { by: "<actor>", at: "<iso-time>" }`).
- PREFER native `okf_*` MCP tools OVER CLI fallback commands.

## 2. Guard Clauses & Scope Governance
- ON author_or_book_query(@title_or_author):
    okf_search(query=@title_or_author, limit=3) => okf_show(concept_id)
    evaluate(critical_reception, biographical_context, cross_references)
- ON user_query(authors | books | literary_reviews | motifs):
    okf_search(query=keywords, limit=3) => evaluate summary description
    IF relevant => okf_show(concept_id) ONLY on demand
    IF updating_existing_entry => PREFER okf_update OVER okf_create

## 3. Completion Pipeline (Sequential Assertion Gates)
1. IF new_book_or_author_analyzed => MUST okf_create(books/* | authors/*, type=..., title=..., desc=...)
2. IF critique_or_review_added => MUST okf_create(reviews/*, type="review", title=..., desc=...)
3. IF concepts_mutated => MUST sync(knowledge/log.md, knowledge/index.md)
4. ASSERT(okf_validate(strict=true, drift=true) == {errors: 0, warnings: 0}, ELSE=fix_before_exit)
<!-- END OKF AGENT MEMORY -->
