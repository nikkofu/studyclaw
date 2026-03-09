# StudyClaw 下阶段 Codex 派单文档

本文档是当前这一轮 GitHub 同步后的正式派单入口。你后续给不同 Codex 分配任务时，直接从本文件复制对应区块即可。

适用时间点：

- 日期：`2026-03-09`
- 当前主分支：`main`
- 当前阶段：Go API、Go Agent、Flutter Pad、Parent Web 的本轮功能已完成；Integration 完成了安全与文档基础，但 `preflight/smoke` 仍未落地

相关参考：

- `docs/08_PARALLEL_WORKSTREAMS.md`
- `docs/09_CODEX_TERMINAL_COMMANDS.md`
- `docs/10_NEXT_PHASE_CODEX_TASKS.md`
- `docs/06_RUNBOOK.md`
- `docs/07_SECURITY.md`

## 1. 当前阶段结论

这一轮的有效产出主要包括：

1. `SC-01-GO-API`
   - 统一了 API 错误结构
   - 补了状态更新与错误路径测试
   - 增加了 API 错误契约文档
2. `SC-02-GO-AGENT`
   - 强化了解析规则
   - 补了更多 parser / weekly insight 测试
3. `SC-03-FLUTTER-PAD`
   - 把 Pad 端收口成 `app + page + controller + repository + api_client`
   - 已通过 analyze / test / web build
4. `SC-04-PARENT-WEB`
   - 增加了风险高亮、流程状态、失败重试
   - 增加了最小自动化测试
   - 支持按日期把任务解析并确认创建到某一天
5. `SC-05-INTEGRATION`
   - 安全脚本可用
   - 文档化分工已完成
   - 但还没有完成 `preflight` 和 `smoke` 脚本化

## 2. 下阶段目标

下一个阶段不是继续分散加功能，而是进入“联调固化 + 交付收口”。

总目标：

1. 把联调从“靠人工记忆命令”推进到“可脚本执行”
2. 把 API / Web / Pad / Agent 的当前行为固化为稳定契约和回归样本
3. 让后续每个 Codex 的工作都更短、更明确、更可验收

## 3. 派单总表

| Codex | 下阶段主目标 | 优先级 | 说明 |
| --- | --- | --- | --- |
| `SC-01-GO-API` | 固化错误契约和联调接口样例 | P0 | 支撑 Web/Pad/Integration |
| `SC-02-GO-AGENT` | 收口解析回归样本集 | P0 | 支撑长期稳定性 |
| `SC-03-FLUTTER-PAD` | 做真实后端联调验收与错误码展示 | P1 | 重点是同步链路 |
| `SC-04-PARENT-WEB` | 做按日期创建的联调验收与测试补强 | P0 | 重点是家长关键路径 |
| `SC-05-INTEGRATION` | 补 preflight/smoke 脚本并接入 runbook | P0 | 这是当前最大缺口 |

## 4. 给每个 Codex 的派单内容

### 4.1 `SC-01-GO-API`

```text
[SC-01-GO-API]
目标：把当前 API 契约从“代码里稳定”升级成“联调时稳定可引用”。

边界：
- apps/api-server/cmd
- apps/api-server/config
- apps/api-server/internal/app
- apps/api-server/internal/interfaces/http
- apps/api-server/internal/modules/taskboard
- apps/api-server/routes

必须完成：
1. 检查并补齐 API 错误契约文档，让 Web、Pad、Integration 都能直接引用。
2. 补一组最小联调样例：
   - 创建单条任务
   - parse 不自动写入
   - confirm 写入
   - 查询某一天任务
   - 单任务/分组/全部状态更新
3. 如果发现仍有不一致错误码或 details 结构，继续收口。
4. 给 SC-05 一份可直接放进 smoke 流程的 curl 样例。

验收：
1. go test ./... 通过
2. API 错误契约文档可直接被前端和集成组引用
3. smoke 所需样例路径、参数、预期结果明确

禁止修改：
- apps/api-server/internal/modules/agent
- apps/pad-app
- apps/parent-web
```

建议验证命令：

```bash
cd /Users/admin/Documents/WORK/ai/studyclaw/apps/api-server
GOCACHE="$(pwd)/../../.cache/go-build" GOMODCACHE=/Users/admin/Documents/WORK/go/pkg/mod GOPROXY=off GOSUMDB=off go test ./...
```

### 4.2 `SC-02-GO-AGENT`

```text
[SC-02-GO-AGENT]
目标：把当前 agent 改动从“本轮增强”收口成“稳定回归包”。

边界：
- apps/api-server/internal/modules/agent
- apps/api-server/internal/platform/llm
- apps/api-server/internal/shared/agentic

必须完成：
1. 把本轮新增解析样本整理成稳定回归测试集合。
2. 明确每类高风险任务为何 needs_review，避免后续团队只能猜规则。
3. 产出一组最小 agent 联调输入样本，交给 SC-05 收进 smoke 或手工验收清单。
4. 补极端输入下的 weekly insight 回归说明。

验收：
1. go test ./internal/modules/agent/... ./internal/platform/llm/... ./internal/shared/agentic/... 通过
2. 样本集对订正、续做、条件任务、对象范围、误报控制都有覆盖
3. 不改 taskboard 和 HTTP handler

禁止修改：
- apps/api-server/internal/modules/taskboard
- apps/api-server/internal/interfaces/http
- apps/pad-app
- apps/parent-web
```

建议验证命令：

```bash
cd /Users/admin/Documents/WORK/ai/studyclaw/apps/api-server
GOCACHE="$(pwd)/../../.cache/go-build" GOMODCACHE=/Users/admin/Documents/WORK/go/pkg/mod GOPROXY=off GOSUMDB=off go test ./internal/modules/agent/... ./internal/platform/llm/... ./internal/shared/agentic/...
```

### 4.3 `SC-03-FLUTTER-PAD`

```text
[SC-03-FLUTTER-PAD]
目标：用真实后端做 Pad 端联调验收，把“能 build”推进到“能稳定同步”。

边界：
- apps/pad-app

必须完成：
1. 用真实后端跑 Chrome 联调，覆盖：
   - 加载任务板
   - 单任务勾选
   - 分组勾选
   - 全部完成
   - 404 错误提示
   - 409 错误提示
2. 把后端 error_code 映射成更明确的用户提示文案。
3. 给 SC-05 一份 Pad 联调最小检查清单。
4. 明确 apps/pad-app 新增文件哪些必须纳入版本库，哪些不该继续扩散。

验收：
1. flutter analyze 通过
2. flutter test 通过
3. flutter build web --dart-define=API_BASE_URL=http://localhost:8080 通过
4. 真实后端联调清单可复现

禁止修改：
- apps/api-server
- apps/parent-web
```

建议验证命令：

```bash
cd /Users/admin/Documents/WORK/ai/studyclaw/apps/pad-app
flutter analyze
flutter test
flutter build web --dart-define=API_BASE_URL=http://localhost:8080
```

### 4.4 `SC-04-PARENT-WEB`

```text
[SC-04-PARENT-WEB]
目标：把家长端从“本地可用”推进到“联调可验收”，重点盯按日期创建和失败保留草稿路径。

边界：
- apps/parent-web/src
- apps/parent-web/package.json
- apps/parent-web/package-lock.json
- apps/parent-web/index.html

必须完成：
1. 用真实后端跑一遍按日期创建链路：
   - 指定日期
   - parse
   - 风险审核
   - confirm
   - 查询当天任务
2. 保持失败时草稿与选中项不丢失。
3. 给 SC-05 一份 Parent Web 最小联调清单。
4. 若测试仍不够，再补最小自动化覆盖，但不要为测试重构整个页面。

验收：
1. npm run test 通过
2. npm run build 通过
3. 按某一天创建任务的链路可复现

禁止修改：
- apps/api-server
- apps/pad-app
```

建议验证命令：

```bash
cd /Users/admin/Documents/WORK/ai/studyclaw/apps/parent-web
npm run test
npm run build
```

### 4.5 `SC-05-INTEGRATION`

```text
[SC-05-INTEGRATION]
目标：补齐当前最大缺口，把联调流程脚本化，而不是继续只写文档。

边界：
- docs
- scripts
- README.md
- .env.example

必须完成：
1. 新增 scripts/preflight_local_env.sh
   至少检查：
   - Go
   - Node / npm
   - Flutter
   - Docker
   - 私有 runtime.env 是否存在
   - 关键目录是否齐全
2. 新增 scripts/smoke_local_stack.sh
   至少覆盖：
   - Go 后端健康检查
   - Parent Web 构建
   - Pad Web 构建
   - 可选：最小 curl API 验证
3. 更新 docs/06_RUNBOOK.md 和 README.md
   - 把 preflight/smoke 纳入最短联调路径
   - 明确真实密钥仍然只放 ~/.config/studyclaw/runtime.env
4. 不改业务源码目录。

验收：
1. bash scripts/check_no_tracked_runtime_env.sh 通过
2. bash scripts/preflight_local_env.sh 可执行
3. bash scripts/smoke_local_stack.sh 可执行
4. 新人可直接按 docs/06_RUNBOOK.md 走联调

禁止修改：
- apps/api-server
- apps/pad-app
- apps/parent-web
```

建议验证命令：

```bash
cd /Users/admin/Documents/WORK/ai/studyclaw
bash scripts/check_no_tracked_runtime_env.sh
bash scripts/preflight_local_env.sh
bash scripts/smoke_local_stack.sh
```

## 5. 推荐执行顺序

### 第一批

- `SC-05-INTEGRATION`
- `SC-01-GO-API`
- `SC-04-PARENT-WEB`

原因：

- `SC-05` 是当前最大缺口
- `SC-01` 和 `SC-04` 能尽快给 `smoke` 和联调清单提供稳定输入

### 第二批

- `SC-03-FLUTTER-PAD`
- `SC-02-GO-AGENT`

原因：

- `SC-03` 适合在 smoke 入口更稳定后补真实联调清单
- `SC-02` 这轮以回归样本收口为主，节奏可稍后，但仍然重要

## 6. 发任务时的最短模板

如果你后续想自己简化任务派发，用这个模板：

```text
[终端名]
目标：
边界：
必须完成：
验收：
禁止修改：
```

## 7. 你后续怎么用

你的后续操作方式可以固定成：

1. 打开本文件
2. 找到某个 Codex 对应章节
3. 复制代码块
4. 粘贴到对应 Codex
5. 等该 Codex 回报后，再回到本文件继续派下一个

这份文档就是下一阶段的正式派单源文件。
