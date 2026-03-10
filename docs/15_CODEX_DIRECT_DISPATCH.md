# StudyClaw 直接派单指令

本文档用于 `2026-03-10` 之后的多 Codex 并行开发。内容不是说明文，而是可以直接复制到对应 Codex 终端里的任务指令。

正式依据：

- 产品需求：`docs/01_PRD.md`
- 路线图：`docs/03_ROADMAP.md`
- Agent 约束：`docs/04_AGENTIC_DESIGN.md`
- 正式派单文档：`docs/14_NEXT_PHASE_DISPATCH.md`
- 第一阶段演示清单：`docs/16_FIRST_PHASE_DEMO_CHECKLIST.md`

使用规则：

1. 每个 Codex 只接收属于自己目录边界的任务
2. 不要删改别组目录
3. 每组完成后必须回报：
   - 改动摘要
   - 验收命令
   - 风险和阻塞

## `SC-01-GO-API`

直接发给该 Codex：

```text
你现在负责 StudyClaw 的 SC-01-GO-API。

当前目标：
把第一阶段核心业务数据和接口固定下来，避免 Parent Web / Pad 在实现时反复追着字段改。

必须先阅读：
- docs/01_PRD.md
- docs/03_ROADMAP.md
- docs/14_NEXT_PHASE_DISPATCH.md

你的唯一边界：
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

完成后必须回复：
- 改了哪些接口和数据结构
- 哪些字段已经冻结
- 运行了哪些测试
```

建议先执行：

```bash
cd /Users/admin/Documents/WORK/ai/studyclaw/apps/api-server
git status --short -- cmd config internal/app internal/interfaces/http internal/modules/taskboard routes
GOCACHE="$(pwd)/../../.cache/go-build" GOMODCACHE=/Users/admin/Documents/WORK/go/pkg/mod GOPROXY=off GOSUMDB=off go test ./...
```

## `SC-02-GO-AGENT`

直接发给该 Codex：

```text
你现在负责 StudyClaw 的 SC-02-GO-AGENT。

当前目标：
把第一阶段需要的 AI 能力严格收口在“任务解析”和“正向反馈总结”两个地方，不让 Agent 越界。

必须先阅读：
- docs/01_PRD.md
- docs/04_AGENTIC_DESIGN.md
- docs/14_NEXT_PHASE_DISPATCH.md

你的唯一边界：
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

完成后必须回复：
- 新增了哪些解析 / 报告能力
- 哪些 agent pattern 被明确采用
- 跑过哪些回归测试
```

建议先执行：

```bash
cd /Users/admin/Documents/WORK/ai/studyclaw/apps/api-server
git status --short -- internal/modules/agent internal/platform/llm internal/shared/agentic
GOCACHE="$(pwd)/../../.cache/go-build" GOMODCACHE=/Users/admin/Documents/WORK/go/pkg/mod GOPROXY=off GOSUMDB=off go test ./internal/modules/agent/... ./internal/platform/llm/... ./internal/shared/agentic/...
```

## `SC-03-FLUTTER-PAD`

直接发给该 Codex：

```text
你现在负责 StudyClaw 的 SC-03-FLUTTER-PAD。

当前目标：
把孩子端做成第一阶段真实可用的执行端，而不是只停留在任务列表演示。

必须先阅读：
- docs/01_PRD.md
- docs/03_ROADMAP.md
- docs/14_NEXT_PHASE_DISPATCH.md

你的唯一边界：
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

完成后必须回复：
- 孩子端补了哪些页面和状态
- 单词播放是怎么组织的
- 跑过哪些 Flutter 验证
```

建议先执行：

```bash
cd /Users/admin/Documents/WORK/ai/studyclaw/apps/pad-app
git status --short -- .
flutter pub get
flutter analyze
```

## `SC-04-PARENT-WEB`

直接发给该 Codex：

```text
你现在负责 StudyClaw 的 SC-04-PARENT-WEB。

当前目标：
把家长端做成第一阶段的主控制台，完成任务发布、积分管理、单词清单管理和报告查看。

必须先阅读：
- docs/01_PRD.md
- docs/03_ROADMAP.md
- docs/14_NEXT_PHASE_DISPATCH.md

你的唯一边界：
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

完成后必须回复：
- 家长端补了哪些主流程
- 哪些页面已可演示
- 跑过哪些前端测试和构建
```

建议先执行：

```bash
cd /Users/admin/Documents/WORK/ai/studyclaw/apps/parent-web
git status --short -- src package.json package-lock.json index.html
npm install
npm run build
```

## `SC-05-INTEGRATION`

直接发给该 Codex：

```text
你现在负责 StudyClaw 的 SC-05-INTEGRATION。

当前目标：
把第一阶段需求、演示路径、验收流程和派单方式继续收口成团队主入口。

必须先阅读：
- docs/01_PRD.md
- docs/03_ROADMAP.md
- docs/04_AGENTIC_DESIGN.md
- docs/14_NEXT_PHASE_DISPATCH.md

你的唯一边界：
- docs
- scripts
- README.md
- .env.example
- CHANGELOG.md

必须完成：
1. 继续维护第一阶段主文档，确保 PRD、Roadmap、Agentic Design、Dispatch 一致。
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

完成后必须回复：
- 更新了哪些文档 / 脚本
- 演示入口和发布入口分别是什么
- 跑过哪些验收命令
```

建议先执行：

```bash
cd /Users/admin/Documents/WORK/ai/studyclaw
git status --short -- README.md .env.example CHANGELOG.md docs scripts
bash scripts/check_no_tracked_runtime_env.sh
```
