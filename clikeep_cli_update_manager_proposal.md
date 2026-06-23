# clikeep：常用 CLI 更新管家方案文档

> 文档状态：方案草案  
> 更新时间：2026-06-23  
> 推荐 CLI 名称：`clikeep`  
> 推荐配套 Skill 名称：`cli-update-manager`

---

## 1. 一句话结论

`clikeep` 是一个**本地优先的常用 CLI 更新管家**，用于发现开发者高频使用的 CLI、维护每个 CLI 的更新 Profile，并通过交互式、多选、可预览、可回放的方式批量执行这些 CLI 自己的更新命令。

它不是新的包管理器，也不是 Topgrade 的替代品，而是一个位于「用户常用 CLI」与「各类安装来源 / 自更新命令」之间的个人更新编排层。

---

## 2. 背景与问题

开发者本机通常会安装大量 CLI 工具，例如：

```bash
lark-cli update
fornaxcli update
bytedcli update
```

这些 CLI 可能来自不同来源：

- Homebrew
- npm global
- pipx
- Cargo install
- Go install
- GitHub Release binary
- 公司内部分发系统
- 直接下载的单文件 binary
- 某个工具自身的 `update` / `self-update` 子命令

现有包管理器主要从「安装来源」视角解决问题：

```text
brew 管 brew 装的
npm 管 npm 全局包
pipx 管 pipx app
cargo-update 管 cargo install 的 executable
mise/aqua 管声明式工具版本
bin/binenv 管它们自己追踪的 binary
Topgrade 编排多个包管理器
```

但用户的真实心智通常不是这样。用户更关心：

```text
我最近经常用哪些 CLI？
它们现在在哪里？
它们当前版本是什么？
它们应该怎么更新？
我这次要不要一起更新？
上次更新有没有失败？
```

因此，`clikeep` 的核心切入点不是「统一所有包管理器」，而是「维护我真正关心的 CLI 的更新入口」。

---

## 3. 核心判断：优先做 Mode A，而不是 Mode B

本方案将 CLI 更新分为两种模式。

| 模式 | 定位 | 示例 | 优先级 |
|---|---|---|---|
| Mode A：高频 CLI 自更新模式 | 围绕用户高频使用的 CLI，维护每个 CLI 的更新命令，并支持交互式批量执行 | `lark-cli update`、`bytedcli update`、`fornaxcli update` | 最高 |
| Mode B：包管理器批量升级模式 | 调用各生态已有更新命令 | `brew upgrade`、`npm update -g`、`pipx upgrade-all`、`cargo install-update -a` | 次优先级，作为 Adapter 补充 |

结论：**MVP 应优先完成 Mode A。**

原因：

1. Mode A 更贴近真实痛点：不想反复记忆和手动执行多个内部 CLI / 高频 CLI 的更新命令。
2. Mode B 已经有大量现成工具和生态内命令。
3. 内部 CLI、私有 CLI、直接下载 binary 往往无法被包管理器统一追踪，但它们通常有自己的自更新命令。
4. 全自动检测所有 bin 的安装来源和更新方式并不可靠，容易误判或误更新。

---

## 4. 现有工具调研与差异分析

### 4.1 Topgrade

Topgrade 的定位是「Upgrade all the things」。其 README 说明：保持系统更新通常需要调用多个包管理器，Topgrade 会检测用户使用了哪些工具并运行相应更新命令；同时它支持在配置文件中定义自定义命令。参考资料：[Topgrade README][topgrade-readme]、[Topgrade config example][topgrade-config]。

它很适合作为系统级升级编排器，也可以通过 `[commands]` 配置内部 CLI 的更新命令。

示例：

```toml
[commands]
"lark-cli" = "lark-cli update"
"bytedcli" = "bytedcli update"
"fornaxcli" = "fornaxcli update"
```

但 Topgrade 的主抽象仍然是：

```text
更新整个系统 / 开发环境
```

而不是：

```text
发现我高频使用的 CLI，维护每个 CLI 的 path/version/update profile/status/log
```

因此，Topgrade 可以作为底层 runner 或竞品参考，但不是完全等价方案。

### 4.2 bin

`marcosnils/bin` 是一个轻量、跨平台的 binary manager，用于下载、安装、管理 binary，并且不要求 root 权限。它支持 GitHub Releases、GitLab Releases、Codeberg Releases、Docker Images、HashiCorp Releases、Go Install 等来源；也支持 `bin update [binary...]` 更新全部或指定 binary。参考资料：[bin README][bin-readme]。

它适合管理由 `bin` 安装和追踪的二进制工具。

但 `clikeep` 要解决的是：

```text
我机器上已经存在一批内部 CLI / 高频 CLI，
它们不一定由同一个 binary manager 安装，
我不想迁移安装来源，
只想统一维护它们的自更新命令和状态。
```

因此，`bin` 是相关工具，但不是完全替代。

### 4.3 mise / aqua / binenv

mise 的 `mise upgrade` 支持交互式多选、dry-run、exclude、jobs 等参数，交互体验值得借鉴。参考资料：[mise upgrade docs][mise-upgrade]。

aqua 是声明式 CLI Version Manager，强调项目 / 团队 / CI 中统一工具版本、Lazy Install、Registry、Renovate 持续更新。参考资料：[aqua README][aqua-readme]。

binenv 用于管理 kubectl、helm、terraform 等常见 DevOps binaries，并支持 `upgrade` 将所有已安装 distributions 升级到已知最新版本。参考资料：[binenv README][binenv-readme]。

这些工具更偏：

```text
工具版本管理
团队工具链声明
被该工具追踪的 binary 更新
```

而 `clikeep` 更偏：

```text
用户本机高频 CLI 的自更新 Profile 管理
```

### 4.4 各生态内置升级能力

很多生态已有自己的升级命令：

| 生态 | 已有能力 | 参考 |
|---|---|---|
| Homebrew | `brew upgrade` 升级 outdated 且未 pinned 的 formula/cask，也可指定包名 | [Homebrew Manpage][homebrew-manpage] |
| npm | `npm update -g` 对全局安装且 outdated 的包执行 update | [npm update docs][npm-update] |
| pipx | `pipx upgrade-all` 会对每个包运行类似 `pip install -U <pkgname>` 的逻辑 | [pipx docs][pipx-docs] |
| Cargo | `cargo-update` 提供 `cargo install-update -a` 检查并更新所有通过 Cargo 安装的 executable packages | [cargo-update README][cargo-update] |
| Go | `gup update` 可并行更新 `$GOBIN` 下的 Go-installed binaries，`gup check` 可检查状态 | [gup README][gup-readme] |
| GitHub CLI Extension | `gh extension upgrade --all` 支持升级所有 gh extensions，并支持 `--dry-run` | [gh extension upgrade docs][gh-ext] |

这说明 Mode B 不是空白市场。`clikeep` 不应该一开始正面替代这些能力，而应该优先聚焦 Mode A，并在后续通过 Adapter 调用这些生态已有命令。

---

## 5. 产品定位

### 5.1 英文定位

```text
clikeep is a local-first update manager for frequently used CLI tools.
```

或者：

```text
clikeep keeps your everyday CLIs discoverable, updatable, and under control.
```

### 5.2 中文定位

```text
clikeep 是一个本地优先的常用 CLI 更新管家，用于发现高频 CLI、维护更新配置，并安全地交互式批量执行自更新命令。
```

### 5.3 不是什么

`clikeep` 不应该定位为：

- 新的 package manager
- 新的 binary manager
- 新的 version manager
- Topgrade replacement
- 自动更新所有本机命令的黑盒工具

### 5.4 是什么

`clikeep` 应定位为：

- Personal CLI update profile manager
- Frequent CLI update orchestrator
- Local-first CLI upkeep layer
- 面向内部 CLI / 高频 CLI 的更新控制台

---

## 6. 命名方案

### 6.1 CLI 工具名：`clikeep`

推荐：

```text
repo: clikeep
bin:  clikeep
```

理由：

- `cli` 表达对象是命令行工具。
- `keep` 表达长期维护、保持新鲜、保持可控。
- 比 `cliup` 更不局限于一次 update 行为。
- 适合承载 `scan`、`profile`、`status`、`doctor`、`history`、`update` 等长期管理能力。

命令体验：

```bash
clikeep scan
clikeep list
clikeep add lark-cli --update "lark-cli update"
clikeep up -i
clikeep status
clikeep doctor
```

### 6.2 配套 Skill 名：`cli-update-manager`

推荐：

```text
skills/cli-update-manager/SKILL.md
```

理由：

- Skill 名称应描述性强，便于 Agent 根据用户意图自动选择。
- `clikeep` 适合作为 CLI 品牌名，但 Skill 名最好直接表达能力。
- 用户说「帮我检查常用 CLI 是否需要更新」时，`cli-update-manager` 更容易被路由命中。

Skill 描述：

```text
Use this skill to discover frequently used local CLI tools, maintain update profiles, and safely batch-run confirmed self-update commands through clikeep.
```

---

## 7. 核心用户场景

### 7.1 手动维护内部 CLI 更新入口

用户已知一批内部 CLI：

```bash
lark-cli update
bytedcli update
fornaxcli update
```

用户希望一次性纳入管理：

```bash
clikeep add lark-cli --update "lark-cli update" --version "lark-cli --version"
clikeep add bytedcli --update "bytedcli update" --version "bytedcli --version"
clikeep add fornaxcli --update "fornaxcli update" --version "fornaxcli --version"
```

之后使用：

```bash
clikeep up -i
```

交互式选择并批量执行。

### 7.2 自动发现最近高频 CLI

用户执行：

```bash
clikeep scan --history 30d
```

输出：

```text
Detected frequently used CLIs:

  [1] lark-cli     128 uses   /opt/homebrew/bin/lark-cli
  [2] bytedcli      73 uses   /usr/local/bin/bytedcli
  [3] fornaxcli     41 uses   ~/.local/bin/fornaxcli
  [4] gh            39 uses   /opt/homebrew/bin/gh
  [5] kubectl       35 uses   /opt/homebrew/bin/kubectl
```

然后用户选择哪些纳入 Profile。

### 7.3 交互式批量更新

```bash
clikeep up --interactive
```

输出：

```text
? Select CLIs to update:
  ◉ lark-cli     lark-cli update
  ◉ bytedcli     bytedcli update
  ◉ fornaxcli    fornaxcli update
  ○ gh           brew upgrade gh

Plan:
  lark-cli
    path: /opt/homebrew/bin/lark-cli
    before: 1.0.56
    command: lark-cli update

  bytedcli
    path: /usr/local/bin/bytedcli
    before: 0.8.12
    command: bytedcli update

Continue? yes
```

执行结果：

```text
lark-cli
  ✓ already up to date
  before: 1.0.56
  after:  1.0.56

bytedcli
  ✓ updated
  before: 0.8.12
  after:  0.8.15

fornaxcli
  ✗ failed
  exit code: 1
  log: ~/.local/state/clikeep/runs/2026-06-23/fornaxcli.log
```

### 7.4 Agent + Skill 辅助使用

用户对 Agent 说：

```text
帮我检查最近常用的 CLI，看看哪些可以纳入批量更新。
```

Agent 通过 `cli-update-manager` Skill：

1. 先解释将读取 shell history 的范围。
2. 运行 `clikeep scan --history 30d`。
3. 给出候选 CLI 列表。
4. 让用户确认 update command。
5. 生成或更新 Profile。
6. 执行 `clikeep up --interactive --dry-run`。
7. 用户确认后再执行真实更新。

---

## 8. 产品原则

### 8.1 本地优先

所有 Profile、状态、日志默认存储在本机，不依赖云端。

推荐路径：

```text
~/.config/clikeep/config.toml
~/.local/state/clikeep/state.json
~/.local/state/clikeep/runs/<date>/<tool>.log
~/.cache/clikeep/
```

macOS 下遵循 XDG 风格即可，后续可适配平台原生目录。

### 8.2 用户确认优先

未知 CLI 的更新命令不能自动执行。`clikeep` 可以探测候选命令，但必须由用户确认后写入 Profile。

### 8.3 不默认 sudo

默认禁止 `sudo`。如果某个更新命令需要更高权限，应中断并提示用户手动处理或显式开启。

### 8.4 默认串行执行

很多 CLI 自更新会修改自身 binary、symlink、配置或登录态，因此默认串行执行更安全。

后续可以支持：

```bash
clikeep up --jobs 3
```

但对自更新类 CLI 默认 `jobs = 1`。

### 8.5 不替代包管理器

`clikeep` 可以调用 Homebrew/npm/pipx/cargo/gup/gh extension 等生态能力，但不应重新实现这些包管理器的升级逻辑。

### 8.6 可预览、可回放、可诊断

每次更新应保留：

- 更新前版本
- 更新后版本
- 执行命令
- 退出码
- stdout/stderr 日志
- 开始 / 结束时间
- 是否成功
- 是否 already up to date

---

## 9. 核心抽象：CLI Update Profile

`clikeep` 的核心对象不是 package，也不是 binary，而是：

```text
CLI Update Profile
```

示例：

```toml
[[tools]]
name = "lark-cli"
aliases = ["lark"]
enabled = true
tags = ["internal", "work"]

[tools.detect]
commands = ["command -v lark-cli"]

[tools.version]
command = "lark-cli --version"
parser = "semver-loose"

[tools.update]
command = "lark-cli update"
timeout = "300s"
interactive = true

[tools.post_check]
command = "lark-cli --version"

[tools.policy]
require_confirm = true
allow_sudo = false
run_sequential = true
```

Profile 字段建议：

| 字段 | 说明 |
|---|---|
| `name` | CLI 主命令名 |
| `aliases` | 可能的别名 |
| `enabled` | 是否启用 |
| `tags` | 标签，如 internal、work、oss |
| `detect.commands` | 用于检测 CLI 是否存在 |
| `version.command` | 查询当前版本命令 |
| `version.parser` | 版本解析策略，可先保留 raw output |
| `update.command` | 更新命令 |
| `update.timeout` | 超时时间 |
| `update.interactive` | 是否允许交互 |
| `post_check.command` | 更新后复查版本 |
| `policy.require_confirm` | 是否需要确认 |
| `policy.allow_sudo` | 是否允许 sudo |
| `policy.run_sequential` | 是否必须串行 |

---

## 10. 系统架构

### 10.1 总体架构图

```text
                ┌────────────────────┐
                │ Shell History       │
                │ zsh/bash/fish       │
                └─────────┬──────────┘
                          │
                          ▼
┌──────────────┐   ┌──────────────┐   ┌──────────────────┐
│ PATH Scanner │──▶│ CLI Ranker   │──▶│ Candidate List    │
└──────────────┘   │ freq/recency │   │ high-frequency CLI│
                   └──────┬───────┘   └─────────┬────────┘
                          │                     │
                          ▼                     ▼
                ┌─────────────────┐   ┌──────────────────┐
                │ Source Hints     │   │ User Confirmation │
                │ brew/npm/pipx... │   │ update command    │
                └────────┬────────┘   └─────────┬────────┘
                         │                      │
                         ▼                      ▼
                  ┌────────────────────────────────┐
                  │ CLI Update Profile Store        │
                  │ ~/.config/clikeep/config.toml   │
                  └────────────────┬───────────────┘
                                   │
                                   ▼
                         ┌──────────────────┐
                         │ Update Planner    │
                         │ dry-run / select  │
                         └────────┬─────────┘
                                  │
                                  ▼
                         ┌──────────────────┐
                         │ Update Executor   │
                         │ sequential / logs │
                         └────────┬─────────┘
                                  │
                                  ▼
                         ┌──────────────────┐
                         │ Result Summary    │
                         │ updated/failed    │
                         └──────────────────┘
```

### 10.2 模块拆分

| 模块 | 职责 |
|---|---|
| PATH Scanner | 扫描 `$PATH` 中可执行文件，解析真实路径、symlink、mtime、owner |
| History Scanner | 读取 zsh/bash/fish history，统计命令使用频率和最近使用时间 |
| CLI Ranker | 根据 frequency、recency、是否已存在 Profile 排序候选 CLI |
| Source Hint Detector | 辅助判断 CLI 可能来自 brew/npm/pipx/cargo/go/mise/bin 等来源 |
| Profile Store | 管理 `config.toml` 中的 CLI Update Profile |
| Suggestion Engine | 根据 help/version/source hint 生成候选 update command，但不自动执行 |
| Update Planner | 生成更新计划，支持 dry-run、interactive select、diff view |
| Update Executor | 执行更新命令，处理 timeout、stdout/stderr、exit code、日志 |
| Status Store | 记录每次运行结果、版本变化、失败原因 |
| Doctor | 检查 path 不存在、version 命令失败、update 命令失效等问题 |
| Adapter Layer | 后续接入 brew/npm/pipx/cargo/gup/gh/mise/topgrade 等生态 |

---

## 11. CLI 命令设计

### 11.1 初始化

```bash
clikeep init
```

创建配置目录和默认配置。

### 11.2 扫描候选 CLI

```bash
clikeep scan
clikeep scan --history 30d
clikeep scan --no-history
clikeep scan --path-only
```

### 11.3 添加 Profile

```bash
clikeep add lark-cli --update "lark-cli update"
clikeep add lark-cli --update "lark-cli update" --version "lark-cli --version"
clikeep add lark-cli --tag internal --tag work
```

### 11.4 查看已管理 CLI

```bash
clikeep list
clikeep list --tag internal
clikeep list --json
```

### 11.5 生成更新计划

```bash
clikeep plan
clikeep plan --interactive
clikeep plan --json
```

### 11.6 执行更新

```bash
clikeep up
clikeep up -i
clikeep up lark-cli bytedcli
clikeep up --dry-run
clikeep up --tag internal
```

### 11.7 状态查看

```bash
clikeep status
clikeep status lark-cli
clikeep status --failed
```

### 11.8 诊断

```bash
clikeep doctor
clikeep doctor --fix
```

### 11.9 包管理器补充模式

后续可以引入：

```bash
clikeep pm scan
clikeep pm outdated
clikeep pm up --interactive
```

但建议不要让 `clikeep up` 默认执行全局包管理器升级。

---

## 12. 自动发现策略

### 12.1 PATH 扫描

扫描 `$PATH` 中的可执行文件，记录：

```text
command name
resolved path
real path
is symlink
mtime
owner
source hint
```

限制：PATH 扫描只能告诉我们「有什么命令」，不能可靠告诉我们「怎么更新」。

### 12.2 Shell History 高频分析

读取：

```text
~/.zsh_history
~/.bash_history
~/.config/fish/fish_history
```

统计过去 N 天的 first token：

```text
lark-cli
bytedcli
fornaxcli
gh
kubectl
npm
pnpm
```

排序维度：

- 使用次数
- 最近使用时间
- 是否存在于 PATH
- 是否已有 Profile
- 是否疑似内部 CLI

隐私要求：

- 第一次读取 shell history 必须提示。
- 支持 `--no-history`。
- 不上传 history。
- 默认只解析 first token，不保存完整命令行。
- 对可疑 token 做过滤，如密码、token、URL、路径等。

### 12.3 Source Hint

Source Hint 只作为辅助信息，不作为真相。

可能的判断方式：

| 来源 | Hint |
|---|---|
| Homebrew | path 在 `/opt/homebrew/bin`、`/usr/local/bin`，可尝试 `brew list --formula` / `brew list --cask` |
| npm global | path 指向 npm prefix bin，或 `npm prefix -g` 可关联 |
| pipx | path 位于 `~/.local/bin`，并可通过 `pipx list` 关联 |
| Cargo | path 位于 `~/.cargo/bin` |
| Go | path 位于 `$GOBIN` / `$GOPATH/bin` / `~/go/bin` |
| mise | path 可能位于 mise shim 目录 |
| manual/internal | 无法匹配已知来源，但存在自更新命令 |

---

## 13. Update Command 推断策略

`clikeep` 可以探测候选更新命令，但不能自动执行未知更新命令。

候选命令：

```text
<cmd> update
<cmd> self-update
<cmd> upgrade
<cmd> version
<cmd> --version
<cmd> -v
```

安全探测可以只执行：

```bash
<cmd> --help
<cmd> help
<cmd> --version
```

如果 help 文本中发现 `update`、`upgrade`、`self-update` 子命令，可以生成建议：

```text
lark-cli
  detected update subcommand: update
  suggested command: lark-cli update
  confidence: high
```

但必须经过用户确认：

```bash
clikeep add lark-cli
? Use update command `lark-cli update`? yes
```

---

## 14. 执行模型

### 14.1 默认串行

```text
lark-cli update
bytedcli update
fornaxcli update
```

原因：

- 自更新命令可能修改自身文件。
- 可能修改 symlink。
- 可能依赖登录态。
- 可能有交互输入。
- 并发更新可能导致输出混乱或锁冲突。

### 14.2 失败隔离

一个 CLI 更新失败，不应阻断后续 CLI，除非用户指定：

```bash
clikeep up --fail-fast
```

默认行为：

```text
continue on failure
collect errors
show summary
save logs
```

### 14.3 日志保存

建议保存：

```text
~/.local/state/clikeep/runs/2026-06-23T09-30-00/lark-cli.log
~/.local/state/clikeep/runs/2026-06-23T09-30-00/bytedcli.log
run-summary.json
```

### 14.4 状态文件

示例：

```json
{
  "last_run_id": "2026-06-23T09-30-00",
  "tools": {
    "lark-cli": {
      "last_status": "up_to_date",
      "before_version": "1.0.56",
      "after_version": "1.0.56",
      "last_exit_code": 0,
      "last_run_at": "2026-06-23T09:30:12+09:00"
    },
    "bytedcli": {
      "last_status": "updated",
      "before_version": "0.8.12",
      "after_version": "0.8.15",
      "last_exit_code": 0,
      "last_run_at": "2026-06-23T09:31:20+09:00"
    }
  }
}
```

---

## 15. Package Manager Adapter 设计

Mode B 不作为 MVP 主路径，但可以在后续以 Adapter 形式扩展。

接口示意：

```go
type Adapter interface {
    Name() string
    Available(ctx context.Context) bool
    Discover(ctx context.Context) ([]ManagedTool, error)
    Outdated(ctx context.Context) ([]UpdateCandidate, error)
    Plan(ctx context.Context, selected []ManagedTool) ([]CommandPlan, error)
    Update(ctx context.Context, plan CommandPlan) (UpdateResult, error)
}
```

首批 Adapter 可考虑：

| Adapter | 命令 |
|---|---|
| Homebrew | `brew outdated`、`brew upgrade <formula>` |
| npm | `npm update -g <pkg>` |
| pipx | `pipx upgrade <pkg>` / `pipx upgrade-all` |
| Cargo | `cargo install-update <crate>` / `cargo install-update -a` |
| Go | `gup check` / `gup update <binary>` |
| gh extension | `gh extension upgrade <name>` / `--all` |
| mise | `mise upgrade --interactive` / `mise upgrade <tool>` |
| Topgrade bridge | 调用 Topgrade custom commands 或生成配置片段 |

关键原则：

```text
clikeep 不重新实现包管理器逻辑，只做发现、计划、解释和安全编排。
```

---

## 16. 配套 Skill 设计：cli-update-manager

### 16.1 Skill 定位

`cli-update-manager` 是配合 `clikeep` 使用的 Agent Skill，用于指导 Agent 安全地发现、配置、预览和执行本机 CLI 更新。

Skill 不负责重新实现 `clikeep`。它负责：

- 什么时候应该使用 `clikeep`
- 如何扫描 CLI
- 如何保护 shell history 隐私
- 如何生成 Profile
- 如何要求用户确认 update command
- 如何执行 dry-run
- 如何展示更新计划
- 如何处理失败日志

### 16.2 Skill 名称

```text
cli-update-manager
```

### 16.3 目录结构

```text
skills/
  cli-update-manager/
    SKILL.md
    examples/
      clikeep-config.toml
      internal-cli-profiles.toml
      update-run-output.md
```

### 16.4 SKILL.md 建议结构

```markdown
# CLI Update Manager

Use this skill when the user wants to discover, configure, check, or update frequently used local CLI tools through clikeep.

## Safety Rules

- Never run unknown update commands without explicit user confirmation.
- Prefer `clikeep plan` or `clikeep up --dry-run` before real updates.
- Do not read shell history unless the user has agreed or the task explicitly requires discovering frequently used CLIs.
- Do not upload shell history or full command lines.
- Do not use sudo unless the user explicitly asks and understands the command.
- Show the exact commands that will be executed.

## Standard Workflow

1. Check whether clikeep is installed.
2. Run `clikeep list` to inspect existing profiles.
3. If discovery is requested, run `clikeep scan --history 30d` or `clikeep scan --no-history` depending on user preference.
4. Propose update profiles for selected CLIs.
5. Ask the user to confirm each update command.
6. Run `clikeep plan --interactive` or `clikeep up --dry-run`.
7. After confirmation, run `clikeep up -i` or targeted update.
8. Summarize updated/up-to-date/failed tools and link to logs.
```

---

## 17. MVP 设计

### 17.1 V0.1：手动 Profile + 交互式批量更新

目标：最快验证真实价值。

必须支持：

```bash
clikeep init
clikeep add <cmd> --update "<cmd> update" --version "<cmd> --version"
clikeep list
clikeep up -i
clikeep up --dry-run
clikeep status
clikeep doctor
```

暂不支持：

- 自动扫描 shell history
- 完整包管理器 Adapter
- 自动检测所有 CLI 更新方式
- 远程提醒
- 后台守护进程

V0.1 成功标准：

- 用户能把 `lark-cli`、`bytedcli`、`fornaxcli` 纳入 Profile。
- 用户可以多选并批量执行更新。
- 每个工具有清晰的 path、version、update command、result、log。
- 失败不会阻断其他工具。

### 17.2 V0.2：高频 CLI 发现

新增：

```bash
clikeep scan --history 30d
clikeep suggest
```

能力：

- PATH 扫描
- zsh/bash/fish history first-token 统计
- 高频 CLI 排序
- source hint
- 候选 update command 建议

### 17.3 V0.3：生态 Adapter

新增：

```bash
clikeep pm scan
clikeep pm outdated
clikeep pm up -i
```

首批 Adapter：

- Homebrew
- npm global
- pipx
- Cargo via cargo-update
- Go via gup
- gh extension
- mise

### 17.4 V0.4：Agent Skill 与团队配置

新增：

- `cli-update-manager` Skill
- 示例 Profile Registry
- 团队内部 CLI Profile 模板
- 可导入 / 导出配置
- 生成 Topgrade custom commands 片段

---

## 18. 实现建议

### 18.1 技术栈建议

可选语言：Go 或 Rust。

推荐优先 Go，原因：

- 单文件分发简单。
- 跨平台编译容易。
- 用户熟悉服务端工程时，维护成本低。
- 执行外部命令、解析文件、处理 JSON/TOML 都足够成熟。

Rust 也适合 CLI，但初期如果目标是快速实现并验证产品体验，Go 更轻。

### 18.2 核心包结构示意

```text
cmd/clikeep/
  main.go
internal/
  config/
  profile/
  scanner/
  history/
  sourcehint/
  planner/
  executor/
  status/
  doctor/
  adapter/
    brew/
    npm/
    pipx/
    cargo/
    gup/
    gh/
    mise/
  tui/
```

### 18.3 执行器注意点

执行外部命令时应支持：

- context timeout
- stdout/stderr 分离捕获
- 实时输出到终端
- 同步保存日志
- TTY 检测
- 是否允许 interactive
- exit code 捕获
- 环境变量白名单 / 继承策略

### 18.4 配置与状态分离

配置是用户意图：

```text
~/.config/clikeep/config.toml
```

状态是运行结果：

```text
~/.local/state/clikeep/state.json
~/.local/state/clikeep/runs/
```

不要把大量运行日志写回配置文件。

---

## 19. 风险与约束

### 19.1 自动发现误判

风险：把系统命令、脚本、一次性命令误判为应管理 CLI。

缓解：

- 只生成候选，不自动纳入。
- 按高频 / 近期使用排序。
- 用户确认后才写入 Profile。

### 19.2 自动执行未知 update 命令

风险：未知命令可能产生副作用。

缓解：

- 未确认 Profile 不执行。
- 首次执行强制 dry-run / plan 展示。
- 展示完整命令和路径。

### 19.3 shell history 隐私

风险：history 中可能包含敏感信息。

缓解：

- 默认只解析 first token。
- 不保存完整 history。
- 不上传。
- 提供 `--no-history`。
- 第一次扫描前提示。

### 19.4 包管理器升级副作用

风险：`brew upgrade`、`npm update -g` 等可能升级大量依赖。

缓解：

- Mode B 单独命令空间：`clikeep pm ...`
- 不让 `clikeep up` 默认触发全局包管理器升级。
- 只对用户选择的 package 执行。

### 19.5 自更新过程中替换当前 binary

风险：某个 CLI 更新自身时可能造成命令不可用。

缓解：

- 默认串行。
- 更新后执行 post_check。
- 保留失败日志。
- 不并发更新同一来源 / 同一路径下的工具。

---

## 20. 评估指标

### 20.1 产品有效性指标

| 指标 | 说明 |
|---|---|
| Profile 接受率 | scan/suggest 生成的候选中，被用户确认加入 Profile 的比例 |
| 高频 CLI 覆盖率 | 用户常用 CLI 中，已被 Profile 管理的比例 |
| 批量更新使用频率 | 用户每周 / 每月运行 `clikeep up` 的次数 |
| 节省操作次数 | 一次 clikeep up 替代的手动 update 命令数量 |
| 更新成功率 | selected tools 中成功或 already up-to-date 的比例 |
| 失败可诊断率 | 失败项是否有明确 exit code、日志和建议动作 |

### 20.2 安全指标

| 指标 | 说明 |
|---|---|
| 未确认命令执行次数 | 应为 0 |
| sudo 默认触发次数 | 应为 0 |
| history 原文保存次数 | 应为 0 |
| 包管理器误触发全局升级次数 | 应为 0 |

### 20.3 体验指标

| 指标 | 说明 |
|---|---|
| 首次配置时间 | 从安装到管理第一个 CLI 的时间 |
| 更新计划可读性 | 用户是否能理解即将执行的命令 |
| 日志可找回性 | 失败后是否能快速定位 log |
| Profile 修复成本 | path 变化 / 命令失效后 doctor 能否提示修复 |

---

## 21. 与现有工具的关系

| 工具 | 定位 | 与 clikeep 的关系 |
|---|---|---|
| Topgrade | 系统级批量升级器 | 可作为底层 runner，也可生成 custom commands；不是高频 CLI Profile 管理器 |
| bin | binary manager | 适合管理由 bin 安装和追踪的 binary；clikeep 可对其管理的工具做补充 |
| binenv | DevOps binary version manager | 适合 kubectl/helm/terraform 等工具版本管理 |
| mise | 开发工具版本管理器 | 交互式 upgrade 体验值得借鉴，可作为 Adapter |
| aqua | 声明式 CLI Version Manager | 适合团队 / CI 工具版本声明，不替代个人高频 CLI 自更新 Profile |
| Homebrew/npm/pipx/cargo/gup/gh | 生态内升级工具 | clikeep 后续可通过 Adapter 调用，不重新实现 |

---

## 22. 推荐落地路线

### 阶段 0：先不造完整轮子，验证需求

先用 Topgrade custom commands 管理内部 CLI：

```toml
[commands]
"lark-cli" = "lark-cli update"
"bytedcli" = "bytedcli update"
"fornaxcli" = "fornaxcli update"
```

验证痛点是否只是「少敲几条命令」。

如果 Topgrade 已经够用，则无需继续造工具。

### 阶段 1：实现最薄 clikeep

只做：

```text
manual profile
interactive selection
dry-run
execution logs
status
```

验证是否比 Topgrade custom commands 更好用。

### 阶段 2：加入高频发现

实现 history scan + PATH scan，证明核心差异：

```text
从“我安装了什么”转向“我经常用什么”
```

### 阶段 3：引入 Adapter

只接入最常用且稳定的生态：brew、pipx、cargo-update、gup、gh extension、mise。

### 阶段 4：配套 Skill

让 Agent 能安全使用 clikeep，并沉淀工作流。

---

## 23. 开放问题

1. `clikeep scan` 是否默认读取 shell history，还是默认只扫 PATH？
2. 是否需要单独的 `clikeep check` 来检查更新可用性？很多自更新 CLI 可能没有 check-only 能力。
3. 是否要支持提醒机制，例如超过 14 天未更新时提示？
4. 是否要支持团队共享 Profile registry？
5. 是否支持导出 Topgrade 配置，还是只把 Topgrade 作为参考？
6. 是否将 `clikeep up` 作为主命令，还是更显式地使用 `clikeep update`？
7. 如何处理内部 CLI 的登录态、代理和网络失败？
8. 是否需要支持 MCP / Agent Runtime 形式，让 Agent 通过工具接口调用 clikeep？

---

## 24. 最终建议

建议不要把 `clikeep` 做成「全自动检测所有 CLI 并统一升级」的软件管家。这个方向复杂、危险，并且容易和 Topgrade / 包管理器重叠。

更稳的产品边界是：

```text
clikeep = 高频 CLI 自更新 Profile 管理器
```

MVP 应聚焦：

```text
scan/list/add/profile/interactive update/status/doctor/logs
```

核心差异是：

```text
现有工具多从安装来源出发；
clikeep 从用户高频使用出发。
```

最终组合建议：

```text
CLI：clikeep
Skill：cli-update-manager
```

---

## 参考资料

[topgrade-readme]: https://github.com/topgrade-rs/topgrade
[topgrade-config]: https://github.com/topgrade-rs/topgrade/blob/main/config.example.toml
[bin-readme]: https://github.com/marcosnils/bin
[mise-upgrade]: https://mise.jdx.dev/cli/upgrade.html
[aqua-readme]: https://github.com/aquaproj/aqua
[binenv-readme]: https://github.com/devops-works/binenv
[homebrew-manpage]: https://docs.brew.sh/Manpage
[npm-update]: https://docs.npmjs.com/cli/v10/commands/npm-update/
[pipx-docs]: https://pipx.pypa.io/stable/docs/
[cargo-update]: https://github.com/nabijaczleweli/cargo-update
[gup-readme]: https://github.com/nao1215/gup
[gh-ext]: https://cli.github.com/manual/gh_extension_upgrade
