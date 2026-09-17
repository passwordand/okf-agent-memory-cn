# OKF Agent Memory 中文检索改造版

这是 [OKF Agent Memory](https://github.com/okf-memory/okf-agent-memory) v0.3.1 的中文检索改造版。项目保留 OKF 的 Markdown 数据格式、CLI 和 MCP 接口，在搜索层集成 `go-ego/gse`，让中文和中英文混合查询可以正常分词和召回。

## 源码

- 上游源码：[okf-memory/okf-agent-memory](https://github.com/okf-memory/okf-agent-memory)
- 本仓库：[passwordand/okf-agent-memory-cn](https://github.com/passwordand/okf-agent-memory-cn)

## 适用场景

- 需要本地运行、低延迟、少维护的 Agent 长期记忆；
- 记忆以可读、可 Git 管理的 Markdown 文件保存；
- 在 Codex、OpenCode、WorkBuddy 等客户端中通过 MCP 使用；
- 中文检索不能接受原版 ASCII 分词造成的漏检。

## 主要改动

- 基于上游 v0.3.1（含 v0.2.0 / v0.3.0 的治理、`code_refs`、AAG / DMAA 和 CLI 加固）；
- 使用 gse 搜索模式处理中文分词；
- 保留英文、数字和概念 ID 检索；
- 分词器只初始化一次，避免每次查询重复加载词典；
- 使用编译时嵌入词典，发布后的可执行文件不依赖 Go 模块缓存路径；
- 增加中文检索回归测试；
- 提供 Windows PowerShell 构建脚本。

## 快速开始

### 下载预编译版本（推荐）

跨电脑部署无需重新编译。前往 [GitHub Releases](https://github.com/passwordand/okf-agent-memory-cn/releases)，按系统和架构下载对应文件：

- Windows x64：`okf-windows-amd64.exe`
- Windows ARM64：`okf-windows-arm64.exe`
- macOS Apple Silicon：`okf-darwin-arm64`
- macOS Intel：`okf-darwin-amd64`
- Linux x64：`okf-linux-amd64`
- Linux ARM64：`okf-linux-arm64`

下载后将文件重命名为 `okf.exe`（Windows）或 `okf`（macOS/Linux），并把它放到 PATH；macOS/Linux 还需要执行 `chmod +x okf`。随后运行 `okf version` 验证即可。只有修改 Go 源码，或 Release 没有覆盖目标平台时，才需要重新编译。

### 从源码构建（开发者）

#### Windows PowerShell

需要 Go 1.25 或兼容版本：

```powershell
gh repo clone passwordand/okf-agent-memory-cn
cd okf-agent-memory-cn
./build.ps1
```

构建产物为 `dist/okf-cn.exe`。构建脚本会先执行测试，再生成可执行文件。

### 常用命令

```powershell
# 严格校验 OKF 知识库
./dist/okf-cn.exe validate knowledge --strict --drift

# 中文搜索
./dist/okf-cn.exe search "登录鉴权" knowledge

# 按源码路径查找治理规则（上游 v0.2+）
./dist/okf-cn.exe search --for-path pkg/okf/types.go knowledge

# 查看概念
./dist/okf-cn.exe show architecture/layers knowledge --json

# 初始化一个新的 OKF 知识库
./dist/okf-cn.exe init .\my-project\knowledge

# 启动 MCP 服务
./dist/okf-cn.exe mcp knowledge
```

## 项目库 + 全局库：双 MCP wrapper

官方 `okf mcp <bundle>` 在启动时固定一个根目录。向已运行的服务传 `bundle` 不能扩大这个边界；Windows 上不同盘符的原始路径也无法通过同一进程的相对路径检查。因此需要**两个 MCP 条目、两份独立进程**，而不是一个服务静默回退。

目标：Agent 负责选库，用户只说「记住 / 查一下 / 做到哪了」。

| MCP 条目 | 启动参数 | 默认 bundle | 用途 |
|---|---|---|---|
| `okf-project` | `--scope project` | 从进程 cwd 向上找最近的 `knowledge/index.md` | 当前项目进度、架构、决策 |
| `okf-global` | `--scope global` | `~/.config/agent-memory/knowledge` | 通用偏好、跨项目约定 |

推荐目录：

```
~/.config/agent-memory/
  bin/okf.exe          # 或 macOS/Linux 下的 okf
  mem-mcp.js           # 复制 examples/dual-mcp/mem-mcp.js
  knowledge/           # 全局库：okf init 后使用
<项目根>/knowledge/    # 项目库，随仓库走
```

示例脚本：[examples/dual-mcp/mem-mcp.js](examples/dual-mcp/mem-mcp.js)。行为约定：

1. 必须显式 `--scope project|global`；缺参或未知参数非零退出。
2. 项目模式可用 `--project-root <绝对项目根>`，只检查该根下的 `knowledge/index.md`，不再向上搜。
3. 找不到项目库时输出 `PROJECT_BUNDLE_NOT_FOUND` 并失败；**不创建库、不回退全局**。
4. 项目模式若定位到全局库真实路径，输出 `PROJECT_SCOPE_REJECTED_GLOBAL` 并失败。
5. 子进程执行 `okf mcp <bundle>`，并把 `OKF_MCP_ROOT` 设为该 bundle 的真实绝对路径；定位日志只写 stderr。

### OpenCode

写入 `~/.config/opencode/opencode.json` 的 `mcp` 对象（保留其他服务）：

```json
{
  "mcp": {
    "okf-project": {
      "type": "local",
      "enabled": true,
      "command": ["node", "C:\\Users\\<你>\\.config\\agent-memory\\mem-mcp.js", "--scope", "project"]
    },
    "okf-global": {
      "type": "local",
      "enabled": true,
      "command": ["node", "C:\\Users\\<你>\\.config\\agent-memory\\mem-mcp.js", "--scope", "global"]
    }
  }
}
```

不要给 `okf-project` 写死全局 `cwd`，否则会锁到一个项目。某个项目定位失败时，再在**该项目**配置里加 `--project-root`。

### Codex

写入 `~/.codex/config.toml`：

```toml
[mcp_servers.okf-project]
command = "node"
args = ["C:\\Users\\<你>\\.config\\agent-memory\\mem-mcp.js", "--scope", "project"]
required = false

[mcp_servers.okf-global]
command = "node"
args = ["C:\\Users\\<你>\\.config\\agent-memory\\mem-mcp.js", "--scope", "global"]
```

`required = false` 允许无项目库时项目服务失败，同时全局服务仍可用。

### WorkBuddy

写入 `~/.workbuddy/mcp.json`：

```json
{
  "mcpServers": {
    "okf-project": {
      "type": "stdio",
      "command": "node",
      "args": ["C:\\Users\\<你>\\.config\\agent-memory\\mem-mcp.js", "--scope", "project"]
    },
    "okf-global": {
      "type": "stdio",
      "command": "node",
      "args": ["C:\\Users\\<你>\\.config\\agent-memory\\mem-mcp.js", "--scope", "global"]
    }
  }
}
```

权限里放行 `mcp__okf-project` 与 `mcp__okf-global`。不要改 marketplace 里会被覆盖的 `connectors/*/mcp.json`。

### Agent 规则

在客户端 `AGENTS.md`（或 WorkBuddy 的 `CODEBUDDY.md`）写明：

- 项目进度 / 决策走 `okf-project`，偏好 / 跨项目约定走 `okf-global`。
- 两边都查时分别调用并注明来源。
- 无项目库时说明缺失；不得用全局进度冒充项目进度，也不得把项目内容静默写入全局。
- 调用使用客户端实际服务器命名空间（例如 OpenCode 的 `okf-project_okf_show`）。

改配置后必须重启客户端。不要把真实记忆数据提交到本仓库。

## 公开文档

- [中文 README（简版）](README-cn.md)
- [OKF CLI 文档](docs/guides/CLI.md)
- [OKF MCP / 接入说明](docs/guides/GETTING_STARTED.md)
- [双 MCP wrapper 示例](examples/dual-mcp/mem-mcp.js)
- [上游项目](https://github.com/okf-memory/okf-agent-memory)

## 验证状态

已基于上游 v0.3.1 合入 gse 中文检索。`go test ./pkg/okf`、`go vet ./...` 和 OKF 严格校验应通过。中文混合语料覆盖 `登录`、`鉴权`、`知识库`、`JWT` 等查询。

Windows 下官方 `cmd/okf` 的部分 MCP 安全测试仍有原有兼容性问题，与 gse 搜索改动无关。

严格 MCP 客户端（OpenCode、Codex 等）请使用 **v0.3.1-cn.3 或更新**。`v0.3.1-cn.1` 的非 object `outputSchema` 会被标准 MCP SDK 在 `tools/list` 拒绝；`v0.3.1-cn.2` 保留的 object schema 因无 `structuredContent` 会在调用时被拒绝。本版起 6 个工具均不声明 `outputSchema`。

## 许可证

本项目沿用上游 OKF Agent Memory 的 MIT 许可证。
