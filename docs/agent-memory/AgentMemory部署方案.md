# Agent Memory 部署方案与步骤

> 面向执行者（AI agent 或人）。本文只描述**方案与步骤**，所有落地程序由执行者按「实现规格」自行编写。
>
> 平台：Windows 10/11 · 需要 Node.js >= 18、git、Python 3.9+
> 版本：v1.2 · 2026-09-11 · 项目 / 全局双 MCP 方案

---

## 执行者须知

1. 步骤有严格依赖顺序，按 **第 5 → 6 → 7 章** 执行。
2. 第 7 章（MegaMemory 迁移）是**确认制**：用户没有明确点头前，**绝不删除任何 `.megamemory`**。
3. 标注 **【实现规格】** 的地方只给"职责 + 验收标准"，不限定语言与写法，由执行者自行实现。
4. 任何改动用户既有配置（`opencode.json`、`config.toml`、wrapper、`AGENTS.md`）的动作，先备份原文件；更新已有记忆指令，不重复追加冲突规则。
5. 本版是待实施的操作规格。更新本文不代表双服务已经部署；不得直接把配置示例用于尚未支持新参数的旧 wrapper。
6. 文中的 `<你>`、`<项目根>` 均为占位符，必须按目标电脑解析；其他机器的盘符和用户名不是本机事实。迁移和 Git 提交 / 推送仍须取得相应授权。

---

## 0. 目标

用 OKF（Open Knowledge Format：Markdown + YAML 开放标准）替换已停更的 MegaMemory，得到一套：

- **跨 agent**：OpenCode、Codex 共用同一套 OKF 数据与引擎，通过项目 / 全局两个 MCP 条目接入，不绑定任何一家；
- **人可读**：全部是 Markdown，Obsidian 直接打开，可 git diff；
- **跨机**：知识库走 git 私仓，引擎各机各放一份；
- **分层记忆**：规程 / 进度 / 知识 / 时间线各归其位，agent 一眼知道进度；
- **可回滚**：迁移采用确认制，未确认不删任何原始数据。

---

## 1. 设计原则

1. **数据中立**：记忆是磁盘上的 OKF bundle，不属于任何 agent；换引擎、换 agent 都不迁数据。
2. **接入中立**：只用纯 stdio MCP，所有支持 MCP 的 agent 平等。
3. **进度锚点固定**：进度不是"搜出来的"，是固定文件 `knowledge/progress/current.md`，直读。
4. **不靠自觉**：把"开工读进度、收工写进度"写死进 `AGENTS.md`，两个 agent 都会读到。
5. **确认制删除**：迁移与删除分离，删除必须用户二次确认。
6. **双入口、分开写**：`okf-project` 管当前项目，`okf-global` 管通用记忆；项目库缺失时不得静默写入全局。
7. **进程作用域固定**：MCP 服务启动时确定 bundle 和根目录。切换项目必须由客户端启动对应项目的服务实例，或明确重启 / 重配；聊天里换路径不会重定位现有服务。

---

## 2. 目标架构

### 2.1 架构图

<img src="architecture.svg" alt="Agent Memory 架构图" width="100%">

每个客户端连接两个独立的 stdio 服务进程，两个进程使用同一个 `okf.exe` 文件。多个客户端同时连接时可产生更多进程；“两个条目”不等于整台机器永远只有两个进程。

| MCP 条目 | 默认 bundle | 用途 |
|---|---|---|
| `okf-project` | 从目标项目定位的 `knowledge/` | 当前项目进度、架构、决策、问题 |
| `okf-global` | 当前用户 `~/.config/agent-memory/knowledge` | 通用偏好、跨项目经验、公共约定 |

### 2.2 组件

| 组件 | 位置 | 作用 | 进 git |
|---|---|---|---|
| 引擎 `okf.exe` | `~/.config/agent-memory/bin/` | OKF 解析 / 校验 / BM25 检索 / MCP 服务 | 否 |
| wrapper `mem-mcp.js` | `~/.config/agent-memory/` | 提供 `--scope project` / `--scope global` 两个启动模式，分别启动引擎 | 是 |
| 全局库 `knowledge/` | `~/.config/agent-memory/` | 跨项目通用记忆 | 是 |
| 项目库 `knowledge/` | 各项目根 | 单项目记忆，随项目 | 随项目 |
| 注入插件 | `~/.config/opencode/plugins/` | OpenCode 压缩时注入进度（可选） | 否 |

### 2.3 目录树

```
~/.config/agent-memory/            ← git 私仓
  mem-mcp.js                       双模式 MCP wrapper（实现见 5.3）
  knowledge/                       全局库（OKF bundle）
    index.md  log.md
    progress/current.md            固定进度锚点
    concepts/
  bin/okf.exe                      引擎
  backup/                          迁移备份

<某个项目>/
  knowledge/                       项目库（OKF bundle）
    index.md  log.md  progress/current.md  concepts/
  AGENTS.md
```

---

### 2.4 为什么采用双 MCP

已核对 [OKF v0.1.5 的 MCP 源码](https://github.com/okf-memory/okf-agent-memory/blob/v0.1.5/cmd/okf/mcp.go) 中 `RunMCPServerIO`、`getMCPTools` 和 `resolveBundleDir`：

- 6 个工具均有可选 `bundle` 参数；单进程可在允许范围内访问多个 bundle。
- 启动时先取 `OKF_MCP_ROOT`；未设置时，启动 bundle 的目录名为 `knowledge` 则取父目录，否则取 bundle 本身；空或 `.` 则取当前目录。随后绝对化并尝试解析符号链接，存入服务器对象。
- 调用时，相对 bundle 基于服务器根目录解析；检查真实路径（包括符号链接和最近存在的祖先）。`filepath.Rel` 报错、结果等于 `..` 或以 `..` 加路径分隔符开头，都会拒绝。不是简单拒绝所有名字以两个点开头的目录。
- 不同 Windows 盘符的原始路径无法通过这项相对路径检查；同盘但在根目录之外也会拒绝。符号链接不是本方案的解决路径。
- `OKF_MCP_ROOT` 设为同盘共同祖先是一种替代配置，但会扩大访问范围，且不能同时覆盖 C、D、E 等不同盘符的原始路径。本方案使用两个独立根目录。

本文 §5.3 进一步规定：每个子进程的 `OKF_MCP_ROOT` 显式设为选中的 bundle 绝对真实路径，避免继承其他进程遗留的宽范围设置。

这些结论针对核对过的 v0.1.5；更换引擎版本后要复核边界行为。双条目不会自动完成跨库合并搜索，Agent 必须分别调用并保留来源。

---

## 3. 记忆协议（必须遵守）

### 3.1 四层结构

| 层 | 载体 | 解决 | 进入 context 的方式 |
|---|---|---|---|
| 规程 procedural | `AGENTS.md` | 规则 / 习惯 | 每次自动加载 |
| 进度 working | `knowledge/progress/current.md` | 做到哪、下一步、阻塞 | 会话开头直读 |
| 知识 semantic | `knowledge/concepts/*.md` | 架构、决策、模式 | 按需 `okf_search` |
| 时间线 episodic | `knowledge/log.md` | 什么时候做了什么 | 追加式回溯 |

### 3.2 进度模板 `progress/current.md`

```markdown
---
type: Progress
title: 当前进度
project: <项目名>
status: stable
updated: 2026-09-10
---

# 当前阶段：<一句话>

## 已完成
- ...

## 下一步
- ...

## 阻塞
- ...

## 关键记忆
- <必须一直记住的坑/约定>
```

原则：只放"当前状态"，控制在十几行；历史沉到 `log.md`，细节沉到 `concepts/`。

### 3.3 概念模板 `concepts/<id>.md`

```markdown
---
type: Decision
title: <标题>
description: <一句话描述>
tags: [模块, 主题]
aliases: [同义词, 英文别名, 中文别名]
generated: { by: agent/<name>, at: 2026-09-10T08:00:00Z }
status: stable
---

# <标题>

## 背景
...

## 内容
...

## 关联
- [相关概念](../concepts/other.md) — depends_on
```

- 同义词、中英文别名应写入实际参与检索的标题、description 或正文；`aliases` 字段是否参与检索须按引擎版本验证。关键词补充不能保证达到语义检索效果，中文换说法要用实际查询验收。
- `status`：`draft | stable | stale | archived`；可选 `stale_after`，过期自动降权。

### 3.4 全局 `AGENTS.md` 记忆指令（替换原 Agent Memory 段）

```markdown
## Agent Memory（跨会话记忆）

长期记忆分为两层，通过两个 MCP 服务读写：
- `okf-project`：当前项目的 `knowledge/`，保存项目进度、架构、决策、问题。
- `okf-global`：当前用户 `~/.config/agent-memory/knowledge`，保存通用偏好、跨项目经验、公共约定。
两边都有 `okf_search / okf_show / okf_create / okf_update / okf_relate / okf_validate`。
调用时使用客户端实际提供的服务器命名空间，不能只按裸工具名猜服务。

### 会话开头
1. 确认项目服务实际定位的项目；先通过 `okf-project` 读 `progress/current`。
2. 需要通用规则或经验时，另通过 `okf-global` 检索 / 直读相关概念。
3. 无项目或项目库缺失时明确说明；全局服务仍可用于通用记忆，但不得用全局进度冒充项目进度，也不得静默把项目内容写进全局库。

### 怎么查
- 问「进度 / 做到哪 / 下一步」：先读对应作用域的 `progress/current`，需要背景再搜索。
- 问项目模块、项目决策：先查 `okf-project`。
- 问偏好、通用约定、跨项目经验：查 `okf-global`。
- 两边都查时分别调用，回答注明来源；不自动复制整份项目记忆到全局。
- 切换项目后先确认服务已重定位；向旧服务传新 `bundle` 不能扩大它的边界。

### 任务结束后
1. 有项目状态变化时，只更新项目 `progress/current`；全局进度仅记录全局记忆系统自身状态。
2. 写前先在目标库搜索，能更新就不新建，避免重复。
3. 新决策在所属库创建概念，并关联同库概念。`okf_relate` 不用于跨库路径。
4. 跨项目经验先提炼通用部分，再写全局库；保留来源说明。
5. 优先用 MCP 写入，让引擎自动维护 `log.md` / 索引；确认已有自动日志后不再重复追加。
6. 多个 Agent 不同时覆盖同一进度页；更新前重读最新正文，有冲突先合并内容。
```

说明：客户端执行规则的效果须验收，`AGENTS.md` 不是服务端自动路由或事务锁。缺少项目库时，如用户需要持久化该项目记忆，先按已授权范围初始化真实项目库，再重启项目服务。

### 3.5 校验命令

```powershell
& "$env:USERPROFILE\.config\agent-memory\bin\okf.exe" validate <bundle路径> --strict
```

---

## 4. 前置检查

确认以下命令可用，缺什么先装：

```powershell
node --version     # >= 18
git --version
python --version   # >= 3.9
```

确认 OpenCode 与 Codex 已安装；记录两个配置文件位置：

- OpenCode：`~/.config/opencode/opencode.json`
- Codex：`~/.codex/config.toml`

---

## 5. A 机部署

### 5.1 建目录

创建 `~/.config/agent-memory/` 及子目录 `bin/`、`knowledge/`。

**验收**：目录存在。

### 5.2 安装引擎

从 okf-agent-memory 的 GitHub Releases 下载 `okf-windows-amd64.exe`（最新版），放到
`~/.config/agent-memory/bin/okf.exe`。可用：

```powershell
$url = "https://github.com/okf-memory/okf-agent-memory/releases/latest/download/okf-windows-amd64.exe"
Invoke-WebRequest -Uri $url -OutFile "$env:USERPROFILE\.config\agent-memory\bin\okf.exe"
& "$env:USERPROFILE\.config\agent-memory\bin\okf.exe" version
```

**验收**：`okf version` 正常输出。

### 5.3 【实现规格】双模式 MCP wrapper

**产出**：升级 `~/.config/agent-memory/mem-mcp.js`。以下参数为本方案待实现接口，不是 OKF 原生命令参数。

| 启动方式 | 行为 |
|---|---|
| `node mem-mcp.js --scope project` | 从进程实际 cwd 逐级向上找最近的 `knowledge/index.md` |
| `node mem-mcp.js --scope project --project-root <绝对项目根>` | 明确指定项目根，只检查该根下的 `knowledge/index.md`，不继续向其他祖先搜索 |
| `node mem-mcp.js --scope global` | 使用 `os.homedir()` 下 `.config/agent-memory/knowledge`，不依赖 cwd |

**共同职责**：

1. 要求显式 `--scope`；缺失、未知参数或全局模式混入项目参数，输出 stderr 错误并非零退出。旧无参启动不再静默回退。
2. 选库后取得 bundle 的绝对真实路径，检查 `index.md` 存在。项目模式不得选中全局库的真实路径。
3. 找不到项目库时，明确报告 `PROJECT_BUNDLE_NOT_FOUND` 和起始目录，非零退出；不创建库、不回退全局。全局库缺失也明确失败，先执行 §5.4。
4. 启动 `okf.exe mcp <bundle绝对路径>`。子进程 cwd 设为 bundle 路径，子进程环境中的 `OKF_MCP_ROOT` 显式覆盖为该 bundle 的真实绝对路径；不更改系统环境变量。
5. stdin/stdout 全量透传，错误与定位信息仅写 stderr。退出码、信号和子进程清理要正确传递；不得向 stdout 混入日志。
6. stderr 输出一条可验收的启动信息：scope、起始 cwd、选中 bundle、root。不得包含密钥。

**定位限制**：wrapper 无法推断客户端没传来的项目。如果 cwd 错误，优先在项目级配置中明确 cwd / `--project-root`；不要把某个项目的固定路径写成所有项目共用的全局配置。桌面客户端是否按任务启动服务，必须实测。

**验收**：用隔离测试库验证两个模式各自完成 MCP 初始化、列出 6 个工具、直读自己的 `progress/current`。还需验证空项目失败、不同 cwd 的全局模式始终选同一库，以及边界外 bundle 被拒绝。完成实现后才应用 §5.6 / §5.7。

### 5.4 初始化全局库

用引擎初始化全局 OKF bundle，并手动创建进度层：

```powershell
$root = "$env:USERPROFILE\.config\agent-memory"
& "$root\bin\okf.exe" init "$root\knowledge"
New-Item -ItemType Directory -Force -Path "$root\knowledge\progress" | Out-Null
```

把 3.2 的进度模板写入 `knowledge\progress\current.md`（内容按实际填写）。

**验收**：`validate` 通过（3.5）。

### 5.5 写全局 AGENTS.md 指令

备份后，用 3.4 的「记忆指令」替换 OpenCode 与 Codex 各自 `AGENTS.md` 中原有 Agent Memory 段；保留其他规则。

**验收**：两个 agent 启动后都能在指令里看到该段落。

### 5.6 接入 OpenCode

先备份 `~/.config/opencode/opencode.json`。确认 §5.3 的新 wrapper 已实现后，将下面两个条目合入现有 `mcp` 对象，保留其他服务：

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

迁移原 `okf-memory` 时，在同次配置更新中禁用旧条目；若旧 `megamemory` 仍在使用，备份后禁用，不删除记忆数据。待新服务验收成功，再移除废弃配置条目。

**验收**：从真实项目启动 OpenCode，两边各列出 6 个工具且可分别调用。记录实际工具命名，确认无覆盖 / 撞名。若项目 cwd 不正确，在该项目的配置中给 wrapper 增加 `--project-root` 与真实绝对路径，再验收。无项目会话可以只启用全局条目，不能以全局库伪装项目服务。

### 5.7 接入 Codex

先备份 `~/.codex/config.toml`；确认新 wrapper 已实现后，合入以下配置并保留其他服务：

```toml
[mcp_servers.okf-project]
command = 'node'
args = ['C:\Users\<你>\.config\agent-memory\mem-mcp.js', '--scope', 'project']
required = false

[mcp_servers.okf-global]
command = 'node'
args = ['C:\Users\<你>\.config\agent-memory\mem-mcp.js', '--scope', 'global']
```

旧 `okf-memory` 条目先设 `enabled = false`，双服务验收后再移除。不要重复留三组可写记忆工具。

Codex 官方支持 stdio 服务的 `cwd` 字段，也支持受信任项目的 `.codex/config.toml`。如果实际启动 cwd 不可靠，在**该项目的配置**中覆盖项目服务，例如：

```toml
[mcp_servers.okf-project]
command = 'node'
args = ['C:\Users\<你>\.config\agent-memory\mem-mcp.js', '--scope', 'project', '--project-root', '<项目根绝对路径>']
cwd = '<项目根绝对路径>'
required = false
```

以上项目配置使用本机绝对路径，不适合未经替换直接跨机复用。`required = false` 允许可选项目服务缺库时失败；仍须在该客户端版本验收全局服务可用。

**验收**：分别从两个不同项目开始新任务，确认各自项目根；同一任务内分别通过两组工具读取项目 / 全局进度。当前会话中已观察到 `mcp__okf_memory__okf_show` 的服务器命名空间，但新服务的实际名称仍以客户端工具列表为准，不把推测的前缀写死。

配置依据：[Codex MCP 官方文档](https://developers.openai.com/codex/mcp/)。

### 5.8 【实现规格】OpenCode 压缩注入插件（可选增强）

**产出**：`~/.config/opencode/plugins/memory-inject.js`。

**职责**：监听 OpenCode 的 `experimental.session.compacting` 钩子，在生成续接摘要前，把当前项目 `knowledge/progress/current.md` 的内容追加进 `output.context`。

**说明**：本节只规定 OpenCode 插件；钩子名称与生命周期按所装版本验证。Codex 的最低验收以 §5.5 的记忆规则为准，不在本文假定其没有其他扩展机制。注入插件必须采用与项目服务一致的定位结果，缺库时跳过并说明，不能注入全局进度冒充项目进度。

**验收**：触发一次会话压缩，续接上下文里含进度内容。

### 5.9 端到端验收

优先使用隔离测试 bundle，分别放不同的 `progress/current` 标识；不要把测试内容写进真实项目进度。

| 场景 | 通过标准 |
|---|---|
| 两服务并存 | 有项目时，两组各 6 个工具都能调用，无命名覆盖 |
| 项目定位 | 项目 A 及子目录均定位 A；项目 B 新任务定位 B |
| 全局定位 | 不同 cwd 都定位当前用户固定全局库 |
| 同名概念隔离 | 两库 `progress/current` 返回各自内容，回答标注来源 |
| 写入分流 | 项目概念仅写项目，全局概念仅写全局；自动日志不重复 |
| 无项目库 | 项目服务明确失败；全局服务仍能操作通用记忆 |
| 跨盘并存 | 测试项目库与全局库在不同本机盘符时，两服务各自访问成功；没有第二盘可标待测，不伪造通过 |
| 越界拒绝 | 项目服务传全局 bundle 被拒，反向也被拒；测试库应放在互不包含的位置 |
| 切换项目 | 旧服务不会自动变化；新任务或重启后读取正确新项目 |
| 双客户端 | OpenCode 与 Codex 均完成以上关键场景；串行更新同一进度，避免互相覆盖 |
| 完整性 | 两库 `validate --strict` 通过，真实数据未被测试覆盖 |

---

## 6. B 机部署

其他电脑环境与 A 机相同。

### 6.1 全局库 git 化（在 A 机做一次）

把 `~/.config/agent-memory/` 初始化为 git 私仓，`bin/` 与 `backup/` 加入 `.gitignore`（引擎和备份不进库），push 到私有远端。

### 6.2 B 机 clone + 安装

1. `git clone <私仓地址>` 到 `~/.config/agent-memory/`；
2. 放好 `bin/okf.exe`（同 5.2）；
3. 按 5.3 确保 `mem-mcp.js` 支持双模式参数；
4. 按 5.6、5.7 在两个客户端分别配置项目 / 全局两个条目，替换成本机用户名及路径；
5. 按 5.5 写 B 机的 AGENTS.md 指令。

**验收**：B 机同一套验收（5.9）。

### 6.3 项目库

项目库随项目仓库走。B 机 clone 后，从正确项目目录启动客户端，并检查 wrapper 的定位日志；如果客户端 cwd 不可靠，按 §5.6 / §5.7 配置本机项目根。全局库通过独立 `okf-global` 服务始终可用。

---

## 7. 迁移 MegaMemory（确认制）

<img src="migration-flow.svg" alt="迁移流程图" width="100%">

**核心操作逻辑：迁移与删除彻底分离。扫描只读 → 转换 `--apply` → 删除 `--purge --confirm`。用户未确认，`.megamemory` 一直原地保留。**

### 7.0 【实现规格】迁移工具

**产出**：一个迁移工具（建议 Python，本机有 3.12），放在 `~/.config/agent-memory/`。

**职责**：

1. 扫描 `%USERPROFILE%`、`D:\`、`E:\` 下的 `.megamemory/knowledge.db`；
2. 只读打开每个库，读 `nodes`（跳过 `removed_at` 非空的）与 `edges`；
3. 按 7.3 分类、按 7.4 映射把概念写入目标 OKF bundle；
4. 三种模式：
   - **默认（dry-run）**：只读，打印清单（库路径、节点数、边数、分类、目标），写 `migration-manifest.json`，**不写任何知识文件、不删任何东西**；
   - **`--apply`**：执行转换写入；
   - **`--purge --confirm`**：删除已迁移的原库（两参数必须同用）。

**硬性要求**：

- `--purge` 不带 `--confirm` 时**必须拒绝执行**；
- 删除前**必须**先把待删库复制到 `backup/purge-<时间戳>/`；
- 转换阶段不得修改或删除 `.megamemory`；
- 目标概念文件若已存在，加后缀避免覆盖。

### 7.1 备份（必做）

在扫描出清单后，把每个 `.megamemory` 目录整体复制到
`~/.config/agent-memory/backup/megamemory-<时间戳>/`。

### 7.2 dry-run

运行迁移工具（默认模式），核对输出的清单。此步结束时应只有两个新文件：`migration-manifest.json`，以及可能的备份。

### 7.3 分类规则

| 分类 | 判定 | 默认动作 |
|---|---|---|
| active | 真实项目（`E:\UnityProgram\*`、`D:\*`、`Desktop\*` 等） | 迁到该项目根的 `knowledge/` |
| temporary | `Documents\Codex\*\new-chat`、会话缓存目录 | 不迁移（`--include-temporary` 才并入全局库） |

### 7.4 字段映射

| MegaMemory | OKF |
|---|---|
| `nodes.id` | concept `id` / 文件名 |
| `nodes.kind` | `type` |
| `nodes.name` | `title` |
| `nodes.summary` | `description` |
| `nodes.why` | 正文「背景 / 理由」 |
| `nodes.file_refs` | 正文「相关文件」 |
| `nodes.parent_id` | 「关联 → 上级」链接 |
| `edges(from,to,relation,description)` | 正文「关联」链接 |
| `removed_at` 非空 | 跳过 |

### 7.5 转换与校验

运行 `--apply`。之后对每个目标 bundle 执行 `validate --strict`，并人工抽查 1~2 个概念文件的通顺度与链接有效性。

### 7.6 人工确认后才删除（关键闸门）

转换完成后，迁移工具把**已成功迁移**的库写入 `~/.config/agent-memory/pending-delete.txt`。

此时：

1. 用户逐条核对清单，确认这些库已迁移、内容可在 `knowledge/` 找到；
2. **只有用户明确确认后**，才运行 `--purge --confirm`；
3. 工具先备份到 `backup/purge-<时间戳>/`，再删除 `knowledge.db` / `-wal` / `-shm`，目录空了才删 `.megamemory`。

**未确认前，禁止执行任何删除。**

---

## 8. 验收清单

- [ ] 已记录引擎版本，确认新版 wrapper 支持双模式参数
- [ ] wrapper、客户端配置和记忆规则已备份
- [ ] 项目库与全局库 `validate --strict` 通过
- [ ] OpenCode 的两个服务均能列工具、直读、正确分流写入
- [ ] Codex 的两个服务均能列工具、直读、正确分流写入
- [ ] 两客户端实际工具名无覆盖，旧 `okf-memory` 已禁用或移除
- [ ] 项目 A / B 定位、无项目库、跨盘和越界拒绝按 §5.9 逐项记录
- [ ] 切项目后服务已重建 / 重定位，没有复用错误进度
- [ ] Agent 规则已替换，未保留“项目缺失静默写全局”的旧规则
- [ ] B 机替换路径后通过同一套验收
- [ ] 迁移 dry-run 清单已复核
- [ ] 仅当用户明确确认后，才执行原库删除

---

## 9. 回滚

| 想回退 | 操作 |
|---|---|
| 双服务回退到原 OKF 接入 | 从备份同时恢复 wrapper、客户端配置和原记忆规则；不要只恢复其中一项造成参数不兼容，知识文件保留 |
| 迁回 MegaMemory | 从 `backup/megamemory-*` 复制回原位，恢复两个 config 的 `megamemory` 块，移除本次新增 `okf-project` / `okf-global` 配置条目 |
| 撤销全局库改动 | `git -C ~/.config/agent-memory revert` |
| 撤销项目知识 | `git -C <项目> revert` |

---

## 10. 排障

| 现象 | 排查 |
|---|---|
| agent 说没有 okf 工具 | 检查 config 的 MCP 块路径；重启 agent；手动跑 wrapper 看报错 |
| 项目服务缺失 / `PROJECT_BUNDLE_NOT_FOUND` | 查看 stderr 中真实 cwd；确认项目根有 `knowledge/index.md`，必要时在项目配置指定根路径 |
| 两个服务都读出全局内容 | 检查是否仍使用旧 wrapper / 静默回退逻辑；项目模式必须拒绝选中全局库 |
| `escapes server root` | 调用了错误作用域或越界 bundle；换正确服务，不靠改盘符或符号链接规避 |
| 切换项目后仍是旧进度 | 重启或重新创建项目服务；聊天文本不会改变已运行的进程作用域 |
| 同名工具被覆盖 | 查看客户端的服务器命名空间；未解决前不得判定双服务部署成功 |
| 中文检索不到 | 在实际检索字段补关键词 / 同义词并实测；确认 UTF-8，不把未验证的 aliases 当成语义检索 |
| `okf.exe` 无法执行 | 确认下载的是 `windows-amd64` 且未被杀软拦截 |
| Python 读不了 db | libsql 文件通常为标准 SQLite；失败改用 Node 的 libsql 读取 |

---

## 附录：配置改动对照

| 文件 | 删除 | 新增 |
|---|---|---|
| `~/.config/opencode/opencode.json` | 禁用旧 `mcp.okf-memory` / 废弃 `mcp.megamemory`，验收后移除 | `mcp.okf-project` + `mcp.okf-global` |
| `~/.codex/config.toml` | 禁用旧 `okf-memory` / 废弃 `megamemory`，验收后移除 | `[mcp_servers.okf-project]` + `[mcp_servers.okf-global]` |
| 两客户端各自读取的 `AGENTS.md` | 原 Agent Memory 段 | §3.4 双层读写规则；其他内容保留 |
| `~/.config/agent-memory/mem-mcp.js` | 旧的无参自动回退行为 | §5.3 双模式定位；备份后升级 |
| `~/.config/opencode/tool/megamemory.ts` | 整个文件 | — |
| `~/.config/opencode/commands/bootstrap-memory.md` | 整个文件 | — |
| `~/.config/opencode/commands/save-memory.md` | 整个文件 | — |

> 末三项的删除，统一在 7.6 用户确认并通过校验后再执行。


## 本版变更记录

- 2026-09-11：v1.2 改为 `okf-project` / `okf-global` 并存，新增固定根目录的源码依据、双模式 wrapper 规格、项目缺库明确失败、读写分流、cwd 修复路径、跨盘 / 越界验收与成套回滚。
- 同步更新 `architecture.svg`；保留迁移流程与删除确认要求。
