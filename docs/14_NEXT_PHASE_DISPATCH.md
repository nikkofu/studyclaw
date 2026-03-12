# StudyClaw 下一轮正式派单文档

说明：

- 本文档主体记录的是从 `v0.1.1` 向 `v0.2.0` 推进时的正式派单背景。
- 当前可发布增强基线已经推进到 `v0.3.1`；如要继续发布，应以 `README.md`、`docs/13`、`docs/17`、`docs/19`、`docs/20` 为准。

本文档是当前唯一正式派单入口，用于把 StudyClaw 从 `v0.1.1` 联调基线推进到第一阶段可签收版本。

适用状态：

- 日期：`2026-03-10`
- 当前仓库基线：`v0.1.1`
- 当前目标版本：`v0.2.0`
- 当前结论：`可演示，但还未达到第一阶段正式签收`

必须先阅读的主文档：

- `docs/01_PRD.md`
- `docs/03_ROADMAP.md`
- `docs/04_AGENTIC_DESIGN.md`
- `docs/06_RUNBOOK.md`
- `docs/13_RELEASE_CHECKLIST.md`
- `docs/16_FIRST_PHASE_DEMO_CHECKLIST.md`
- `docs/17_DELIVERY_READINESS.md`

## 1. 当前这轮真正要解决的问题

经过交付审计，当前主线不是缺“新架构”，而是缺“统一后端事实源”。

本轮只解决 3 个第一阶段正式签收阻塞项：

1. 单词清单和听写播放仍有本地态，尚未形成 `Parent Web -> Go API -> Pad` 的真实闭环
2. 积分流水、余额、孩子端积分展示尚未完全统一到 Go 后端
3. 月报和部分日反馈仍有前端聚合 / 前端估算，不是完全的后端确定性统计

本轮不做：

- 新增第二阶段功能
- 扩展多模态
- 新增复杂消息推送
- 重做现有任务发布主链路

## 2. 本轮退出标准

只有下面条件都满足，才允许把版本朝 `v0.2.0` 签收推进：

1. Parent Web 单词清单改为真实后端 `word-lists`，不再以 `localStorage` 为事实源
2. Pad 单词播放改为真实后端 `dictation-session`，不再以样例词文本为主路径
3. Parent Web 积分改为真实后端 `points/ledger` 与 `points/balance`
4. Pad 积分展示改为真实后端数据，不再使用本地估算积分
5. Parent Web 月视图改为真实后端 `/api/v1/stats/monthly`
6. 日 / 周 / 月鼓励文案继续遵循“统计值由后端确定，Agent 只解释不改写”
7. `preflight / smoke / demo / go test / web test-build / pad analyze-test-build` 全部通过

## 3. 本轮工作流总表

| Codex | 本轮主目标 | 优先级 | 交付物 |
| --- | --- | --- | --- |
| `SC-01-GO-API` | 冻结统计 / 积分 / 单词 / 听写 API 契约 | P0 | DTO、handler、route test、契约文档 |
| `SC-02-GO-AGENT` | 收口 daily / weekly / monthly 鼓励与解析边界 | P0 | agent pattern 元数据、fixture、回归测试 |
| `SC-03-FLUTTER-PAD` | Pad 改为消费真实单词 / 听写 / 积分数据 | P0 | 页面、repository、测试、演示链路 |
| `SC-04-PARENT-WEB` | 家长端改为消费真实单词 / 积分 / 月报数据 | P0 | UI、API 集成、测试、构建 |
| `SC-05-INTEGRATION` | 继续维护交付文档、演示路径、验收入口 | P0 | readiness / checklist / dispatch / demo 文档同步 |

## 4. 各 Codex 详细任务

### 4.1 `SC-01-GO-API`

目标：

- 把第一阶段真正会被双端消费的接口冻结下来，避免前端继续本地猜字段

唯一边界：

- `apps/api-server/cmd`
- `apps/api-server/config`
- `apps/api-server/internal/app`
- `apps/api-server/internal/interfaces/http`
- `apps/api-server/internal/modules/taskboard`
- `apps/api-server/routes`

必须完成：

1. 冻结并补齐以下接口的稳定返回结构、错误码和示例：
   - `/api/v1/stats/daily`
   - `/api/v1/stats/monthly`
   - `/api/v1/points/ledger`
   - `/api/v1/points/balance`
   - `/api/v1/word-lists`
   - `/api/v1/dictation-sessions/start`
   - `/api/v1/dictation-sessions/:session_id`
   - `/api/v1/dictation-sessions/:session_id/replay`
   - `/api/v1/dictation-sessions/:session_id/next`
2. 明确积分字段口径：
   - 总余额
   - 自动积分
   - 手工积分
   - 当日变化
   - 流水来源类型
3. 明确单词与听写字段口径：
   - word list 查询条件
   - item 结构
   - dictation session 当前词、索引、总量、状态
4. 补 route test / contract test，确保 Parent Web 和 Pad 不需要本地猜测字段。
5. 更新 API 契约文档与 smoke 文档，给 SC-03 / SC-04 可直接消费的示例。

验收：

1. `go test ./...` 通过
2. `API_ERROR_CONTRACT.md` 与真实实现一致
3. `TASKBOARD_API_SMOKE.md` 或同级文档包含本轮新增主链路
4. 至少有一组 route test 覆盖 points / words / monthly stats 主路径

禁止修改：

- `apps/api-server/internal/modules/agent`
- `apps/pad-app`
- `apps/parent-web`

### 4.2 `SC-02-GO-AGENT`

目标：

- 严格落实 Google Agentic design pattern：Agent 只负责解析和解释，不负责改统计、不负责越权写业务状态

唯一边界：

- `apps/api-server/internal/modules/agent`
- `apps/api-server/internal/platform/llm`
- `apps/api-server/internal/shared/agentic`

必须完成：

1. 把第一阶段报告型能力继续收口到：
   - daily encouragement
   - weekly encouragement
   - monthly encouragement
2. 所有报告输入必须来自确定性统计结构，不能让 Agent 自行计算任务数、积分、完成率。
3. 补 daily / monthly encouragement fixture 和回归测试。
4. 继续提升学校群式作业解析样本，重点覆盖：
   - 多学科混合
   - 子步骤拆分
   - 条件任务
   - 续做 / 订正但对象不明确
5. 在代码元数据或文档中明确本轮采用的 agentic pattern，并说明为什么适合第一阶段。

验收：

1. `go test ./internal/modules/agent/... ./internal/platform/llm/... ./internal/shared/agentic/...` 通过
2. Agent 输出不改写统计值
3. 模型不可用时仍有稳定模板回退
4. 不修改 taskboard / points / words 的业务写入路径

禁止修改：

- `apps/api-server/internal/modules/taskboard`
- `apps/api-server/internal/interfaces/http`
- `apps/pad-app`
- `apps/parent-web`

### 4.3 `SC-03-FLUTTER-PAD`

目标：

- 把 Pad 从“可演示”推进到“真实消费后端事实源”

唯一边界：

- `apps/pad-app`

必须完成：

1. 单词播放主路径改为真实后端：
   - 按 `family_id + child_id + assigned_date` 获取单词清单
   - 启动 dictation session
   - 当前词播放
   - 重播当前词
   - 下一词
   - 展示当前进度
2. 去掉主路径上的样例词初始化和手工文本框依赖。
3. 孩子端积分改为消费后端数据：
   - 今日积分变化
   - 当前积分或余额
   - 不再使用 `completed * 常量` 估算
4. 若后端已提供 daily stats，则孩子端简报优先基于 daily stats 展示。
5. 补 widget / integration 测试，覆盖：
   - 任务执行
   - 单词播放
   - 积分 / 简报展示

验收：

1. `flutter analyze` 通过
2. `flutter test` 通过
3. `flutter build web --dart-define=API_BASE_URL=http://localhost:8080` 通过
4. Chrome 或真实设备上可演示：
   - 打开当天任务
   - 完成任务
   - 拉取单词清单并播放
   - 显示后端积分结果

禁止修改：

- `apps/api-server`
- `apps/parent-web`

### 4.4 `SC-04-PARENT-WEB`

目标：

- 把家长端的几个关键“本地态”清掉，真正切到 Go 后端事实源

唯一边界：

- `apps/parent-web/src`
- `apps/parent-web/package.json`
- `apps/parent-web/package-lock.json`
- `apps/parent-web/index.html`

必须完成：

1. 单词清单改为真实后端：
   - 创建 / 更新清单走 `/api/v1/word-lists`
   - 按孩子和日期读取清单
   - 不再以 `localStorage` 为事实源
2. 积分改为真实后端：
   - 提交后读取或刷新 `/api/v1/points/ledger`
   - 展示 `/api/v1/points/balance`
   - 最近明细不再只靠本地数组
3. 月视图改为真实后端：
   - 使用 `/api/v1/stats/monthly`
   - 不再以前端 28 天任务聚合为主路径
4. 保持现有 `parse -> review -> confirm -> refresh` 主链路稳定，不倒退。
5. 补测试，明确验证：
   - word list 持久化来自后端
   - monthly view 使用真实接口
   - points 视图使用后端余额 / 流水

验收：

1. `npm run test` 通过
2. `npm run build` 通过
3. 家长端可演示：
   - 发布当天任务
   - 查看当日 / 周 / 月反馈
   - 创建单词清单
   - 查看积分流水与余额

禁止修改：

- `apps/api-server`
- `apps/pad-app`

### 4.5 `SC-05-INTEGRATION`

目标：

- 持续维护“交付就绪度”这条主线，让团队不再按旧文档和旧派单做事

唯一边界：

- `docs`
- `scripts`
- `README.md`
- `.env.example`
- `CHANGELOG.md`

必须完成：

1. 以 `docs/17_DELIVERY_READINESS.md` 为准，持续更新这轮阻塞项状态。
2. 维护 `Runbook / release checklist / demo checklist / dispatch` 一致。
3. 当 SC-03 / SC-04 合并后，第一时间补一轮：
   - `check_no_tracked_runtime_env`
   - `preflight_local_env`
   - `smoke_local_stack`
   - `demo_local_stack`
   - Go / Parent Web / Pad 的标准验证
4. 把“先验证，再 scoped add，再 commit，再 push”的模板继续维护成固定发布入口。

验收：

1. `bash scripts/check_no_tracked_runtime_env.sh` 通过
2. `bash scripts/preflight_local_env.sh` 通过
3. `bash scripts/smoke_local_stack.sh` 通过
4. `bash scripts/demo_local_stack.sh` 通过
5. `README / Runbook / Release Checklist / Delivery Readiness / Dispatch` 五份文档指向一致

禁止修改：

- `apps/api-server`
- `apps/pad-app`
- `apps/parent-web`

## 5. 推荐执行顺序

建议顺序：

1. `SC-01-GO-API` 先把双端依赖的 points / words / stats 契约冻结
2. `SC-02-GO-AGENT` 并行收口 daily / monthly encouragement 和 parser fixture
3. `SC-03-FLUTTER-PAD`、`SC-04-PARENT-WEB` 在契约冻结后并行接入
4. `SC-05-INTEGRATION` 最后统一复核演示与签收入口

## 6. 交付提醒

本轮要追求的是：

- 少本地态
- 少前端估算
- 少前端聚合
- 多后端确定性事实源

只要这条原则守住，第一阶段就会明显更接近可签收版本。
