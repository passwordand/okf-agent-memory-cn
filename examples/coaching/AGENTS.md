# AGENTS.md - DMAA Protocol v0.1 - Instructions for AI Agents in `Executive Coaching Reference`

> Powered by [OKF Agent Memory](https://github.com/okf-memory/okf-agent-memory) -- Open Knowledge Format (OKF) v0.2 persistent project memory for AI agents.

## 0. Project & Domain Codex (Executive & Life Coaching)
- TONE: style == empathetic_reflective, solution_focused, non_prescriptive
- FORMAT: diagrams => ASSERT(syntax == mermaid, ELSE=STOP("All diagrams MUST use Mermaid syntax; ASCII/text art prohibited."))
- ETHICS: adhere(icf_code_of_ethics, strict_client_confidentiality)
- PRIVACY: pii => NEVER store_real_names (use pseudonymous IDs like `client-c42`)
- METHOD: ask(powerful_open_questions), mirror(client_language), avoid(premature_advice)

<!-- BEGIN OKF AGENT MEMORY -->
## 1. Behavioral Invariants & Constraints (RFC 2119)
- MUST execute `okf_search(query=keywords, limit=3)` before reviewing client history, goals, or proposing interventions.
- NEVER scan `knowledge/` via `list_dir`, `grep_search`, `find`, or raw file readers.
- NEVER forge human verification (`verified:` is human-only; declare `generated: { by: "<actor>", at: "<iso-time>" }`).
- PREFER native `okf_*` MCP tools OVER CLI fallback commands.

## 2. Guard Clauses & Scope Governance
- ON session_prep(@client_id):
    okf_search(query=@client_id, limit=3) => okf_show(concept_id)
    evaluate(active_goals, previous_breakthroughs, homework_commitments)
- ON user_query(client_history | coaching_goals | session_notes | methodologies):
    okf_search(query=keywords, limit=3) => evaluate summary description
    IF relevant => okf_show(concept_id) ONLY on demand
    IF updating_client_or_goal => PREFER okf_update OVER okf_create

## 3. Completion Pipeline (Sequential Assertion Gates)
1. IF new_session_conducted => MUST okf_create(sessions/*, type="session", title=..., desc=...)
2. IF goal_status_changed => MUST okf_update(goals/*)
3. IF concepts_mutated => MUST sync(knowledge/log.md, knowledge/index.md)
4. ASSERT(okf_validate(strict=true, drift=true) == {errors: 0, warnings: 0}, ELSE=fix_before_exit)
<!-- END OKF AGENT MEMORY -->
