# AGENTS.md - Agent Action Grammar (AAG v0.1)

## 0. Project & Domain Codex
- TONE: style == direct_concise
- FORMAT: diagrams => ASSERT(syntax == mermaid, quote_labels == true, ELSE=STOP("Mermaid required; quote node labels with special characters or braces; ASCII/box art prohibited."))

## 1. Behavioral Invariants (RFC 2119)
- MUST use AES-256-GCM with 96-bit (12-byte) nonce for all payload encryption.
- MUST include header `X-OKF-Encryption-Version: v2` in encrypted outputs.
- NEVER use ECB or CBC cipher modes.
- NEVER forge human verification (`verified:` is human-only; declare `generated: { by: "<actor>", at: "<iso-time>" }`).

## 2. Guard Clauses & Scope Governance
- ON edit(@internal/crypto/legacy):
    IF governance == "hold" => STOP("Subsystem frozen by governance. Confirm with user.")
- ON edit(@path/):
    IF first_visit(@path/) => okf_search(for_path=@path/)
