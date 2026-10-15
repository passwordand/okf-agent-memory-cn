---
type: Status
title: 当前项目进度
description: v0.6.1-cn.1 正式适配完成，Windows 核心兼容测试通过，正在发布并准备本机替换。
generated: { by: agent/cli, at: "2026-10-15T03:18:49Z" }
---

# 当前项目进度

## 2026-10-15

v0.6.1-cn.1 正式适配完成，进入发布及本机安装阶段。用户明确 Windows 核心使用为验收范围，沿用隔离测试结果，仅对正式构建和安装入口做必要验证。GitHub 发布和本机切换完成后再更新部署结果。


# 2026-10-14 隔离升级测试

现用版本仍为 v0.4.3-cn.1。独立工作树 codex/okf-v061-isolation 已构建 v0.6.1-cn.1-isolation，并完成中文、显式初始化、双库和 junction 边界测试；当前没有切换、提交或发布。

Go 测试 311 项通过，17 项条件跳过，2 项权限受限测试显式排除；wrapper 6 项、真实 MCP 集成 13 项、go vet 和严格数据校验通过。完整符号链接检查、race 与真实远端联调尚未完成，详细结论见 docs/project/UPGRADE_V061_ISOLATION_REPORT.md。

以下保留现用版本发布记录：

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
- [上游 v0.6.1 隔离升级与双库兼容方案](../architecture/upstream-v0-6-1-isolation.md): 本次隔离升级方案与测试边界
