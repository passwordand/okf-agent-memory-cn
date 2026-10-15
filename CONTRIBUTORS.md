# Contributors

Thank you to everyone who has contributed to **OKF Agent Memory**! Every contribution—whether code, bug reports, architectural ideas, security disclosures, or documentation improvements—helps build resilient persistent memory for autonomous AI agents.

---

## 🌟 Code Contributors

<a href="https://github.com/okf-memory/okf-agent-memory/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=okf-memory/okf-agent-memory" alt="OKF Agent Memory Contributors" />
</a>

* **[@sknr](https://github.com/sknr)** — Project Creator & Core Maintainer
* **Denis Samatov ([@denis-samatov](https://github.com/denis-samatov))** — CLI argument handling, Windows link resolution, and actor family expansions (#14, #15, #16, #17)
* **Wu Shuwen ([@dajiaohuang](https://github.com/dajiaohuang))** — GitFlow integration branch setup and issue-first development policy (#19)
* **[@yakimoto](https://github.com/yakimoto)** — MCP tool outputSchema specifications and test coverage (#20, #22)
* **[@akushonkamen](https://github.com/akushonkamen)** — Persist sync engine state across CLI invocations (#51, #53)

---

## 💡 Architecture, Security & Community Insights

* **[@krakozavr](https://github.com/krakozavr)** — Creator of [MemContinuum](https://github.com/krakozavr/MemContinuum). Inspired the 3-tier epistemic governance model (`constraint`, `hold`, `context`) and code-to-knowledge binding (`code_refs`).
* **Martin ([@djmaze](https://github.com/djmaze))** — Responsible security disclosure and patch review for path traversal boundary containment in `SaveConcept`.
* **Ashish ([@ashishtele](https://github.com/ashishtele))** — Investigation and report on Windows relative link path resolution (#6).
* **Rogelio ([@rogeliodh](https://github.com/rogeliodh))** — Spec analysis for open actor families (#5), dot-directory bundle root scanning (#4), diagnosis of MCP outputSchema and bundle root resolution in OpenCode (#30, #31), and reproduction and root-cause analysis for CLI and MCP metadata mutation parity (#38).
* **Majid ([@majido](https://github.com/majido))** — Diagnosis of strict MCP outputSchema and missing structuredContent payload in Pi Agent (#30).
* **Matt ([@mattgdrums-cloud](https://github.com/mattgdrums-cloud))** — Detailed bug report, root-cause diagnosis, and taxonomy reconciliation proposal for CLI validate warning and gate counts (#35).
* **Alvise ([@alvistar](https://github.com/alvistar))** — Bug report and reproduction for drift validation false positives with bundle-absolute and dot-relative index links (#41).
* **Khouloud Kessentini ([@Khouloud-Kessentini](https://github.com/Khouloud-Kessentini))** — Analysis and bug report on DF/TF asymmetry, acronym suppression, and prefix false positives in BM25 scoring (#42).
* **Ritsu Kuroda ([@kurodaritsu](https://github.com/kurodaritsu))** — Architectural proposal and requirement analysis for multi-scope bundle memory layering and composite BM25 search ranking (#11).
* **Cordero Core ([@cdcore09](https://github.com/cdcore09))** — Bug report, reproduction, and root-cause analysis for lost sync state and index collisions in hub sync (#51, #52)

---

## 🤝 Becoming a Contributor

We welcome contributions of all types! Check out our [Contribution Guidelines](CONTRIBUTING.md) to get started.
