---
type: Research
title: 上游 v0.4.3 与中文检索版兼容性审查
description: OKF v0.2 知识库可互读，但上游 v0.4.3 不能直接替换中文检索版；显式初始化、中文分词和严格校验需要适配。
generated: { by: agent/mcp, at: "2026-09-27T15:17:47Z" }
---

# 上游 v0.4.3 与中文检索版兼容性审查

## 结论（2026-09-27）

当前 `v0.3.1-cn.4` 可以继续独立使用；上游稳定版 `v0.4.3` 不是可直接替换的二进制或无冲突合并目标。上游默认分支 `develop` 另有 4 个提交，已将 CLI 移入 `internal/cli`、MCP 移入 `pkg/okf`，集成成本更高。建议先以稳定版 `v0.4.3` 为基线规划移植，保留中文分词、显式 `okf_init` 和项目 / 全局库隔离。

## 可复核证据

- 两边仍使用 OKF v0.2。当前中文检索版 CLI 对上游 `v0.4.3` 知识库执行 `validate --strict --drift`：20 个概念，0 错误，0 警告。
- 使用上游 `v0.4.3` 及 `develop` 的 `pkg/okf` 核心代码加载本项目知识库：审查记录写入前均加载 20 个概念；写入后均加载 21 个概念、0 结构错误，但严格 gate 仍因根索引指向 `--help.md` 的一条断链未通过。此测试在临时源码副本中将模块声明降为 Go 1.25 以运行核心包；正式上游模块要求 Go 1.26。
- 对本项目库搜索“显式初始化项目”“缺失项目知识库初始化”“项目库显式初始化”，中文检索版均找到相关概念，上游 `v0.4.3` 和 `develop` 核心均返回空结果。上游 `search.go` 使用 Unicode 连续词分割，本分支使用嵌入词典的 `gse`。
- 上游 `v0.4.3` 的 MCP 工具列表仅有 6 个工具，没有 `okf_init`；`okf mcp` 会在工具调用前加载知识库，缺库时不能显式初始化。上游无参数 `okf init` 仍回退到 `.`，而本分支固定为 `knowledge`。
- 上游 MCP 改为嵌入 `schemas/tools.json`，每个工具有 object `outputSchema` 并返回 `structuredContent`；本分支当前没有输出 schema。移植 `okf_init` 时须同时补工具 schema 和结构化返回。
- 上游 `InitBundle` 使用 `atomicWriteFile`，本分支使用 `Lstat` 与排他创建防止覆盖及 symlink。整合时须同时保留原子写入和不覆盖 / 路径边界。
- 从共同基线 `v0.3.1` 到上游 `v0.4.3` 有 49 个提交。三方文件合并模拟在 `mcp.go`、`mutate.go`、`go.mod` 等关键文件出现冲突；`develop` 还涉及文件搬迁。
- 上游模块要求 Go 1.26、`golang.org/x/crypto` 与 `golang.org/x/sys`；本分支使用 Go 1.25、`gse` 与 `cedar`。若引入 vault/hub，需合并依赖并升级构建环境。

## 后续验证重点

移植时先修复严格校验断链，再按同一 MCP 连接的缺库初始化、重复调用、双库隔离、中文查询、schema / structuredContent、跨平台路径与 symlink、Go race 测试进行回归。

# Related Concepts
- [显式初始化缺失的项目知识库](../architecture/explicit-bundle-initialization.md): 上游升级必须保留显式 okf_init 与缺库可连接的项目库行为。
- [Unicode and Deterministic Search](../architecture/search-tokenization.md): 上游 Unicode 连续词分割会使本分支的中文分词检索退化，移植时需保留 gse 实现。
