# OKF 中文检索改造版

这是 [OKF Agent Memory](https://github.com/okf-memory/okf-agent-memory) v0.4.3 的中文检索改造版，使用 `go-ego/gse` 处理中文和中英文混合文本。它保持 OKF 的 Markdown 数据格式、CLI 和 MCP 接口兼容性。

## 源码

- 上游源码：https://github.com/okf-memory/okf-agent-memory
- 本仓库：https://github.com/passwordand/okf-agent-memory-cn

## 改动内容

- 基于上游 v0.4.3，包含加密 vault、Hub 同步、MCP 结构化返回和安全修复；
- 使用 gse 搜索模式处理中文分词；
- 保留英文、数字和概念 ID 检索；
- 分词器只初始化一次，避免重复加载词典；
- 使用 gse 编译时嵌入词典，发布后的 exe 不依赖 Go 模块缓存路径；
- 增加中文检索回归测试；
- 提供 Windows PowerShell 构建脚本。
- 新增 `okf_init` MCP 工具：项目库缺失时仍可连接，只有显式调用才创建 `knowledge/`；无参数 `okf init` 也固定创建当前目录的 `knowledge/`。

`okf_init` 从 v0.3.1-cn.4 起提供预编译版本；使用双库方案时还需更新已安装的 `mem-mcp.js` wrapper。

## 验证状态

本源码已合入上游 v0.4.3，同时保留 gse 中文检索、显式 `okf_init()` 和双库 wrapper。七个 MCP 工具成功调用均返回与 object 类型 `outputSchema` 对应的 `structuredContent`。

## 公开文档

- [主 README（含双 MCP wrapper 方案）](README.md)
- [OKF CLI 文档](docs/guides/CLI.md)
- [OKF MCP / 接入说明](docs/guides/GETTING_STARTED.md)
- [双 MCP wrapper 示例](examples/dual-mcp/mem-mcp.js)

## 下载预编译版本（推荐）

跨电脑部署无需重新编译。前往 [GitHub Releases](https://github.com/passwordand/okf-agent-memory-cn/releases)，下载与你的系统和架构匹配的文件：

- Windows x64：`okf-windows-amd64.exe`
- Windows ARM64：`okf-windows-arm64.exe`
- macOS Apple Silicon：`okf-darwin-arm64`
- macOS Intel：`okf-darwin-amd64`
- Linux x64：`okf-linux-amd64`
- Linux ARM64：`okf-linux-arm64`

下载后将文件重命名为 `okf.exe`（Windows）或 `okf`（macOS/Linux），并把它放到 PATH；macOS/Linux 还需要执行 `chmod +x okf`。随后运行 `okf version` 验证即可。只有修改 Go 源码，或 Release 没有覆盖目标平台时，才需要重新编译。

## 从源码构建

需要 Go 1.26 或更新版本。在 Windows PowerShell 中运行：

```powershell
./build.ps1
```

构建后的 `dist/okf-cn.exe` 是本地产物，不纳入 Git；跨电脑部署时可从源码重新构建。现有 `%USERPROFILE%\.config\agent-memory\bin\okf.exe` 不会被替换。
