# StudyClaw 第一阶段交付就绪度审计

本文档回答一个更具体的问题：

在 `2026-03-12` 的当前仓库状态下，StudyClaw 第一阶段目标版本 `v0.2.0` 是否已经达到“功能可交付、文档可交接、仓库可进入 GitHub 正式同步”的标准。

本审计以 `docs/01_PRD.md` 中定义的第一阶段 7 类能力为准，并以真实脚本、真实测试和真实本地联调结果作为证据。

## 1. 审计基线

- 审计日期：`2026-03-12`
- 当前交付版本：`v0.2.0`
- 仓库分支：`main`
- GitHub 同步状态：已完成 scoped release commit / tag / push，`main` 与 `v0.2.0` 已同步到 GitHub

当前结论分两层：

- **产品与运行时结论**：`通过`。三端功能已经闭环，达到第一阶段正式交付标准。
- **仓库同步结论**：`已完成`。GitHub release sync 已完成，当前仓库可以作为下一阶段开发与交接基线。

## 2. 已执行验证

以下验证已在 `2026-03-12` 执行：

- `bash scripts/check_no_tracked_runtime_env.sh`
- `bash scripts/preflight_local_env.sh`
- `go test ./... -count=1`
- `npm test`
- `npm run build`
- `flutter analyze`
- `flutter test`
- `flutter build web --dart-define=API_BASE_URL=http://127.0.0.1:38080`
- `STUDYCLAW_SMOKE_API_BASE_URL=http://127.0.0.1:38080 bash scripts/smoke_local_stack.sh`
- `STUDYCLAW_SMOKE_API_BASE_URL=http://127.0.0.1:38080 STUDYCLAW_PARENT_WEB_URL=http://127.0.0.1:5173 bash scripts/demo_local_stack.sh`

三端联调实况：

- API: `http://127.0.0.1:38080`
- Parent Web: `http://127.0.0.1:5173`
- Pad Web: `http://127.0.0.1:55771`

页面可用性：

- `curl http://127.0.0.1:5173/` 返回 Parent Web HTML
- `curl http://127.0.0.1:55771/` 返回 Pad Web HTML

真实业务链路验证：

- 家长发布：`POST /api/v1/tasks/parse` -> `POST /api/v1/tasks/confirm`
- 孩子同步：`GET /api/v1/tasks` -> `PATCH /api/v1/tasks/status/item`
- 家长反馈：`GET /api/v1/stats/daily` -> `GET /api/v1/stats/monthly`
- 积分闭环：`POST /api/v1/points/ledger` -> `GET /api/v1/points/ledger` -> `GET /api/v1/points/balance`
- 词单闭环：`POST /api/v1/word-lists/parse` -> `POST /api/v1/word-lists`
- 听写会话：`POST /api/v1/dictation-sessions/start` -> `POST /next` -> `POST /replay` -> `GET /api/v1/dictation-sessions`

## 3. 第一阶段主线需求审计

### `R1` 家长发布每日任务
状态：`已闭环`

- 证据：Parent Web 实际接口链路为 `/api/v1/tasks/parse -> /api/v1/tasks/confirm`
- 实测结果：固定日期 `2026-03-12` 成功解析并确认 4 条任务

### `R2` AI 自动解析任务并生成草稿
状态：`已闭环`

- 证据：Go 端 `taskparse` 回归测试通过
- 实测结果：在无可用 LLM 时自动回退到 `rule_fallback`，仍能稳定给出 4 条结构化任务

### `R3` 孩子在 Pad 上完成当天任务并同步
状态：`已闭环`

- 证据：Pad 端通过 `/api/v1/tasks` 与 `/api/v1/tasks/status/*` 读写同一任务板
- 实测结果：`task_id=1` 更新成功后，任务板与统计结果同步反映完成数 `1/4`

### `R4` 家长查看每日完成情况和当天反馈
状态：`已闭环`

- 证据：Parent Web 已接入 `/api/v1/stats/daily` 与 `/api/v1/stats/monthly`
- 实测结果：日统计返回 `completion_rate=0.25`，月统计同步包含 `auto_points=1`、`manual_points=2`

### `R5` 单词清单与 Pad 逐词播放
状态：`已闭环`

- 证据：词单由 `/api/v1/word-lists` 持久化，会话由 `/api/v1/dictation-sessions` 驱动
- 实测结果：`wordlist_000001` 创建成功，`session_000002` 可以 `start -> next -> replay`

### `R6` 积分与奖惩
状态：`已闭环`

- 证据：积分流水和余额全部由 Go 后端提供
- 实测结果：完成一个任务自动生成 `+1`，家长奖励再写入 `+2`，余额汇总为 `3`

### `R7` 日 / 周 / 月图表与 AI 正向鼓励
状态：`已闭环`

- 证据：统计接口全部由后端按日期或月份聚合
- 实测结果：月统计已正确反映任务、积分、词单、听写会话数量

## 4. 已知非阻塞风险

- `flutter build web` 会输出第三方依赖 `flutter_tts` 的 wasm dry-run warning。当前 HTML/Web 构建成功，因此不阻塞 `v0.2.0` 交付，但后续若要把 wasm 作为正式目标，需要单独处理。
- `apps/api-server/.gopath/` 的历史跟踪缓存已在本次 release sync 中作为一次性仓库清洁项处理，并由 `scripts/check_release_scope.sh` 持续约束，避免后续再次把环境缓存带入 GitHub。

## 5. 审计结论

### 功能交付结论

StudyClaw 第一阶段已经消除“前端本地估算是事实源”的关键问题，实现了以 Go 后端为唯一事实源的闭环架构。
**`v0.2.0` 已达到第一阶段正式交付与签收标准。**

### 仓库同步结论

`2026-03-12` 已按 `docs/20_RELEASE_SYNC_PLAYBOOK.md` 完成 scoped release commit、版本标签 `v0.2.0` 和 GitHub push 复核。
当前仓库已满足第一阶段正式签收标准，可以作为下一阶段的开发基线。
