# StudyClaw 第一阶段交付就绪度审计

本文档回答一个更具体的问题：

在 `2026-03-13` 的当前仓库状态下，StudyClaw 当前目标版本 `v0.3.3` 是否已经达到“功能可交付、文档可交接、仓库可进入 GitHub 正式同步”的标准。

本审计以 `docs/01_PRD.md` 中定义的第一阶段 7 类能力为准，并以真实脚本、真实测试和真实本地联调结果作为证据。

## 1. 审计基线

- 审计日期：`2026-03-13`
- 当前交付版本：`v0.3.3`
- 仓库分支：`main`
- GitHub 同步状态：`v0.1.0`、`v0.2.0`、`v0.3.0`、`v0.3.1`、`v0.3.2` 历史正式版本已同步；`v0.3.3` 为当前正式版本

当前结论分两层：

- **产品与运行时结论**：`通过`。三端功能已经闭环，达到 `v0.3.3` 本地发布标准。
- **仓库同步结论**：`通过`。GitHub release sync 完成后，当前仓库可以把 `v0.3.3` 当作新的远端交付标签。

## 2. 已执行验证

以下验证已在 `2026-03-13` 执行：

- `bash scripts/check_no_tracked_runtime_env.sh`
- `bash scripts/preflight_local_env.sh`
- `go test ./... -count=1`
- `npm test`
- `npm run build`
- `flutter analyze`
- `flutter test --no-pub`
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
- 学习素材自动补全：`/api/v1/tasks/parse` -> 草稿保存 -> `/api/v1/tasks/confirm` 保留 `reference_*` 与 `analysis_mode`
- 家长原文入口：点击“去录入原文”后进入 `原文` 子页面并可直接看到输入框
- 孩子同步：`GET /api/v1/tasks` -> `PATCH /api/v1/tasks/status/item`
- 语音任务完成：`POST /api/v1/voice-commands/resolve` -> Pad 执行任务板按钮动作
- 背诵分析：`POST /api/v1/recitation/analyze` -> 返回标题 / 作者 / 完成度 / 逐句匹配
- 家长反馈：`GET /api/v1/stats/daily` -> `GET /api/v1/stats/monthly`
- 积分闭环：`POST /api/v1/points/ledger` -> `GET /api/v1/points/ledger` -> `GET /api/v1/points/balance`
- 词单闭环：`POST /api/v1/word-lists/parse` -> `POST /api/v1/word-lists`
- 听写会话：`POST /api/v1/dictation-sessions/start` -> `POST /next` -> `POST /replay` -> `GET /api/v1/dictation-sessions`
- 听写语音推进：Pad STT -> `/api/v1/voice-commands/resolve` -> `POST /dictation-sessions/:session_id/next`
- 孩子端鼓励：任务完成即时鼓励、每日鼓励卡片、听写过程鼓励均可正常显示

## 3. 第一阶段主线需求审计

### `R1` 家长发布每日任务
状态：`已闭环`

- 证据：Parent Web 实际接口链路为 `/api/v1/tasks/parse -> /api/v1/tasks/confirm`
- 实测结果：固定日期 `2026-03-12` 成功解析并确认 4 条任务

### `R2` AI 自动解析任务并生成草稿
状态：`已闭环`

- 证据：Go 端 `taskparse` 回归测试通过
- 实测结果：在无可用 LLM 时自动回退到 `rule_fallback`，仍能稳定给出 4 条结构化任务

### `R2A` 背诵 / 朗读任务自动补全学习素材
状态：`已闭环`

- 证据：解析结果、草稿保存和确认发布链路均保留 `reference_title`、`reference_author`、`reference_text`、`hide_reference_from_child`、`analysis_mode`
- 实测结果：老师原文里带有古诗词正文时，系统可直接抽取标题、作者和正文；老师只给标题时，可在配置 LLM 的情况下补全缺口

### `R3` 孩子在 Pad 上完成当天任务并同步
状态：`已闭环`

- 证据：Pad 端通过 `/api/v1/tasks` 与 `/api/v1/tasks/status/*` 读写同一任务板
- 实测结果：`task_id=1` 更新成功后，任务板与统计结果同步反映完成数 `1/4`，Pad 同时显示成长型鼓励

### `R3A` 孩子通过语音完成任务与推进听写
状态：`已闭环`

- 证据：Pad 新增语音助手，统一调用 `/api/v1/voice-commands/resolve`
- 实测结果：任务板场景可识别“数学订正好了”，听写场景可识别“好了 / 下一个 / 重播”

### `R3B` 孩子长段背诵 / 朗读与分析
状态：`已闭环`

- 证据：Pad 学习语音工作台与 `/api/v1/recitation/analyze` 已联通
- 实测结果：在古诗词 noisy transcript 条件下，系统仍能识别标题、输出匹配率和是否建议重背

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
- 实测结果：月统计已正确反映任务、积分、词单、听写会话数量；Pad 会展示后端 `encouragement`

## 4. 已知非阻塞风险

- `flutter build web` 会输出第三方依赖 `flutter_tts` 的 wasm dry-run warning。当前 HTML/Web 构建成功，因此不阻塞 `v0.3.3` 交付，但后续若要把 wasm 作为正式目标，需要单独处理。
- `apps/api-server/.gopath/` 的历史跟踪缓存已在本次 release sync 中作为一次性仓库清洁项处理，并由 `scripts/check_release_scope.sh` 持续约束，避免后续再次把环境缓存带入 GitHub。

## 5. 审计结论

### 功能交付结论

StudyClaw 第一阶段已经消除“前端本地估算是事实源”的关键问题，实现了以 Go 后端为唯一事实源的闭环架构。
`v0.3.3` 在 `v0.3.2` 的热修复基线上，进一步把学习素材自动补全、孩子学习语音工作台和背诵分析纳入正式主链，已经达到本地发布与交接标准。

### 仓库同步结论

`2026-03-13` 已按 `docs/20_RELEASE_SYNC_PLAYBOOK.md` 完成 `v0.3.3` 的 scoped release commit、版本标签和 GitHub push。
当前仓库已具备下一阶段启动前的远端正式基线条件。
