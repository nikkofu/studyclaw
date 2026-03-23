# StudyClaw 第一阶段交付就绪度审计（含 `v0.4.0` 接管治理闸门）

> `2026-03-16` 补充结论：`v0.4.0` 已完成对 `docs/03_ROADMAP.md` 成功标准的收口核对，并在新增家长端背诵 / 朗读语音学习结果摘要复盘后，达到第二阶段正式签收条件。当前仓库正式基线已从 `v0.3.5` 切换到 `v0.4.0`。

本文档回答一个更具体的问题：

在 `2026-03-14` 的当前仓库状态下，StudyClaw 当时目标版本 `v0.3.5` 是否已经达到“功能可交付、文档可交接、仓库可进入 GitHub 正式同步”的标准。

本审计以 `docs/01_PRD.md` 中定义的第一阶段 7 类能力为准，并以真实脚本、真实测试和真实本地联调结果作为证据。

## 接管治理闸门（`v0.4.0`）

在第一阶段交付结论不回退的前提下，第二阶段所有任务必须同时满足以下治理约束：

1. 核心主题不偏移：统一围绕 `家长准备学习素材 -> 孩子语音学习 -> 系统 grounded 分析 -> 家长复盘干预`
2. 架构边界不越线：任务状态、积分、统计、会话事实继续由确定性服务维护；Agent 仅负责 transcript 归一化与解释增强
3. 执行顺序不逆行：`SC-01 API 契约冻结` -> `SC-03 Pad 主链补齐` + `SC-04 Parent 复盘补齐` -> `SC-05 集成验收收口`，`SC-02 Agent` 并行支撑
4. 文档口径不分叉：`README.md`、`docs/03_ROADMAP.md`、`docs/14_NEXT_PHASE_DISPATCH.md`、`docs/17_DELIVERY_READINESS.md` 保持一致

## 1. 审计基线（`v0.3.5` 历史收口快照）

- 审计日期：`2026-03-14`
- 当前交付版本：`v0.3.5`
- 仓库分支：`main`
- GitHub 同步状态：`v0.1.0`、`v0.2.0`、`v0.3.0`、`v0.3.1`、`v0.3.2`、`v0.3.3`、`v0.3.4` 历史正式版本已同步；`v0.3.5` 为当前正式版本

当前结论分两层：

- **产品与运行时结论**：`通过`。当前主线能力、发布工具链和三端自动化验证已经达到 `v0.3.5` 本地发布标准。
- **仓库同步结论**：`通过`。GitHub release sync 完成后，当前仓库可以把 `v0.3.5` 当作新的远端交付标签。

## 2. 已执行验证

以下验证已在 `2026-03-14` 执行：

- `bash scripts/check_no_tracked_runtime_env.sh`
- `bash scripts/preflight_local_env.sh`
- `bash scripts/check_release_scope.sh`
- `go test ./... -count=1`
- `npm test -- --run`
- `npm run build`
- `flutter analyze`
- `flutter test --no-pub`
- `flutter build web --dart-define=API_BASE_URL=http://127.0.0.1:38080`
- `STUDYCLAW_SMOKE_API_BASE_URL=http://127.0.0.1:38080 bash scripts/smoke_local_stack.sh`
- `STUDYCLAW_SMOKE_API_BASE_URL=http://127.0.0.1:38080 STUDYCLAW_PARENT_WEB_URL=http://127.0.0.1:5173 bash scripts/demo_local_stack.sh`
- `curl http://127.0.0.1:5173/`
- `curl http://127.0.0.1:55771/`

本轮自动化基线：

- API：Go 单测全量通过
- Parent Web：Vitest 与生产构建通过
- Pad：`flutter analyze`、`flutter test`、`flutter build web` 全部通过

本轮补充联调结论：

- `smoke_local_stack.sh` 已在 `API=http://127.0.0.1:38080` 下重新执行并通过
- `demo_local_stack.sh` 已在 `Parent=http://127.0.0.1:5173` 下重新执行并通过
- Parent Web 与 Pad Web 入口页面均已返回有效 HTML，可直接作为三端演示起点

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
- 词单缺失等待态：Pad 将 `word_list_not_found` 转成“等家长补充词单后再来默写”的友好提示，不再直接暴露 `TaskApiException`
- 鼓励语音播报：任务板和语音工作台的成长鼓励支持自动播报、手动重播和自动播报开关；Widget 回归已覆盖

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

- `flutter build web` 会输出第三方依赖 `flutter_tts` 的 wasm dry-run warning。当前 HTML/Web 构建成功，因此不阻塞 `v0.3.5` 交付，但后续若要把 wasm 作为正式目标，需要单独处理。
- 当前未发现该 warning 已经破坏 Pad Web 的 HTML/Web 调试主链；因此在 `v0.4.0` 阶段继续按“已知非阻塞风险”处理即可，不建议为此提前做高风险依赖替换。
- `apps/api-server/.gopath/` 的历史跟踪缓存已在本次 release sync 中作为一次性仓库清洁项处理，并由 `scripts/check_release_scope.sh` 持续约束，避免后续再次把环境缓存带入 GitHub。

## 5. 当前阻塞 / 未完成项台账（`2026-03-23`）

- `P0-文档口径统一（Dispatch/UAT/Release Sync/Readiness）`
  - 状态：`done`
  - Owner：`SC-05-INTEGRATION`
  - 证据：`bash scripts/check_release_scope.sh`（2026-03-23，PASS）
  - 下一步：随 `v0.4.1` 迭代继续维持同一口径。

- `P0-热任务推荐契约与排序（API）`
  - 状态：`done`
  - Owner：`SC-01-GO-API`
  - 证据：
    - `cd apps/api-server && go test ./routes -run 'TestDailyAssignment_LaunchRecommendationContract|TestDailyAssignment_LaunchRecommendation_OmittedWhenNoTasks|TestHotTaskFlagsOff_PayloadUnchanged' -count=1`（PASS）
    - `cd apps/api-server && go test ./... -count=1`（PASS）
  - 下一步：在后续新增推荐策略字段时继续遵循 additive contract。

- `P0-先做推荐（Pad）+ 回退行为`
  - 状态：`done`
  - Owner：`SC-03-FLUTTER-PAD`
  - 证据：
    - `cd apps/pad-app && flutter test --no-pub test/task_board/launch_recommendation_test.dart`（PASS）
    - `cd apps/pad-app && flutter analyze`（PASS）
    - `cd apps/pad-app && flutter test --no-pub test/widget_test.dart -r compact`（PASS）
  - 下一步：后续 UI 迭代保持 flag-off 不展示“先做推荐”。

- `P1-家长端基线稳定性回归`
  - 状态：`done`
  - Owner：`SC-04-PARENT-WEB`
  - 证据：
    - `cd apps/parent-web && npm test -- --run src/App.test.jsx`（PASS）
    - `cd apps/parent-web && npm run build`（PASS）
  - 下一步：随着 `v0.4.1` 功能变更持续补最小回归断言。

## 6. 审计结论

### 功能交付结论

StudyClaw 第一阶段已经消除“前端本地估算是事实源”的关键问题，实现了以 Go 后端为唯一事实源的闭环架构。
`v0.3.5` 在 `v0.3.4` 正式基线上，进一步把 release scope 校验脚本硬化、三端联调复核和发布资产同步纳入正式主链，已经达到本地发布与交接标准。

### 仓库同步结论

`2026-03-14` 已按 `docs/20_RELEASE_SYNC_PLAYBOOK.md` 完成 `v0.3.5` 的 scoped release commit、版本标签和 GitHub push。
当前仓库已具备下一阶段启动前的远端正式基线条件。
