# OKF Agent Memory 中文检索改造版

这是 [OKF Agent Memory](https://github.com/okf-memory/okf-agent-memory) v0.4.3 的中文检索改造版。项目保留 OKF 的 Markdown 数据格式、CLI 和 MCP 接口，在搜索层集成 `go-ego/gse`，让中文和中英文混合查询可以正常分词和召回。

## 源码

- 上游源码：[okf-memory/okf-agent-memory](https://github.com/okf-memory/okf-agent-memory)
- 本仓库：[passwordand/okf-agent-memory-cn](https://github.com/passwordand/okf-agent-memory-cn)

## 适用场景

- 需要本地运行、低延迟、少维护的 Agent 长期记忆；
- 记忆以可读、可 Git 管理的 Markdown 文件保存；
- 在 Codex、OpenCode、WorkBuddy 等客户端中通过 MCP 使用；
- 中文检索不能接受原版 ASCII 分词造成的漏检。

## 主要改动

- 基于上游 v0.4.3，包含加密 vault、Hub 同步、MCP 结构化返回和安全修复；
- 使用 gse 搜索模式处理中文分词；
- 保留显式 `okf_init()`、无参数 `okf init` 创建 `knowledge/` 和双库 wrapper；
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

需要 Go 1.26 或更新版本：

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

# 在当前项目创建 knowledge/；也可显式传入目标路径
./dist/okf-cn.exe init

# 启动 MCP 服务
./dist/okf-cn.exe mcp knowledge
```

## 项目库 + 全局库：双 MCP wrapper

官方 `okf mcp <bundle>` 在启动时固定一个根目录。向已运行的服务传 `bundle` 不能扩大这个边界；Windows 上不同盘符的原始路径也无法通过同一进程的相对路径检查。因此需要**两个 MCP 条目、两份独立进程**，而不是一个服务静默回退。

目标：Agent 负责选库，用户只说「记住 / 查一下 / 做到哪了」。

| MCP 条目 | 启动参数 | 默认 bundle | 用途 |
|---|---|---|---|
| `okf-project` | `--scope project` | 优先找最近的 `knowledge/index.md`；缺库时绑定 cwd 下的 `knowledge/` | 当前项目进度、架构、决策 |
| `okf-global` | `--scope global` | `~/.config/agent-memory/knowledge` | 通用偏好、跨项目约定 |

项目库尚不存在时，`okf-project` 仍可连接；只有用户明确要求初始化，Agent 才调用 `okf_init()`。启动 MCP、搜索失败或缺少参数都不会自动创建文件。直接使用 CLI 时，运行 `okf init` 会在当前目录创建 `knowledge/`；`okf init .` 则显式在当前目录创建根级库。

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
2. 项目模式可用 `--project-root <绝对项目根>` 固定目标；未指定时优先向上查找已有项目库，找不到则把进程 cwd 作为项目根，不依赖 Git。
3. 项目库缺失时仅启动绑定到 `<项目根>/knowledge` 的 MCP 服务；调用 `okf_init()` 才创建库，不回退全局库。全局库缺失时仍报 `GLOBAL_BUNDLE_NOT_FOUND`。
4. 项目模式若定位到全局库真实路径，输出 `PROJECT_SCOPE_REJECTED_GLOBAL` 并失败。
5. 子进程执行 `okf mcp <bundle>`，并把 `OKF_MCP_ROOT` 限定为该 bundle 路径；缺库时工作目录设为项目根，已有库时设为 bundle 目录。定位日志只写 stderr。

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

本源码已合入上游 v0.4.3，并保留中文分词及项目库显式初始化。七个 MCP 工具均声明 object 类型 `outputSchema`，成功调用返回 `structuredContent`；错误调用使用 `isError`。测试结果以本次升级的本地验证为准。

## 许可证

本项目沿用上游 OKF Agent Memory 的 MIT 许可证。
