---
type: Decision
title: 显式初始化缺失的项目知识库
description: 项目 MCP 可在知识库缺失时连接，只有显式调用无路径参数的 okf_init 或 CLI init 才创建受边界约束的 knowledge 知识库。
generated: { by: agent/mcp, at: "2026-09-27T14:29:01Z" }
---

# 显式初始化缺失的项目知识库

## 决定

- `okf_init()` 只创建 MCP 启动时绑定的知识库 `index.md` 和 `log.md`，不接受运行时路径参数。
- MCP 连接、读取与搜索失败均不自动创建知识库；Agent 仅在用户明确要求初始化时调用该工具。
- 无参数 `okf init` 默认创建 `./knowledge/`；显式 `okf init .` 保留根目录知识库用法。
- 项目 wrapper 在没有现有知识库时使用显式 `--project-root` 或启动 cwd 作为项目根；全局库继续与项目库隔离。
- 初始化不得覆盖现有文件，须拒绝越界路径及不安全的符号链接。`okf bootstrap` 继续负责完整 Agent 配置。

## 实现与验证

Go CLI/MCP、双库 wrapper、Agent 指引及使用文档同步修改。新增 CLI、MCP 和 wrapper 回归测试，并通过严格知识库校验。

# Related Concepts
- [Go Single-Binary CLI & MCP Architecture Decision](tooling-decision.md): 显式初始化扩展现有 Go CLI 与 MCP 双接口能力。
- [Bundle Isolation and Mutation Security Boundaries](security-boundaries.md): 初始化路径继承 MCP 根目录和符号链接边界约束。
