---
type: Status
title: 当前项目进度
description: 上游稳定版 v0.4.3 已在本地集成，保留中文 gse、显式 okf_init、CLI 默认 knowledge 和双库 wrapper；验证通过，待决定发布。
generated: { by: agent/mcp, at: "2026-09-27T15:43:33Z" }
---

# 当前项目进度

## 2026-09-27

`v0.3.1-cn.4` 已发布到 https://github.com/passwordand/okf-agent-memory-cn/releases/tag/v0.3.1-cn.4，发布提交为 `8a4d278`。

本地分支 `codex/upgrade-upstream-v0.4.3` 已在升级前建立检查点提交 `8f30f0c`。现已将上游稳定版 `v0.4.3` 的源码、vault / Hub、MCP `outputSchema` 与 `structuredContent` 及安全修复合入，同时保留 `gse` 中文分词、显式 `okf_init()`、CLI 默认 `knowledge/` 和双库 wrapper。清理了根索引遗留的 `--help.md` 断链。

Go 1.26 下除两项需要 Windows 符号链接权限的测试外，全部 Go 测试通过；Node wrapper 6 项通过，`go vet ./...` 与 `okf validate knowledge --strict --drift` 通过。当前升级尚未提交、推送或发布；后续如发布，需确定本分支新版本号并运行发布流程。

# Related Concepts
- [显式初始化缺失的项目知识库](../architecture/explicit-bundle-initialization.md): 当前发行版的显式初始化架构决定。
- [上游 v0.4.3 与中文检索版兼容性审查](../research/upstream-v0-4-3-compatibility.md): 升级前的兼容性证据。
- [上游 v0.4.3 与中文记忆功能的集成决定](../architecture/upstream-v0-4-3-integration.md): 本地升级实施决定。
