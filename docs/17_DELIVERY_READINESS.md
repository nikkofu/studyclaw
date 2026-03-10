# StudyClaw 第一阶段交付就绪度审计

本文档用于回答一个具体问题：

在 `2026-03-10` 这个仓库状态下，StudyClaw 距离第一阶段目标版本 `v0.2.0` 还差什么，哪些已经闭环，哪些仍会阻塞正式交付验收。

本审计不是未来规划文档，而是交付前核查文档。它以 `docs/01_PRD.md` 中定义的第一阶段 7 类能力为准，并结合真实代码、真实脚本和真实测试结果给出结论。

## 1. 审计基线

- 审计日期：`2026-03-10`
- 当前仓库版本基线：`v0.1.1`
- 目标交付版本：`v0.2.0`
- 当前结论：`可演示，但还不建议宣称已完成第一阶段正式交付`

原因很明确：

- Go 后端的第一阶段接口已经基本具备
- Parent Web 与 Pad 已具备主链路演示能力
- 但“单词清单 / 听写会话 / 积分流水与余额 / 月报统计”仍存在前端本地态或前端聚合态，没有完全收口到统一后端事实源

这意味着：

- `v0.1.1` 已经是一个可联调、可演示的版本
- 但 `v0.2.0` 若要作为第一阶段交付版，仍需先补齐几个端到端一致性缺口

## 2. 已执行验证

以下命令已在 `2026-03-10` 实际执行并通过：

- `bash scripts/check_no_tracked_runtime_env.sh`
- `bash scripts/preflight_local_env.sh`
- `bash scripts/smoke_local_stack.sh`
- `bash scripts/demo_local_stack.sh`
- `GOCACHE=/Users/admin/Documents/WORK/ai/studyclaw/.cache/go-build GOMODCACHE=/Users/admin/Documents/WORK/ai/studyclaw/apps/api-server/.gomodcache GOPROXY=off GOSUMDB=off go test ./...`
- `cd apps/parent-web && npm run test`
- `cd apps/parent-web && npm run build`
- `cd apps/pad-app && flutter analyze`
- `cd apps/pad-app && flutter test`
- `cd apps/pad-app && flutter build web`

验证结论：

- SC-05 的 `preflight / smoke / demo / release` 入口仍然有效
- Go 后端测试通过
- Parent Web 测试和构建通过
- Pad analyze / test / build web 通过

## 3. 第一阶段主线需求审计

以下状态以 `docs/01_PRD.md` 中第一阶段 7 类能力为准。

### `R1` 家长发布每日任务

状态：`已基本闭环`

已具备：

- 家长端已有 `parse -> 审核编辑 -> confirm -> 刷新 tasks` 主链路
- 后端路由已提供任务解析、发布与任务查询能力
- 演示脚本与现有测试已覆盖基础链路

证据：

- `apps/parent-web/src/App.jsx`
- `apps/api-server/internal/interfaces/http/router.go`
- `apps/api-server/routes/api_success_contract_test.go`

交付判断：

- 这一条已经接近第一阶段要求
- 发布链路不是当前的主要交付阻塞点

### `R2` AI 自动解析任务并生成草稿

状态：`已基本闭环`

已具备：

- Go 后端已收口为 `LLM 优先 + 规则兜底`
- 草稿包含风险信息和人工审核前置
- 解析回归测试已存在

证据：

- `apps/api-server/internal/modules/agent/taskparse/service.go`
- `apps/api-server/internal/modules/agent/taskparse/regression_fixture_test.go`
- `apps/parent-web/src/App.jsx`

交付判断：

- 解析链路可进入正式交付收尾阶段
- 当前更大的问题不在“能否拆任务”，而在“拆完后的数据是否真正统一落到同一后端事实源”

### `R3` 孩子在 Pad 上完成当天任务并同步

状态：`已基本闭环`

已具备：

- Pad 默认进入当天任务板
- 支持单任务、分组、学科、全部任务完成同步
- 有刷新、错误提示、重试与友好提示

证据：

- `apps/pad-app/lib/task_board/repository.dart`
- `apps/pad-app/lib/task_board/controller.dart`
- `apps/pad-app/test/widget_test.dart`

交付判断：

- 任务执行板已经具备第一阶段交付基础
- 孩子执行任务这一条不是当前的核心阻塞项

### `R4` 家长查看每日完成情况和当天反馈

状态：`部分闭环`

已具备：

- 家长端可刷新当天任务
- 周趋势已调用真实 `/api/v1/stats/weekly`
- 当天页面已有任务统计与反馈展示

不足：

- 月趋势仍由前端按近 28 天任务板聚合，不是直接消费后端 `/api/v1/stats/monthly`
- 日报摘要仍主要由前端基于当前任务和本地积分记录生成

关键证据：

- 月趋势由前端聚合：`apps/parent-web/src/App.jsx:1890` 附近
- 页面说明明确写着“月趋势使用前端按近 28 天任务板聚合”：`apps/parent-web/src/App.jsx:1146`
- 周趋势确实走真实接口：`apps/parent-web/src/App.jsx:1850`

交付判断：

- 如果只是现场演示，这一条能讲通
- 如果要做正式验收，“统计图表必须由确定性后端数据生成”这一 PRD 要求还没有完全达标

### `R5` 单词清单与 Pad 逐词播放

状态：`当前阻塞项`

后端已具备：

- `/api/v1/word-lists`
- `/api/v1/dictation-sessions/start`
- `/api/v1/dictation-sessions/:session_id`
- `/api/v1/dictation-sessions/:session_id/replay`
- `/api/v1/dictation-sessions/:session_id/next`

关键证据：

- 路由存在：`apps/api-server/internal/interfaces/http/router.go:46`
- 路由测试存在：`apps/api-server/routes/phase_one_api_test.go:248`

但前端现状仍未闭环：

- Parent Web 单词清单仍使用浏览器 `localStorage`
- Parent Web 测试也验证的是“刷新后本地仍存在”
- Pad 端单词播放仍以本地文本框和样例词为主，不是从真实后端单词清单和听写会话拉取

关键证据：

- 本地存储键与 `localStorage` 读写：`apps/parent-web/src/App.jsx:10`、`apps/parent-web/src/App.jsx:37`、`apps/parent-web/src/App.jsx:55`
- Word list state 直接从本地读取：`apps/parent-web/src/App.jsx:1520`
- 创建单词清单走本地 state，不调用后端：`apps/parent-web/src/App.jsx:2035`
- 测试验证本地持久化：`apps/parent-web/src/App.test.jsx:323`
- Pad 初始化加载样例文本：`apps/pad-app/lib/task_board/page.dart:72`
- Pad 切换语言后重新填样例文本：`apps/pad-app/lib/task_board/page.dart:256`
- 播放控制器只消费原始文本，不访问后端会话：`apps/pad-app/lib/word_playback/controller.dart:106`

交付判断：

- 这一条是第一阶段正式交付的硬阻塞项
- 必须把“家长创建清单 -> 后端保存 -> Pad 拉取 -> 开始会话 -> 重播 / 下一个”完整打通

### `R6` 积分与奖惩

状态：`当前阻塞项`

后端已具备：

- `POST /api/v1/points/update`
- `POST /api/v1/points/ledger`
- `GET /api/v1/points/ledger`
- `GET /api/v1/points/balance`

关键证据：

- 路由存在：`apps/api-server/internal/interfaces/http/router.go:42`
- 积分流水与余额测试已存在：`apps/api-server/routes/phase_one_api_test.go:198`

但前端现状仍未闭环：

- Parent Web 提交积分仍主要调用 `/api/v1/points/update`
- 最近积分明细仍保存在本地浏览器
- Pad 的“今日积分”仍是按已完成任务数直接估算，不是后端权威余额或当日积分变化

关键证据：

- 本地积分流水 state：`apps/parent-web/src/App.jsx:1510`
- 页面说明写明“本地保留最近手工操作记录”：`apps/parent-web/src/App.jsx:1181`
- 提交成功后把积分记录插入本地数组：`apps/parent-web/src/App.jsx:1975`
- 联调说明再次写明最近明细暂存在浏览器：`apps/parent-web/src/App.jsx:2382`
- Pad 今日积分按完成任务数估算：`apps/pad-app/lib/task_board/page.dart:287`
- `estimatedPoints = completed * 2`：`apps/pad-app/lib/task_board/page.dart:654`

交付判断：

- 这一条也是正式交付阻塞项
- 必须让家长端和 Pad 端共同消费后端积分流水与余额，避免双端出现不同口径

### `R7` 日 / 周 / 月图表与 AI 正向鼓励

状态：`部分闭环，接近阻塞`

已具备：

- 周趋势已经调用真实 `/api/v1/stats/weekly`
- 后端已经提供 `/stats/daily`、`/stats/weekly`、`/stats/monthly`
- 周度 Agent 分析链路已经存在

不足：

- Parent Web 月视图仍是前端聚合
- Pad 侧当前只明确提供“今日简报”和“本周鼓励”，没有看到明确的月反馈入口
- Pad 的日鼓励文案主要是本地模板，不是基于后端统计结果生成

关键证据：

- 后端日 / 周 / 月统计路由：`apps/api-server/internal/interfaces/http/router.go:52`
- 后端月统计测试：`apps/api-server/routes/phase_one_api_test.go:320`
- Parent Web 月视图前端聚合：`apps/parent-web/src/App.jsx:1890`
- Pad 只显式提供今日简报与本周鼓励：`apps/pad-app/lib/task_board/page.dart:470`
- Pad 本地简报估算与静态鼓励：`apps/pad-app/lib/task_board/page.dart:648`

交付判断：

- 周报部分已经可用
- 但若要满足 PRD 中“日 / 周 / 月数据可视化，并由 AI 输出积极、正向、可读的分析”，仍需把月视图和孩子端反馈继续收口

## 4. 当前结论

### 可以认定已经做成的部分

- Go 后端已经具备第一阶段大部分领域接口
- 家长发布作业主链路可用
- Pad 当天任务执行与状态同步可用
- 本地安全配置分离方案已建立
- 演示与发布前脚本入口已经形成固定流程

### 还不能直接宣称“第一阶段已完全交付”的原因

存在 3 个核心验收缺口：

1. 单词清单与听写播放尚未真正改成“后端事实源 + 双端统一消费”
2. 积分流水、余额、孩子端积分展示尚未完全统一到后端
3. 月报和部分日反馈仍带有前端聚合 / 前端估算性质

因此当前更准确的表述应是：

- `v0.1.1`：联调基线版，已可演示
- `v0.2.0`：尚未达到正式签收条件

## 5. 交付前必须完成的动作

以下动作完成后，再考虑把版本抬到 `v0.2.0`。

### `P0` 阻塞项

1. Parent Web 改为真实接入 `/api/v1/word-lists`
2. Pad 改为真实接入 `word-list + dictation-session`
3. Parent Web 改为真实接入 `/api/v1/points/ledger` 与 `/api/v1/points/balance`
4. Pad 改为展示后端权威积分，而不是 `completed * 2` 估算
5. Parent Web 月视图改为真实接入 `/api/v1/stats/monthly`

### `P1` 交付稳定化项

1. 把日报口径继续向后端收口，减少前端自行计算
2. 补一轮家长端与 Pad 端的一致性联调用例
3. 把第一阶段演示脚本升级为更接近签收脚本的固定入口

## 6. 下一轮多 Codex 派单建议

### `SC-01-GO-API`

目标：

- 冻结第一阶段 `stats / points / words / dictation` 契约，补清楚返回字段、错误码与示例

下一步：

- 确认 `/stats/daily`、`/stats/monthly`、`/points/ledger`、`/points/balance`、`/word-lists`、`/dictation-sessions` 的前端消费字段
- 补接口契约文档或 contract test，避免前端继续本地猜字段

### `SC-02-GO-AGENT`

目标：

- 把第一阶段 AI 输出继续收口到“只解释统计，不改统计”的模式

下一步：

- 核对日 / 周 / 月鼓励文案生成边界
- 补 daily / monthly 正向反馈测试样本
- 确保文案不输出负向批评式内容

### `SC-03-FLUTTER-PAD`

目标：

- 让 Pad 端完全改为消费真实后端的单词、听写和积分数据

下一步：

- 接入 `GET /api/v1/word-lists`
- 接入 `POST /api/v1/dictation-sessions/start`
- 接入 `GET /api/v1/dictation-sessions/:session_id`
- 接入 `POST /api/v1/dictation-sessions/:session_id/replay`
- 接入 `POST /api/v1/dictation-sessions/:session_id/next`
- 接入后端积分与统计，不再使用本地估算积分

### `SC-04-PARENT-WEB`

目标：

- 让家长端从“演示可用”升级为“交付可验收”

下一步：

- 用真实 `/api/v1/word-lists` 替换本地 word list
- 用真实 `/api/v1/points/ledger`、`/api/v1/points/balance` 替换本地积分记录
- 用真实 `/api/v1/stats/monthly` 替换前端 28 天聚合
- 保留当前成功的 parse / review / confirm 交互，不要把已稳定链路再打散

### `SC-05-INTEGRATION`

目标：

- 把第一阶段验收入口从“可演示”推进到“可签收”

下一步：

- 在 `docs/13_RELEASE_CHECKLIST.md` 中新增第一阶段阻塞项清零检查
- 在 `docs/16_FIRST_PHASE_DEMO_CHECKLIST.md` 中新增“后端事实源一致性”检查点
- 等 SC-03 / SC-04 收口后，补一轮端到端演示复核

## 7. 推荐的版本策略

当前不建议直接把仓库版本号上调到 `v0.2.0`。

更稳妥的方式是：

1. 先保持当前版本为 `v0.1.1` 联调基线
2. 先关闭本文档列出的 `R5 / R6 / R7` 关键缺口
3. 所有缺口关闭后，再执行一次完整 `preflight -> smoke -> demo -> tests/build -> release checklist`
4. 验证全部通过后，再把版本提升到 `v0.2.0`

## 8. 一句话结论

主线没有跑偏，真正缺的不是“再造一套新架构”，而是把已经存在的 Go 第一阶段接口真正接到 Parent Web 和 Pad 上，消除本地态、估算态和前端聚合态。
