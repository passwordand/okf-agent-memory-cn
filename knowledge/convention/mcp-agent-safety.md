---
type: Convention
title: MCP Tool Security & Untrusted Agent Input
description: Conventions for agentic memory operations treating all tool arguments as untrusted input and enforcing boundary confinement.
generated: { by: agent/mcp, at: 2026-09-08T07:51:23Z }
---

# MCP Tool Security & Untrusted Agent Input

## Context & Threat Model

AI agents interacting with the **OKF Agent Memory** Model Context Protocol (MCP) server execute in autonomous loops. These agents may process web scrapes, untrusted pull requests, or external messages that contain adversarial prompt injections.

Consequently, tool arguments (`concept_id`, `bundle`, `title`, `description`, `type`, `body`) cannot be assumed benign:
1. **Indirect Prompt Injection**: An attacker embedding instructions in a repository file may attempt to trick an agent into calling `okf_create` or `okf_show` with arbitrary file paths (`/etc/passwd`, `~/.ssh/id_rsa`).
2. **Confused Deputy Attacks**: An agent granted filesystem access must not become an unintended file-overwrite or read proxy.

## Core Behavioral Guidelines for Memory Tools

### 1. Treat Tool Arguments as Untrusted
Every tool implementation and agent interaction must enforce:
- **Server Root Confinement**: Dynamic `bundle` selection via `resolveBundleDir` is constrained to the canonical server workspace root.
- **Path Sanitization**: Concept IDs must never contain `..` traversal sequences, leading slashes, or reference root reserved files (`index.md`, `log.md`).

### 2. Guard Metadata Against Injection
When constructing concept frontmatter:
- Avoid multi-line values in scalar metadata fields (`type`, `title`, `description`, `actor`) to prevent YAML attribute smuggling.
- Never allow unverified AI content to mark itself with `human:` provenance.

### 3. Read-Before-Write Progressive Disclosure
- Always perform scoped searches (`okf search "<query>"`) and inspect concept descriptions before retrieving entire files.
- Refuse to execute bulk recursive reads or dump operations across the filesystem.

## Relationships

- Governed by the core principles in [principles](principles.md).
- Enforced deterministically by Layer 4 boundaries in [../architecture/security-boundaries](../architecture/security-boundaries.md).
