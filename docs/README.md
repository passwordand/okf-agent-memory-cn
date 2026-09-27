# OKF Agent Memory — Documentation Index

Welcome to the comprehensive documentation for **OKF Agent Memory** (Open Knowledge Format v0.2 persistent project memory for AI agents).

---

## 🧭 Navigation & Overview

```
docs/
├── guides/        # Getting started, CLI & MCP reference, agent prompting best practices
├── spec/          # Formal specifications, OKF v0.2 compatibility & architecture RFCs
├── security/      # Security governance, threat modeling & adversarial audit checklists
├── project/       # Project roadmap, release playbook, multi-agent testing & alternatives
└── releases/      # Versioned changelogs & release archive
```

---

## 📚 Documentation Map

### 1. [Guides & References](guides/)
* **[Getting Started Guide](guides/GETTING_STARTED.md)** — Comprehensive onboarding guide for agents and humans.
* **[CLI & MCP Reference](guides/CLI.md)** — Complete command-line arguments, deterministic exit codes, and MCP tool protocols.
* **[Agent Instruction Best Practices](guides/AGENT_INSTRUCTION_BEST_PRACTICES.md)** — Guide to deterministic, token-efficient instruction design and Agent Action Grammar (AAG).
* **[LLM Instruction Patterns Cheat Sheet](guides/LLM_INSTRUCTION_PATTERNS_CHEATSHEET.md)** — Quick reference for RFC 2119 modals, guard clauses, tables, and anti-patterns.

### 2. [Specifications & RFCs](spec/)
* **[OKF Agent Memory Convention v0.1](spec/CONVENTION.md)** — Behavioral rules, trust tiers, and lifecycle specification.
* **[OKF v0.2 Compatibility Matrix](spec/OKF-COMPATIBILITY.md)** — Strict Open Knowledge Format specification validation analysis.
* **[Dual-Memory Agent Architecture RFC](spec/DUAL_MEMORY_AGENT_ARCHITECTURE_RFC.md)** — RFC proposing a 2-layer model: Normative Working Memory (Push) and Semantic Knowledge Memory (Pull).

### 3. [Security & Governance](security/)
* **[Security & Privacy Guidelines](security/SECURITY.md)** — Data governance, zero secret leaks, PII protection, and memory poisoning defenses.
* **[Security Audit Checklist](security/SECURITY_AUDIT.md)** — Standardized 4-area security checklist and prompt for dedicated Security Reviewer Agents.

### 4. [Project & Operations](project/)
* **[Project Roadmap & Milestones](project/ROADMAP.md)** — Phased development plan and future milestones.
* **[Release Playbook](project/playbooks/RELEASE_PLAYBOOK.md)** — Canonical release procedure, master file inventory, checklist, and notes template.
* **[Multi-Agent Testing & Evaluation](project/AGENT_TESTING.md)** — Test scenarios (TC-01 to TC-07), compatibility matrix, and benchmarks.
* **[Alternatives & Ecosystem Comparison](project/ALTERNATIVES.md)** — Comparison with Mem0, Letta, and ad-hoc markdown files.

### 5. [Release Notes & History](releases/)
* **[Release Notes Archive](releases/README.md)** — Versioned changelogs and release notes archive from v0.1.0 to present.
