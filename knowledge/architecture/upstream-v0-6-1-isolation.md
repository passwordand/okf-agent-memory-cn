---
type: Decision
title: 上游 v0.6.1 隔离升级与双库兼容方案
description: 在独立工作树合入 v0.6.1，保留中文分词和显式初始化，强化双库与 Windows junction 边界；测试通过部分与环境限制分开记录。
generated: { by: agent/cli, at: "2026-10-15T03:18:49Z" }
status: stable
---

# 上游 v0.6.1 隔离升级试验

2026-10-14，按用户要求在独立工作树 codex/okf-v061-isolation 中集成 v0.4.3 到 v0.6.1 的差异，未切换现用客户端、程序或 wrapper，也未提交、推送或发布。

保留 gse 中文分词、嵌入词典、显式 okf_init、缺库不回退及 CLI 默认 knowledge/。现有 wrapper 设置 OKF_MCP_ROOT 时，服务端将搜索和读取限制在绑定库内，工具 schema 同时声明本库范围。未显式绑定根目录的独立 MCP 与 CLI 保留上游多层能力。

Windows Go 1.26.0 的 filepath.EvalSymlinks 不展开部分 ModeIrregular junction。试验版使用文件句柄最终路径解析，覆盖库加载、MCP 边界和写入路径；真实 junction 越界读写测试通过。

311 项 Go 测试（含子用例）通过；17 项平台 / 权限条件跳过，另显式排除 TestCLIAgentsLint 和 TestCreateAndCheckToolSymlinks 两项文件符号链接权限受限测试。wrapper 6 项与真实 stdio MCP 集成 13 项通过；go vet 和项目 / 三个示例库的严格数据校验通过。

完整 --agents 检查受 Windows Git 符号链接检出限制，race 需要本机当前不可用的 cgo 工具链。尚未做真实 Hub / Registry 联网同步或非 Windows 运行验证。详细报告保存在 docs/project/UPGRADE_V061_ISOLATION_REPORT.md。

# Related Concepts
- [上游 v0.4.3 与中文记忆功能的集成决定](upstream-v0-4-3-integration.md): 沿用中文版保留功能，并在 v0.6.1 试验中延续双库隔离

## 2026-10-15 正式适配

用户已授权发布 v0.6.1-cn.1、更新 GitHub 与替换本机版本，并明确仅关注 Windows 正常使用。沿用已验证的中文与双库适配，不追加非 Windows 实机或 race 测试；本机采用版本化 exe 并保留旧版及 wrapper 备份。
