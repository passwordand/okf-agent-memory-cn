---
type: Status
title: 当前项目进度
description: v0.3.1-cn.4 已发布到 GitHub，新增显式 okf_init、CLI 默认知识库路径与双库 wrapper 缺库接入。
generated: { by: agent/mcp, at: "2026-09-27T14:54:09Z" }
---

# 当前项目进度

## 2026-09-27

`v0.3.1-cn.4` 已发布到 https://github.com/passwordand/okf-agent-memory-cn/releases/tag/v0.3.1-cn.4。版本包含显式 `okf_init()`、无参数 `okf init` 默认创建 `./knowledge/`、双库 wrapper 在非 Git 项目中接入缺失项目库，以及相应 Agent 指引、文档和回归测试。项目库与全局库继续隔离，读取和连接不会自动初始化。

发布提交为 `8a4d278`。GitHub Ubuntu CI 与 Release 工作流均通过；Release 包含 6 个平台二进制、2 个 starter pack、校验和及 Homebrew Formula。新增 CLI、MCP、wrapper 的本地回归测试、`go vet`、Skill 模板同步检查和 OKF 严格校验也通过（20 个概念，0 错误，0 警告）。Windows 本地完整 Go 测试仍受旧有 symlink 权限及 MCP 对抗用例的路径转义兼容问题影响；Ubuntu CI 已完整通过。`govulncheck` 因本机无法连接 Go 模块代理而未运行，已按项目安全清单人工检查本版路径与 symlink 边界。

# Related Concepts
- [显式初始化缺失的项目知识库](../architecture/explicit-bundle-initialization.md): 本次版本对应的架构决定。
