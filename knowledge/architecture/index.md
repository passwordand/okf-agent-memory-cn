# Architecture

* [5-Layer System Architecture](layers.md) - Structural separation of concerns across the OKF specification, agent convention, skills, deterministic tooling, and knowledge corpus.
* [Go Single-Binary CLI & MCP Architecture Decision](tooling-decision.md) - Architectural decision to implement the deterministic OKF tooling layer as a standalone Go binary with dual CLI and MCP support.
* [Bundle Isolation and Mutation Security Boundaries](security-boundaries.md) - Defensive security architecture enforcing canonical bundle boundaries, symlink containment, path traversal prevention, and frontmatter injection defense.
* [Unicode and Deterministic Search](search-tokenization.md) - Search tokenizes Unicode letters and digits and resolves equal scores by concept ID for reproducible results.
* [Relationship Identity and Logging](relationship-identity.md) - Relationships are identified by target path and description, making retries idempotent while retaining distinct relationship contexts.
* [Safe Unknown Metadata Round-Trip](metadata-roundtrip.md) - Unknown frontmatter keys are serialized deterministically with safe quoting and JSON-compatible scalar and collection preservation.
* [CLI Optional Path Boundary](cli-argument-boundary.md) - CLI commands consume an optional bundle or target path only from the first remaining argument, preserving all subsequent flag values.
* [Governance vs. Execution Context and Code Binding](governance-model.md) - 3-tier epistemic governance model (constraint, hold, context) and code-to-knowledge binding via code_refs.
* [显式初始化缺失的项目知识库](explicit-bundle-initialization.md) - 项目 MCP 可在知识库缺失时连接，只有显式调用无路径参数的 okf_init 或 CLI init 才创建受边界约束的 knowledge 知识库。
