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

## 接入原则

MCP 服务启动时指定要使用的 `knowledge` bundle。项目库和全局库可通过客户端配置分别指定路径；不要把真实记忆数据提交到本仓库。仓库只提供引擎、示例知识库和部署文档。

## 公开文档

- [中文 README（简版）](README-cn.md)
- [OKF CLI 文档](docs/guides/CLI.md)
- [OKF MCP / 接入说明](docs/guides/GETTING_STARTED.md)
- [上游项目](https://github.com/okf-memory/okf-agent-memory)

## 验证状态

已基于上游 v0.3.1 合入 gse 中文检索。`go test ./pkg/okf`、`go vet ./...` 和 OKF 严格校验应通过。中文混合语料覆盖 `登录`、`鉴权`、`知识库`、`JWT` 等查询。

Windows 下官方 `cmd/okf` 的部分 MCP 安全测试仍有原有兼容性问题，与 gse 搜索改动无关。

## 许可证

本项目沿用上游 OKF Agent Memory 的 MIT 许可证。
