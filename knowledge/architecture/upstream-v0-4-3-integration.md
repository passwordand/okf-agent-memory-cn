---
type: Decision
title: 上游 v0.4.3 与中文记忆功能的集成决定
description: 以稳定版 v0.4.3 为源码基线，保留 gse 中文检索、显式 okf_init、CLI 默认 knowledge 和双库隔离，并接入上游结构化 MCP 与 Hub。
generated: { by: agent/mcp, at: "2026-09-27T15:43:21Z" }
---

# 上游 v0.4.3 集成

## 决定

- 从上游稳定版 `v0.4.3` 合入 vault、Hub、严格 MCP schema / `structuredContent` 和安全修复，继续使用 OKF v0.2 知识结构。
- 保留 `gse` 中文分词、无参数 `okf init` 创建 `./knowledge/`、无路径参数的显式 `okf_init()`、缺库可连接的项目 MCP，以及项目 / 全局库隔离的 wrapper。
- `okf_init()` 成功返回初始化状态和绑定路径；现有文件不覆盖，越界或无效知识库返回错误。普通连接与查询不自动创建文件。
- 模块升级到 Go 1.26，并同时保留 `gse` / `cedar` 与上游 `x/crypto` / `x/sys` 依赖。

## 本地验证

Go 测试在跳过两项需要 Windows 创建符号链接权限的测试后全部通过；Node wrapper 6 项测试、`go vet ./...` 和 `okf validate knowledge --strict --drift` 通过。完整 Go 测试中的两项符号链接测试因当前 Windows 权限失败。根索引中历史遗留的 `--help.md` 断链已清理。

# Related Concepts
- [显式初始化缺失的项目知识库](explicit-bundle-initialization.md): 升级后继续执行显式初始化约束
- [Unicode and Deterministic Search](search-tokenization.md): 升级后保留 gse 中文检索行为
