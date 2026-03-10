# StudyClaw 下一阶段详细派单文档

本文档是当前唯一正式派单入口，用于把 StudyClaw 从 `v0.1.1` 联调基线推进到 `v0.2.0` 第一阶段交付版。

适用状态：

- 日期：`2026-03-09`
- 当前项目版本：`v0.1.1`
- 当前目标版本：`v0.2.0`
- 当前阶段目标：完成第一阶段 7 类核心产品能力

相关入口：

- `docs/01_PRD.md`
- `docs/03_ROADMAP.md`
- `docs/04_AGENTIC_DESIGN.md`
- `docs/15_CODEX_DIRECT_DISPATCH.md`
- `docs/16_FIRST_PHASE_DEMO_CHECKLIST.md`
- `docs/06_RUNBOOK.md`
- `docs/13_RELEASE_CHECKLIST.md`
- `scripts/demo_local_stack.sh`

## 1. 下一阶段总目标

下一阶段不再只围绕“演示脚本和联调基线”，而是进入真正的第一阶段产品交付。

总目标：

1. 家长能发布某一天的老师作业
2. AI 能返回可审核的结构化任务草稿
3. 孩子能在 Pad 上完成当天任务
4. 家长能看到及时同步的完成情况和当日反馈
5. 单词清单可在 Pad 上逐词播放
6. 积分可自动结算，也可由家长手工 `+/-`
7. 日 / 周 / 月数据可视化和 AI 鼓励可用

## 2. 第一阶段退出标准

只有下面条件都满足，才算 `v0.2.0` 可验收：

1. 家长端支持“输入 -> 解析 -> 审核 -> 发布”
2. 任务草稿包含 `subject / group_title / title / confidence / needs_review`
3. Pad 端支持当天任务拉取、勾选、刷新和错误反馈
4. 家长端支持查看当天完成统计和最近同步结果
5. 家长可创建单词清单，Pad 可逐词播放
6. 积分支持自动和手工两种来源，并双端可见
7. 日 / 周 / 月图表和 AI 鼓励可用

## 3. 各 Codex 优先级总表

| Codex | 主目标 | 优先级 | 核心产出 |
| --- | --- | --- | --- |
| `SC-01-GO-API` | 第一阶段领域模型与接口冻结 | P0 | 任务发布、积分、统计、单词清单 API |
| `SC-02-GO-AGENT` | 解析与报告 Agent 收口 | P0 | 解析质量、日报周报月报鼓励、pattern 约束 |
| `SC-03-FLUTTER-PAD` | 孩子端任务 + 播放 + 积分 | P0 | 今日任务板、单词播放、孩子端简报 |
| `SC-04-PARENT-WEB` | 家长端发布 + 报告 + 积分 | P0 | 发布流、报告页、积分页、单词清单管理 |
| `SC-05-INTEGRATION` | 需求、脚本、验收与发布收口 | P0 | PRD/Runbook/Checklist/派单/演示清单 |

## 4. 各 Codex 详细任务

### 4.1 `SC-01-GO-API`

```text
[SC-01-GO-API]
目标：把第一阶段核心业务数据和接口固定下来，避免 Parent Web / Pad 在实现时反复追着字段改。

边界：
- apps/api-server/cmd
- apps/api-server/config
- apps/api-server/internal/app
- apps/api-server/internal/interfaces/http
- apps/api-server/internal/modules/taskboard
- apps/api-server/routes

必须完成：
1. 设计并落地第一阶段核心实体在 API 层的表达：
   - daily assignment draft / publish
   - task item
   - points ledger / points balance
   - word list / word item / dictation session
   - daily / weekly / monthly stats response
2. 固化“家长发布任务”和“Pad 拉取当天任务”的接口契约。
3. 增加积分流水相关接口，并明确自动积分与手工积分的来源字段。
4. 增加日 / 周 / 月统计接口，确保图表数据由后端确定性生成。
5. 输出第一阶段 API 冻结说明，明确哪些字段是双端稳定依赖。

验收：
1. go test ./... 通过
2. API_ERROR_CONTRACT.md 补齐第一阶段新增接口和错误路径
3. TASKBOARD_API_SMOKE.md 或同级 smoke 文档能覆盖第一阶段主链路
4. Parent Web 和 Pad 不需要自行猜测接口结构

禁止修改：
- apps/api-server/internal/modules/agent
- apps/pad-app
- apps/parent-web
```

### 4.2 `SC-02-GO-AGENT`

```text
[SC-02-GO-AGENT]
目标：把第一阶段需要的 AI 能力严格收口在“任务解析”和“正向反馈总结”两个地方，不让 Agent 越界。

边界：
- apps/api-server/internal/modules/agent
- apps/api-server/internal/platform/llm
- apps/api-server/internal/shared/agentic

必须完成：
1. 继续提升学校群式任务解析质量，重点覆盖：
   - 多学科混合
   - 子步骤拆分
   - 条件任务
   - 续做 / 订正但对象不明确
   - 不该误判的普通任务
2. 设计并落地日报 / 周报 / 月报的 AI 总结能力：
   - 输入必须来自确定性统计
   - 输出必须是正向、支持型文案
   - 模型不可用时有模板回退
3. 把第一阶段 Agentic pattern 选择写入代码元数据和文档，确保和 Google 设计约束一致。
4. 产出第一阶段 agent fixture，便于联调和回归。

验收：
1. go test ./internal/modules/agent/... ./internal/platform/llm/... ./internal/shared/agentic/... 通过
2. 解析结果对 front-end 可解释
3. 报告文案不改写统计值
4. 不改 taskboard、积分和 HTTP 状态写入逻辑

禁止修改：
- apps/api-server/internal/modules/taskboard
- apps/api-server/internal/interfaces/http
- apps/pad-app
- apps/parent-web
```

### 4.3 `SC-03-FLUTTER-PAD`

```text
[SC-03-FLUTTER-PAD]
目标：把孩子端做成第一阶段真实可用的执行端，而不是只停留在任务列表演示。

边界：
- apps/pad-app

必须完成：
1. 做好“当天任务板”主路径：
   - 默认加载当天任务
   - 单任务勾选
   - 分组勾选
   - 全部完成
   - 刷新与错误反馈
2. 增加“单词播放模式”：
   - 当前词播放
   - 重播
   - 下一词
   - 播放进度展示
3. 在孩子端展示：
   - 今日积分
   - 今日完成进度
   - 简化版日报 / 周报入口
4. 补 widget / integration 测试，覆盖任务同步和单词播放关键路径。

验收：
1. flutter analyze 通过
2. flutter test 通过
3. flutter build web --dart-define=API_BASE_URL=http://localhost:8080 通过
4. Chrome 或真实设备上可以完成“任务执行 + 单词播放 + 查看积分”

禁止修改：
- apps/api-server
- apps/parent-web
```

### 4.4 `SC-04-PARENT-WEB`

```text
[SC-04-PARENT-WEB]
目标：把家长端做成第一阶段的主控制台，完成任务发布、积分管理、单词清单管理和报告查看。

边界：
- apps/parent-web/src
- apps/parent-web/package.json
- apps/parent-web/package-lock.json
- apps/parent-web/index.html

必须完成：
1. 做好“某一天作业发布”主路径：
   - 选择孩子
   - 选择日期
   - 输入原文
   - parse
   - 风险审核
   - 编辑 / 删除 / 补充
   - publish
2. 做好“家长查看反馈”主路径：
   - 当日完成率
   - 积分变化
   - 日报摘要
   - 周 / 月图表入口
3. 做好“积分操作”主路径：
   - 家长手工加分
   - 家长手工扣分
   - 录入表扬 / 批评原因
4. 做好“单词清单管理”：
   - 创建清单
   - 编辑词项
   - 绑定到某个孩子和日期

验收：
1. npm run test 通过
2. npm run build 通过
3. 家长端可完成“发布任务 + 看报告 + 调积分 + 配单词清单”

禁止修改：
- apps/api-server
- apps/pad-app
```

### 4.5 `SC-05-INTEGRATION`

```text
[SC-05-INTEGRATION]
目标：把第一阶段需求、演示路径、验收流程和派单方式继续收口成团队主入口。

边界：
- docs
- scripts
- README.md
- .env.example
- CHANGELOG.md

必须完成：
1. 把 docs/01_PRD.md、docs/03_ROADMAP.md、docs/04_AGENTIC_DESIGN.md 和 docs/14_NEXT_PHASE_DISPATCH.md 收口到第一阶段目标。
2. 继续维护 demo_local_stack.sh、Runbook 和 release checklist，让新人能按第一阶段链路演示。
3. 为第一阶段增加更明确的演示清单：
   - 发布当天作业
   - Pad 完成任务
   - 家长查看统计
   - 单词播放
   - 积分变化
   - 日 / 周 / 月反馈与 AI 鼓励
4. 为后续 GitHub 同步维护固定的“验证 -> scoped add -> commit -> push”模板。

验收：
1. bash scripts/check_no_tracked_runtime_env.sh 通过
2. bash scripts/preflight_local_env.sh 通过
3. bash scripts/smoke_local_stack.sh 通过
4. bash scripts/demo_local_stack.sh 通过
5. PRD / Roadmap / Agentic Design / Dispatch 四份文档对齐

禁止修改：
- apps/api-server
- apps/pad-app
- apps/parent-web
```

## 5. 推荐执行顺序

### 第一批：先锁领域和约束

- `SC-01-GO-API`
- `SC-02-GO-AGENT`
- `SC-05-INTEGRATION`

原因：

- 这三组决定第一阶段的数据边界、AI 边界和验收方式
- 如果这一步不先收口，前端会反复返工

### 第二批：双端并行实现

- `SC-04-PARENT-WEB`
- `SC-03-FLUTTER-PAD`

原因：

- 家长端和孩子端都依赖第一批先稳定接口和文档
- 一旦接口冻结，这两组可以高并行推进

### 第三批：集成回归

- `SC-05-INTEGRATION`
- 必要时由 owner 组各自回收修复

原因：

- 最后一轮应回到演示、验收、发布前检查
- 不应在最后阶段由集成组越界改业务代码

## 6. 派单模板

```text
[终端名]
目标：
边界：
必须完成：
验收：
禁止修改：
```

## 7. 使用方式

你后续派任务时直接：

1. 打开本文件
2. 找到目标 Codex
3. 复制对应代码块
4. 粘贴到对应 Codex

注意：

- `docs/10`、`docs/11`、`docs/12` 只保留为历史记录
- 本文件就是当前唯一正式派单入口
