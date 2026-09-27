---
type: Status
title: 当前项目进度
description: v0.4.3-cn.1 已发布；上游 v0.4.3 与中文检索、显式初始化及双库 wrapper 完成集成，本机四套 Agent 已切换到新版。
generated: { by: agent/mcp, at: "2026-09-27T16:00:05Z" }
---

# 当前项目进度

## 2026-09-27

`v0.4.3-cn.1` 已发布到 https://github.com/passwordand/okf-agent-memory-cn/releases/tag/v0.4.3-cn.1。发布提交为 `42a1078`，GitHub CI 与 Release 工作流均成功；Windows、macOS、Linux 资产已上传。

本版合入上游稳定版 `v0.4.3` 的 vault / Hub、MCP `outputSchema` 与 `structuredContent` 及安全修复，保留 `gse` 中文分词、显式 `okf_init()`、CLI 默认 `knowledge/` 和双库 wrapper。OKF v0.2 知识结构无需迁移；根索引的历史 `--help.md` 断链已清理。

本机 Codex、OpenCode、DeepSeek Harness、WorkBuddy 均使用同一 `~/.config/agent-memory/mem-mcp.js`。公共 wrapper 已指向 `bin/okf-v0.4.3-cn.1.exe`，该程序由发布标签对应提交用 Go 1.26 本地构建；旧版程序并排保留，正在运行的旧 MCP 连接需重新连接后才使用新版。旧程序和 wrapper 已备份到 `~/.config/agent-memory/backup/v0.4.3-cn.1-20260927/`。

本地 Go 测试除两项需 Windows 符号链接权限的测试外均通过；GitHub Ubuntu CI 完整测试通过。Node wrapper 六项测试、`go vet ./...`、项目与全局库严格校验通过。新版安装 wrapper 的真实 MCP 会话已通过缺库初始化、搜索、重复初始化及全局库读取 / 校验。

# Related Concepts
- [显式初始化缺失的项目知识库](../architecture/explicit-bundle-initialization.md): 显式初始化架构决定。
- [上游 v0.4.3 与中文检索版兼容性审查](../research/upstream-v0-4-3-compatibility.md): 升级前证据。
- [上游 v0.4.3 与中文记忆功能的集成决定](../architecture/upstream-v0-4-3-integration.md): 本次集成决定。
