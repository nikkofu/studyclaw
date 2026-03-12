# StudyClaw 下一阶段正式派单文档

归档说明：

- 本文档已不再是当前正式派单入口。
- 当前唯一正式派单入口是 `docs/14_NEXT_PHASE_DISPATCH.md`。
- 本文件保留为历史快照，避免覆盖已经完成的阶段记录。
- 本文件中的 `v0.2.0` 目标版本属于历史阶段，不代表当前发布基线。

本文档是 `SC-05-INTEGRATION` 完成之后的最新派单入口。你后续给不同 Codex 派任务时，统一以本文件为准。

适用状态：

- 日期：`2026-03-09`
- 当前阶段：`SC-01` 到 `SC-05` 本轮任务已全部完成并完成 GitHub 同步
- 当前项目版本：`v0.1.1`
- 当前目标：推进 `v0.2.0` 演示稳定版

配套参考：

- `docs/08_PARALLEL_WORKSTREAMS.md`
- `docs/09_CODEX_TERMINAL_COMMANDS.md`
- `docs/06_RUNBOOK.md`
- `docs/07_SECURITY.md`

## 1. 当前阶段已完成内容

### `SC-01-GO-API`

- API 错误结构统一
- 错误码与 `details` 结构收口
- 状态更新异常路径测试增强
- 新增 API 错误契约文档

### `SC-02-GO-AGENT`

- homework parser 规则增强
- weekly insight 输出归一化增强
- agent 相关回归测试增强

### `SC-03-FLUTTER-PAD`

- Pad 端收口成清晰分层
- 状态同步测试增强
- Chrome/Web 构建通过

### `SC-04-PARENT-WEB`

- 风险高亮与三段式操作流程完成
- 支持按日期 parse / confirm
- 自动化测试已接入

### `SC-05-INTEGRATION`

- 私有运行时密钥方案已落地
- `preflight_local_env.sh` 已落地
- `smoke_local_stack.sh` 已落地
- `README` / `Runbook` / `Roadmap` / `Changelog` 已同步

## 2. 下一阶段总目标

下一阶段不再以“补新功能”为主，而是以 `v0.2.0` 演示稳定版为目标，收口成更稳的团队工作流。

总目标：

1. 把联调从“能跑”推进到“可重复验收”
2. 把 API / Agent / Web / Pad 的关键行为固化成回归包
3. 把本地脚本推进到更接近发布前检查
4. 让每个 Codex 的任务都可以单独验收和单独提交

## 3. 各 Codex 任务总表

| Codex | 下阶段主目标 | 优先级 | 交付重点 |
| --- | --- | --- | --- |
| `SC-01-GO-API` | API 冻结与 smoke 样例收口 | P0 | 契约稳定、curl 样例、兼容性 |
| `SC-02-GO-AGENT` | 解析回归包与质量说明 | P0 | 样本集、误报控制、解释清晰 |
| `SC-03-FLUTTER-PAD` | 真实后端联调验收 | P1 | Chrome/iPad/真机清单、错误态完善 |
| `SC-04-PARENT-WEB` | 家长端真实后端验收 | P0 | 按日期链路、失败保留、测试补强 |
| `SC-05-INTEGRATION` | 演示脚本与发布前检查 | P0 | 一键演示入口、release checklist、文档同步 |

## 4. 给每个 Codex 的正式派单

### 4.1 `SC-01-GO-API`

```text
[SC-01-GO-API]
目标：把当前 API 从“本地稳定”推进到“演示期冻结可复用”。

边界：
- apps/api-server/cmd
- apps/api-server/config
- apps/api-server/internal/app
- apps/api-server/internal/interfaces/http
- apps/api-server/internal/modules/taskboard
- apps/api-server/routes

必须完成：
1. 审核并补齐 API_ERROR_CONTRACT.md，确保 Web、Pad、Integration 都能直接引用。
2. 产出一组标准 smoke curl 样例，至少覆盖：
   - /ping
   - /api/v1/tasks 查询
   - /api/v1/tasks/parse
   - /api/v1/tasks/confirm
   - status/item、status/group、status/all
3. 检查是否仍存在不一致错误码、details 字段或成功响应形态。
4. 明确本轮 API 冻结边界，避免下轮 UI 再跟着改字段。

验收：
1. go test ./... 通过
2. 契约文档与 curl 样例可直接被 SC-05 引入
3. 当前对外路由和返回字段稳定

禁止修改：
- apps/api-server/internal/modules/agent
- apps/pad-app
- apps/parent-web
```

### 4.2 `SC-02-GO-AGENT`

```text
[SC-02-GO-AGENT]
目标：把当前 agent 改动沉淀为“可长期回归”的质量基线。

边界：
- apps/api-server/internal/modules/agent
- apps/api-server/internal/platform/llm
- apps/api-server/internal/shared/agentic

必须完成：
1. 把本轮新增解析样本整理成结构化回归集合。
2. 给每类高风险任务补明确说明：
   - 为什么 needs_review
   - 什么情况不应误判
3. 输出一组最小联调样本，供 SC-05 写进 smoke 或演示清单。
4. 补极端输入下 weekly insight 的回归说明和测试补点。

验收：
1. go test ./internal/modules/agent/... ./internal/platform/llm/... ./internal/shared/agentic/... 通过
2. 回归样本能覆盖订正、续做、条件任务、对象范围、误报控制
3. 不改 taskboard 和 HTTP handler

禁止修改：
- apps/api-server/internal/modules/taskboard
- apps/api-server/internal/interfaces/http
- apps/pad-app
- apps/parent-web
```

### 4.3 `SC-03-FLUTTER-PAD`

```text
[SC-03-FLUTTER-PAD]
目标：把 Pad 端从“测试通过”推进到“真实后端可验收”。

边界：
- apps/pad-app

必须完成：
1. 用真实后端完成 Chrome 联调验收，并记录：
   - 任务板加载
   - 单任务勾选
   - 分组勾选
   - 全部完成
   - 404 错误
   - 409 错误
2. 如有必要，继续把 error_code 映射成更清晰的提示文案。
3. 输出一份 Pad 最小联调清单给 SC-05。
4. 明确哪些 Flutter 平台文件应该纳入版本库，哪些不应继续扩散。

验收：
1. flutter analyze 通过
2. flutter test 通过
3. flutter build web --dart-define=API_BASE_URL=http://localhost:8080 通过
4. 真实后端联调清单可复现

禁止修改：
- apps/api-server
- apps/parent-web
```

### 4.4 `SC-04-PARENT-WEB`

```text
[SC-04-PARENT-WEB]
目标：把家长端关键路径从“本地通过”推进到“真实后端稳定可演示”。

边界：
- apps/parent-web/src
- apps/parent-web/package.json
- apps/parent-web/package-lock.json
- apps/parent-web/index.html

必须完成：
1. 用真实后端完成按日期创建链路验收：
   - 指定日期
   - parse
   - 风险审核
   - confirm
   - 查询当天结果
2. 确认失败时草稿和选中项不会丢失。
3. 输出一份 Parent Web 最小联调清单给 SC-05。
4. 若联调暴露出小问题，优先做小修，不为局部问题重构页面。

验收：
1. npm run test 通过
2. npm run build 通过
3. 按日期创建链路能稳定复现

禁止修改：
- apps/api-server
- apps/pad-app
```

### 4.5 `SC-05-INTEGRATION`

```text
[SC-05-INTEGRATION]
目标：把现有 preflight/smoke 再推进一步，变成演示与发布前检查的主入口。

边界：
- docs
- scripts
- README.md
- .env.example
- CHANGELOG.md

必须完成：
1. 新增一键演示脚本或演示说明入口，减少手工逐条输命令。
2. 增加 release checklist 文档，覆盖：
   - preflight
   - smoke
   - Go tests
   - Parent Web test/build
   - Pad analyze/test/build web
   - 密钥检查
   - 版本与 changelog 同步
3. 把 docs/06_RUNBOOK.md 和 README.md 继续收口，避免信息分散。
4. 为下一次 GitHub 同步准备固定的发布前检查流程。

验收：
1. bash scripts/check_no_tracked_runtime_env.sh 通过
2. bash scripts/preflight_local_env.sh 通过
3. bash scripts/smoke_local_stack.sh 通过
4. 新人按 runbook 和 release checklist 能独立完成本地演示

禁止修改：
- apps/api-server
- apps/pad-app
- apps/parent-web
```

## 5. 推荐执行顺序

### 第一批

- `SC-01-GO-API`
- `SC-04-PARENT-WEB`
- `SC-05-INTEGRATION`

原因：

- 这三组最直接影响演示稳定性
- `SC-05` 需要依赖 `SC-01` 和 `SC-04` 给出的联调清单与 smoke 样例

### 第二批

- `SC-02-GO-AGENT`
- `SC-03-FLUTTER-PAD`

原因：

- `SC-02` 以质量回归沉淀为主
- `SC-03` 更适合在演示链路稳定后做真实设备联调和清单固化

## 6. 发任务模板

你后续可以继续用这个最短模板派任务：

```text
[终端名]
目标：
边界：
必须完成：
验收：
禁止修改：
```

## 7. 你后续怎么使用这份文档

后续流程固定成：

1. 打开本文件
2. 找到目标 Codex 章节
3. 复制对应代码块
4. 粘贴到对应 Codex
5. 等它回报后，再进入下一组

本文件就是下一阶段的唯一正式派单入口。
