# StudyClaw 本轮可直接派发指令

本文档用于 `2026-03-10` 之后的多 Codex 并行开发。内容不是解释文，而是可以直接复制到对应 Codex 终端里的任务指令。

正式依据：

- `docs/01_PRD.md`
- `docs/03_ROADMAP.md`
- `docs/04_AGENTIC_DESIGN.md`
- `docs/14_NEXT_PHASE_DISPATCH.md`
- `docs/17_DELIVERY_READINESS.md`

使用规则：

1. 每个 Codex 只接收自己目录边界内的任务
2. 不要删改别组目录
3. 每组完成后必须回复：
   - 改动摘要
   - 验收命令
   - 风险和阻塞

## `SC-01-GO-API`

直接发给该 Codex：

```text
你现在负责 StudyClaw 的 SC-01-GO-API。

当前目标：
冻结第一阶段真正会被双端消费的 points / words / dictation / stats API 契约，避免前端继续本地猜字段。

必须先阅读：
- docs/01_PRD.md
- docs/03_ROADMAP.md
- docs/14_NEXT_PHASE_DISPATCH.md
- docs/17_DELIVERY_READINESS.md

你的唯一边界：
- apps/api-server/cmd
- apps/api-server/config
- apps/api-server/internal/app
- apps/api-server/internal/interfaces/http
- apps/api-server/internal/modules/taskboard
- apps/api-server/routes

必须完成：
1. 冻结并补齐以下接口的稳定返回结构、错误码和示例：
   - /api/v1/stats/daily
   - /api/v1/stats/monthly
   - /api/v1/points/ledger
   - /api/v1/points/balance
   - /api/v1/word-lists
   - /api/v1/dictation-sessions/start
   - /api/v1/dictation-sessions/:session_id
   - /api/v1/dictation-sessions/:session_id/replay
   - /api/v1/dictation-sessions/:session_id/next
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
4. 补 route test / contract test，确保 Parent Web 和 Pad 不需要本地猜字段。
5. 更新 API 契约文档与 smoke 文档，给 SC-03 / SC-04 可直接消费的示例。

验收：
1. go test ./... 通过
2. API_ERROR_CONTRACT.md 与真实实现一致
3. TASKBOARD_API_SMOKE.md 或同级文档包含本轮新增主链路
4. 至少有一组 route test 覆盖 points / words / monthly stats 主路径

禁止修改：
- apps/api-server/internal/modules/agent
- apps/pad-app
- apps/parent-web

完成后必须回复：
- 冻结了哪些接口和字段
- 更新了哪些契约文档 / 测试
- 跑过哪些 Go 验证
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
严格落实 Google Agentic design pattern：Agent 只负责解析和解释，不负责改统计、不负责越权写业务状态。

必须先阅读：
- docs/01_PRD.md
- docs/04_AGENTIC_DESIGN.md
- docs/14_NEXT_PHASE_DISPATCH.md
- docs/17_DELIVERY_READINESS.md

你的唯一边界：
- apps/api-server/internal/modules/agent
- apps/api-server/internal/platform/llm
- apps/api-server/internal/shared/agentic

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
1. go test ./internal/modules/agent/... ./internal/platform/llm/... ./internal/shared/agentic/... 通过
2. Agent 输出不改写统计值
3. 模型不可用时仍有稳定模板回退
4. 不修改 taskboard / points / words 的业务写入路径

禁止修改：
- apps/api-server/internal/modules/taskboard
- apps/api-server/internal/interfaces/http
- apps/pad-app
- apps/parent-web

完成后必须回复：
- 新增了哪些 daily / monthly agent 能力
- 明确采用了哪些 agent pattern
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
把 Pad 从“可演示”推进到“真实消费后端事实源”。

必须先阅读：
- docs/01_PRD.md
- docs/03_ROADMAP.md
- docs/14_NEXT_PHASE_DISPATCH.md
- docs/17_DELIVERY_READINESS.md

你的唯一边界：
- apps/pad-app

必须完成：
1. 单词播放主路径改为真实后端：
   - 按 family_id + child_id + assigned_date 获取单词清单
   - 启动 dictation session
   - 当前词播放
   - 重播当前词
   - 下一词
   - 展示当前进度
2. 去掉主路径上的样例词初始化和手工文本框依赖。
3. 孩子端积分改为消费后端数据：
   - 今日积分变化
   - 当前积分或余额
   - 不再使用 completed * 常量 估算
4. 若后端已提供 daily stats，则孩子端简报优先基于 daily stats 展示。
5. 补 widget / integration 测试，覆盖：
   - 任务执行
   - 单词播放
   - 积分 / 简报展示

验收：
1. flutter analyze 通过
2. flutter test 通过
3. flutter build web --dart-define=API_BASE_URL=http://localhost:8080 通过
4. Chrome 或真实设备上可演示：
   - 打开当天任务
   - 完成任务
   - 拉取单词清单并播放
   - 显示后端积分结果

禁止修改：
- apps/api-server
- apps/parent-web

完成后必须回复：
- 去掉了哪些本地样例 / 估算逻辑
- Pad 现在接了哪些真实接口
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
把家长端的几个关键“本地态”清掉，真正切到 Go 后端事实源。

必须先阅读：
- docs/01_PRD.md
- docs/03_ROADMAP.md
- docs/14_NEXT_PHASE_DISPATCH.md
- docs/17_DELIVERY_READINESS.md

你的唯一边界：
- apps/parent-web/src
- apps/parent-web/package.json
- apps/parent-web/package-lock.json
- apps/parent-web/index.html

必须完成：
1. 单词清单改为真实后端：
   - 创建 / 更新清单走 /api/v1/word-lists
   - 按孩子和日期读取清单
   - 不再以 localStorage 为事实源
2. 积分改为真实后端：
   - 提交后读取或刷新 /api/v1/points/ledger
   - 展示 /api/v1/points/balance
   - 最近明细不再只靠本地数组
3. 月视图改为真实后端：
   - 使用 /api/v1/stats/monthly
   - 不再以前端 28 天任务聚合为主路径
4. 保持现有 parse -> review -> confirm -> refresh 主链路稳定，不倒退。
5. 补测试，明确验证：
   - word list 持久化来自后端
   - monthly view 使用真实接口
   - points 视图使用后端余额 / 流水

验收：
1. npm run test 通过
2. npm run build 通过
3. 家长端可演示：
   - 发布当天任务
   - 查看当日 / 周 / 月反馈
   - 创建单词清单
   - 查看积分流水与余额

禁止修改：
- apps/api-server
- apps/pad-app

完成后必须回复：
- 去掉了哪些本地状态来源
- 哪些页面已经切到真实后端
- 跑过哪些前端测试和构建
```

建议先执行：

```bash
cd /Users/admin/Documents/WORK/ai/studyclaw/apps/parent-web
git status --short -- src package.json package-lock.json index.html
npm install
npm run test
```

## `SC-05-INTEGRATION`

直接发给该 Codex：

```text
你现在负责 StudyClaw 的 SC-05-INTEGRATION。

当前目标：
持续维护“交付就绪度”这条主线，让团队不再按旧文档和旧派单做事。

必须先阅读：
- docs/01_PRD.md
- docs/03_ROADMAP.md
- docs/04_AGENTIC_DESIGN.md
- docs/14_NEXT_PHASE_DISPATCH.md
- docs/17_DELIVERY_READINESS.md

你的唯一边界：
- docs
- scripts
- README.md
- .env.example
- CHANGELOG.md

必须完成：
1. 以 docs/17_DELIVERY_READINESS.md 为准，持续更新这轮阻塞项状态。
2. 维护 Runbook / release checklist / demo checklist / dispatch 一致。
3. 当 SC-03 / SC-04 合并后，第一时间补一轮：
   - check_no_tracked_runtime_env
   - preflight_local_env
   - smoke_local_stack
   - demo_local_stack
   - Go / Parent Web / Pad 的标准验证
4. 把“先验证，再 scoped add，再 commit，再 push”的模板继续维护成固定发布入口。

验收：
1. bash scripts/check_no_tracked_runtime_env.sh 通过
2. bash scripts/preflight_local_env.sh 通过
3. bash scripts/smoke_local_stack.sh 通过
4. bash scripts/demo_local_stack.sh 通过
5. README / Runbook / Release Checklist / Delivery Readiness / Dispatch 五份文档指向一致

禁止修改：
- apps/api-server
- apps/pad-app
- apps/parent-web

完成后必须回复：
- 更新了哪些文档 / 脚本
- 当前阻塞项状态是否变化
- 跑过哪些验收命令
```

建议先执行：

```bash
cd /Users/admin/Documents/WORK/ai/studyclaw
git status --short -- docs scripts README.md .env.example CHANGELOG.md
bash scripts/check_no_tracked_runtime_env.sh
bash scripts/preflight_local_env.sh
```
