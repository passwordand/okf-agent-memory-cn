# Dual-Memory Agent Architecture (DMAA) Benchmark Suite

> **Automated Benchmark Suite** quantifying prompt token reduction, prefill latency (Time-To-First-Token / TTFT), and instruction adherence on local Large Language Models (Gemma, Qwen, Llama 3) and cloud APIs (OpenAI, Anthropic, Google) via `okf-benchmark`.

* 📖 **Detailed Specification & Peer Methodology Guide**: [docs/guides/BENCHMARKING_METHODOLOGY.md](../docs/guides/BENCHMARKING_METHODOLOGY.md)

---

## 🎯 Benchmark Objectives

AI coding agents suffer from **Context Bloat** and **Attention Degradation (Lost-in-the-Middle)** when entire project architectures and verbose conversational rules are dumped into monolithic context files (`CLAUDE.md`, `.cursorrules`).

This benchmark suite measures the quantitative impact of the **Dual-Memory Agent Architecture (DMAA)**:

1. **Layer 1 (Push Working Memory / `-suite push`)**:  
   Compares conversational English `.cursorrules` (~650 tokens) against compact **Agent Action Grammar (AAG)** guard clauses (~210 tokens). Evaluates token reduction, TTFT, and negative constraint adherence (Mermaid syntax, provenance integrity, governance stop triggers).
2. **Layer 2 (Pull Knowledge Memory / `-suite pull`)**:  
   Compares a monolithic documentation dump (~3,000 tokens) against **OKF Progressive Disclosure** via fast in-memory BM25 retrieval (~550 tokens).
3. **Full DMAA Stack (`-suite dmaa`)**:  
   Evaluates the unified system impact of combining both layers against the industry status quo.

---

## 📊 Live Benchmark Evidence

### Apple Silicon M2 Pro (32 GB) — Empirical DMAA Benchmark (`qwen/qwen2.5-coder-14b` via LM Studio)

| Architecture Tier | Industry Monolith | OKF DMAA Stack | Savings / Acceleration |
| :--- | :--- | :--- | :--- |
| **Layer 1 (Push Working Memory)** | `687 tok` | `270 tok` | **-60.7% steering tax** (2.3x faster TTFT) |
| **Layer 2 (Pull Knowledge Memory)** | `3,055 tok` | `627 tok` | **-79.5% context overhead** (4.7x faster TTFT) |
| **Combined System Overhead** | `3,742 tok` | `897 tok` | **🔥 -76.0% context tax per turn** (3.9x faster TTFT) |

---

## 🚀 How to Reproduce Locally

The benchmark runner `okf-benchmark` is written in **100% pure Go** with zero external dependencies.

### 1. Local LLMs (LM Studio or Ollama)
```bash
# 1. Local LM Studio (Default, listening on http://localhost:1234)
go run ./cmd/okf-benchmark -suite push
go run ./cmd/okf-benchmark -suite pull
go run ./cmd/okf-benchmark -suite dmaa

# 2. Local Ollama (listening on http://localhost:11434)
go run ./cmd/okf-benchmark -p ollama -m llama3.2 -suite dmaa
```

### 3. Cloud Providers (OpenAI, Claude, Gemini)
```bash
# OpenAI (uses OPENAI_API_KEY)
export OPENAI_API_KEY="sk-..."
go run ./cmd/okf-benchmark -p openai -m gpt-4o -suite dmaa

# Anthropic Claude (uses ANTHROPIC_API_KEY)
export ANTHROPIC_API_KEY="sk-ant-..."
go run ./cmd/okf-benchmark -p claude -m claude-3-7-sonnet-20250219 -suite dmaa

# Google Gemini (uses GEMINI_API_KEY)
export GEMINI_API_KEY="AIza..."
go run ./cmd/okf-benchmark -p gemini -m gemini-2.5-flash -suite dmaa
```

### Useful Flags
```bash
# Compare the exact generated responses side-by-side:
go run ./cmd/okf-benchmark -suite dmaa -o

# Adjust timeout for deep reasoning models:
go run ./cmd/okf-benchmark -timeout 300s
```

---

## 📁 Directory Layout

```
benchmarks/
├── README.md               # This overview
├── data/
│   ├── PROSE_RULES.md      # Baseline conversational steering rules (.cursorrules)
│   ├── AAG_RULES.md        # Treatment Agent Action Grammar steering rules (AGENTS.md)
│   ├── MONOLITH_DOCS.md    # Baseline monolithic project documentation dump
│   └── knowledge/          # Treatment OKF v0.2 knowledge corpus for BM25 retrieval
└── results/                # Timestamped markdown benchmark reports
```
