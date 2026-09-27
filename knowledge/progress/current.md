---
type: Status
title: 当前项目进度
description: v0.3.1-cn.4 已发布；上游 v0.4.3 兼容性审查发现不能直接替换，迁移需保留中文分词与 okf_init。
generated: { by: agent/mcp, at: "2026-09-27T15:16:26Z" }
---

# 当前项目进度

## 2026-09-27

`v0.3.1-cn.4` 已发布到 https://github.com/passwordand/okf-agent-memory-cn/releases/tag/v0.3.1-cn.4，发布提交为 `8a4d278`。GitHub Ubuntu CI 与 Release 工作流通过，10 个下载资产已上传。该版包含显式 `okf_init()`、CLI 默认 `./knowledge/`、双库 wrapper 缺库接入以及相应测试与文档。

已审查上游稳定版 `v0.4.3`（相对共同基线 `v0.3.1` 新增 49 个提交）和默认分支 `develop`（比稳定版多 4 个提交）。结论：OKF v0.2 知识库可互相加载，但不能直接换用上游二进制或无冲突合并。上游没有 `okf_init` 和中文 `gse` 分词；其严格校验对本项目根索引的 `--help.md` 给出一条断链，gate 未通过。上游要求 Go 1.26，并改变 MCP 的 schema / structuredContent；`develop` 还搬迁 CLI 和 MCP 包。详细证据见兼容性审查概念。下一步如决定跟进上游，宜从稳定版 `v0.4.3` 移植并逐项回归。

# Related Concepts
- [显式初始化缺失的项目知识库](../architecture/explicit-bundle-initialization.md): 当前发行版的架构决定。
- [上游 v0.4.3 与中文检索版兼容性审查](../research/upstream-v0-4-3-compatibility.md): 升级的兼容性证据与风险。
