# StudyClaw 下一阶段正式派单文档

说明：

- 本文档从 `2026-03-14` 开始，作为下一阶段唯一正式派单入口。
- 当前稳定基线是 `v0.3.5`，不再沿用旧的 `v0.2.0` / `v0.3.1` 派单背景。
- 所有后续任务分发，都应以本文件和 `docs/03_ROADMAP.md` 为准。

适用状态：

- 日期：`2026-03-16`
- 当前正式基线：`v0.4.0`
- 当前目标版本：`v0.4.1`
- 当前结论：`第二阶段主链已正式签收，v0.4.0 已成为当前 Agentic 学习助手正式基线`
- 当前已完成：学习素材来源可追溯、Pad 持续监听不因短暂停顿自动退出、transcript 分段时间线、背诵/朗读 noisy transcript 对齐增强、家长端语音学习结果摘要复盘闭环
- 下一阶段重点：语音稳定化、时间线体验优化、评测误判回归与家长端解释可读性提升

必须先阅读的主文档：

- `README.md`
- `docs/01_PRD.md`
- `docs/03_ROADMAP.md`
- `docs/04_AGENTIC_DESIGN.md`
- `docs/06_RUNBOOK.md`
- `docs/13_RELEASE_CHECKLIST.md`
- `docs/17_DELIVERY_READINESS.md`
- `docs/19_DELIVERY_UAT_CASES.md`

## 1. 当前这轮真正要解决的问题

当前主线已经不再是“把第一阶段补到能演示”，而是把下面这条主线做成真实可用的第二阶段能力：

“学习素材前置准备 -> 孩子语音学习 -> 系统分段记录 -> grounded 分析 -> 家长复盘与干预”

本轮只解决 5 个 `v0.4.0` 核心阻塞项：

1. 学习素材虽然已经能自动补全，但还没有形成稳定、可管理、可追溯的前置准备链路
2. 孩子端语音工作台虽然已经支持多模式，但还缺真正稳定的“人工开始到人工结束”的持续监听主路径
3. transcript 分段仍然偏弱，缺少接近会议纪要式的时间线和停顿切分能力
4. 背诵 / 朗读分析对 noisy ASR 的标题识别、正文对齐、错漏定位还不够稳
5. 家长端还缺“孩子本次语音学习结果”的清晰复盘入口

本轮不做：

- OCR / VLM 多模态作业解析
- 多智能体编排
- 自动奖惩决策
- 强实时推送
- 新一轮大规模 UI 重构

## 2. 本轮退出标准

只有下面条件都满足，才允许把版本朝 `v0.4.0` 签收推进：

1. 家长在任务发布阶段就能管理或确认背诵 / 朗读学习素材
2. 家长未输入学习素材时，系统仍可按“老师原文抽取优先，LLM 只补缺口”的规则稳定补全
3. Pad 语音工作台支持三种主路径：
   - 短口令模式
   - 长段朗读 / 背诵模式
   - 陪伴式持续监听模式
4. 在人工点击“开始说话”到“结束说话”之间，监听主流程不因短暂停顿自动退出
5. transcript 能输出分段结果、时间点和最终合并文本
6. 背诵 / 朗读分析能输出：
   - 标题 / 作者识别
   - 标准参考文本
   - 完成度
   - 逐句匹配
   - 错漏点
   - 是否建议重试
7. 家长端能查看语音学习结果摘要，并据此决定是否重练
8. `preflight / smoke / demo / go test / web test-build / pad analyze-test-build` 全部通过

## 3. 本轮工作流总表

治理约束（接管后统一执行）：

- 所有 lane 必须围绕家庭学习主环路交付，不新增偏离主线的“炫技需求”
- 业务状态坚持确定性优先，Agent 输出不能直接改任务状态、积分或统计事实
- 执行顺序按依赖推进：`SC-01` 冻结契约 -> `SC-03/SC-04` 对齐消费 -> `SC-05` 验收收口，`SC-02` 贯穿提供分析能力
- 文档口径以 `README.md`、`docs/03_ROADMAP.md`、`docs/14_NEXT_PHASE_DISPATCH.md`、`docs/17_DELIVERY_READINESS.md` 为接管后四份 canonical 文档

### 3.1 `v0.4.0` Integration / Release Gates（跨端切片）

为避免“功能做完但无法交接”，本轮所有跨端切片统一按以下 gate 推进。每过一层 gate，才允许进入下一层。

| Gate | 目标 | 必跑命令 / 操作 | 通过标准 | 证据落点 |
| --- | --- | --- | --- | --- |
| `G0-Verification` | 本地环境与 release 范围可验证 | `bash scripts/check_no_tracked_runtime_env.sh`<br>`bash scripts/preflight_local_env.sh`<br>`bash scripts/check_release_scope.sh` | 无运行时密钥泄漏；环境依赖齐全；工作区不含禁止发布路径 | `docs/13_RELEASE_CHECKLIST.md`、`docs/17_DELIVERY_READINESS.md` |
| `G1-Smoke` | 三端最小可运行主链打通 | `STUDYCLAW_SMOKE_API_BASE_URL=http://127.0.0.1:38080 bash scripts/smoke_local_stack.sh` | API 健康、最小任务板链路、Parent build、Pad web build 全部通过 | `docs/17_DELIVERY_READINESS.md`、`docs/19_DELIVERY_UAT_CASES.md` |
| `G2-Demo` | 可演示的跨端业务故事成立 | `STUDYCLAW_SMOKE_API_BASE_URL=http://127.0.0.1:38080 STUDYCLAW_PARENT_WEB_URL=http://127.0.0.1:5173 bash scripts/demo_local_stack.sh` | 能按 runbook 完成 `发布 -> 孩子执行 -> 家长复盘` 演示路径 | `docs/06_RUNBOOK.md`、`docs/16_FIRST_PHASE_DEMO_CHECKLIST.md` |
| `G3-Acceptance` | `v0.4.0` 切片级签收可复核 | 按 `docs/19_DELIVERY_UAT_CASES.md` 执行对应切片用例（含语音、背诵分析、学习素材管理） | 每个切片至少 1 条跨端主用例通过，并能回填风险/未完成项 | `docs/19_DELIVERY_UAT_CASES.md`、PR 描述或 release notes |

跨端切片统一最小口径（用于 `G3`）：

1. **学习素材切片**：Parent 发布/审核字段 -> API 持久化 -> Pad 读取元数据一致。
2. **语音学习切片**：Pad 语音输入/分段 -> API 分析返回 -> Parent 可复盘结果。
3. **任务与反馈切片**：Parent 发布 -> Pad 完成 -> Parent 日/月反馈与积分变化一致。

说明：

- `G0~G3` 是本轮 `SC-05-INTEGRATION` 的统一出口标准；未通过 `G3` 的切片不得标记“可发布”。
- 若本轮只改文档或脚本，也至少要重新执行 `G0`，并在 PR 中说明未触发的 gate 原因。

| Codex | 本轮主目标 | 优先级 | 交付物 |
| --- | --- | --- | --- |
| `SC-01-GO-API` | 冻结学习素材、语音会话、recitation 分析接口契约 | P0 | DTO、handler、route test、契约文档 |
| `SC-02-GO-AGENT` | 强化 transcript 归一化、标题识别、正文对齐和重试建议 | P0 | agent logic、fixture、回归测试 |
| `SC-03-FLUTTER-PAD` | 把学习语音工作台做成稳定主链 | P0 | 页面、controller、repository、测试 |
| `SC-04-PARENT-WEB` | 增加学习素材管理和语音结果复盘入口 | P0 | H5 子页面、API 集成、测试 |
| `SC-05-INTEGRATION` | 维护路线图、验收标准、评测数据和发版文档 | P0 | roadmap / dispatch / UAT / release docs 同步 |

## 4. 各 Codex 详细任务

### 4.1 `SC-01-GO-API`

目标：

- 把第二阶段真正会被双端消费的学习素材和语音分析接口冻结下来，避免前端继续本地猜字段

唯一边界：

- `apps/api-server/cmd`
- `apps/api-server/config`
- `apps/api-server/internal/app`
- `apps/api-server/internal/interfaces/http`
- `apps/api-server/internal/modules/taskboard`
- `apps/api-server/routes`

必须完成：

1. 明确任务学习素材的稳定字段口径：
   - `reference_title`
   - `reference_author`
   - `reference_text`
   - `hide_reference_from_child`
   - `analysis_mode`
   - `reference_source`
2. 冻结语音工作台会话结构：
   - session id
   - mode
   - scene
   - started_at / ended_at
   - transcript segments
   - merged transcript
   - analysis summary
3. 明确 `recitation/analyze` 的稳定返回字段：
   - normalized title / author / text
   - matched lines
   - missing / extra / confused tokens
   - completion rate
   - retry recommendation
4. 补 route test / contract test，确保 Parent Web 和 Pad 不需要猜字段
5. 更新 API 契约文档与 runbook 示例

验收：

1. `go test ./... -count=1` 通过
2. API 契约文档与真实实现一致
3. 至少一组 route test 覆盖 study material / voice session / recitation analyze 主路径
4. `docs/06_RUNBOOK.md` 或同级文档包含第二阶段新增主链路示例

禁止修改：

- `apps/api-server/internal/modules/agent`
- `apps/pad-app`
- `apps/parent-web`

### 4.2 `SC-02-GO-AGENT`

目标：

- 严格落实 Google Agentic design pattern：Agent 只负责 transcript 归一化、标题识别、正文对齐和解释，不负责改业务状态

唯一边界：

- `apps/api-server/internal/modules/agent`
- `apps/api-server/internal/platform/llm`
- `apps/api-server/internal/shared/agentic`

必须完成：

1. 增强 noisy ASR transcript 归一化：
   - 常见同音误识别修复
   - 标点与停顿恢复
   - 标题 / 作者识别
2. 对古诗词、课文、英语朗读分别补 fixture 和回归样本
3. 产出逐句对齐结果，而不是只返回粗略完成率
4. 输出清晰的“建议重背 / 建议继续 / 需要家长协助”结论
5. 所有结论必须保留置信度或解释依据

验收：

1. `go test ./internal/modules/agent/... ./internal/platform/llm/... ./internal/shared/agentic/...` 通过
2. 模型不可用时仍有稳定模板 / 规则回退
3. Agent 输出不直接写任务状态、积分或统计值
4. 至少一组 noisy poem transcript 能正确识别标题并对齐正文

禁止修改：

- `apps/api-server/internal/modules/taskboard`
- `apps/api-server/internal/interfaces/http`
- `apps/pad-app`
- `apps/parent-web`

### 4.3 `SC-03-FLUTTER-PAD`

目标：

- 把学习语音工作台从“可演示”推进到“孩子能连续使用”的真实主链

唯一边界：

- `apps/pad-app`

必须完成：

1. 保证三种语音模式都走统一稳定入口：
   - 短口令
   - 长段 transcript
   - 陪伴模式
2. 在人工点击“开始说话”后，不因短暂停顿自动关闭主流程
3. 在人工点击“结束说话”前，持续累计 transcript segment
4. 用时间线展示分段结果，而不是只有最终一句合并文本
5. 背诵 / 朗读分析结果在 Pad 上可读、可重试、可收到鼓励
6. 对隐藏参考原文任务，孩子仍看不到答案，但系统可以用参考原文分析

验收：

1. `flutter analyze` 通过
2. `flutter test --no-pub` 通过
3. `flutter build web --dart-define=API_BASE_URL=http://127.0.0.1:38080` 通过
4. 真机或浏览器可演示：
   - 开始说话
   - 中途停顿
   - 继续说话
   - 手动结束
   - 看到 transcript 时间线与分析结果

禁止修改：

- `apps/api-server`
- `apps/parent-web`

### 4.4 `SC-04-PARENT-WEB`

目标：

- 让家长端不仅能布置任务，还能真正管理学习素材并复盘孩子的语音学习结果

唯一边界：

- `apps/parent-web/src`
- `apps/parent-web/package.json`
- `apps/parent-web/package-lock.json`
- `apps/parent-web/index.html`

必须完成：

1. 在发布 / 审核链路中把学习素材管理前置化：
   - 标题
   - 作者
   - 正文
   - 是否对孩子隐藏
   - 来源说明
2. 补充“语音学习结果”或同等复盘入口
3. 家长能看到：
   - 最近一次 transcript 摘要
   - 完成度
   - 错漏点
   - 是否建议重练
4. 保持现有 H5 工位与发布主路径不倒退
5. 补测试，验证学习素材和语音结果来自真实后端结构

验收：

1. `npm test -- --run` 通过
2. `npm run build` 通过
3. 手机视口下不退回 PC 长页面
4. 背诵 / 朗读任务的素材与复盘入口可在 H5 中稳定找到

禁止修改：

- `apps/api-server`
- `apps/pad-app`

### 4.5 `SC-05-INTEGRATION`

目标：

- 继续维护“交付就绪度”这条主线，让团队不再按旧文档和旧派单做事

唯一边界：

- 根目录 `README.md`
- `docs/`
- `scripts/`
- 如需验证可只读三端代码，但不修改非本边界源码

必须完成：

1. 维护 `README / Runbook / Roadmap / Dispatch / Release Checklist / Delivery Readiness / UAT` 一致
2. 固定第二阶段的评测样本：
   - 古诗词 noisy transcript
   - 课文背诵 transcript
   - 英语朗读 transcript
3. 明确 `v0.4.0` 的 smoke / demo / UAT 主路径
4. 维护发版说明、操作手册和一页摘要
5. 确保 GitHub 同步和本地文档指向一致

验收：

1. `bash scripts/check_no_tracked_runtime_env.sh` 通过
2. `bash scripts/preflight_local_env.sh` 通过
3. `bash scripts/check_release_scope.sh` 通过
4. `README / Runbook / Release Checklist / Delivery Readiness / Dispatch` 五份文档指向一致
5. 交付清单中能直接找到第二阶段主线验收命令

## 5. 推荐提交流程

每个 Codex 完成后，建议统一按下面格式回报：

1. 修改范围
2. 关键决策
3. 验证命令与结果
4. 风险 / 未完成项
5. 是否可以进入 GitHub sync

## 6. 当前阶段总判断

`v0.3.5` 已经把第一阶段正式交付问题收口完成。

从这一刻开始，如果还继续沿用旧派单口径，团队只会反复清理已经解决过的问题；真正应该投入的是：

- 孩子学习语音工作台
- 学习素材前置管理
- grounded 的背诵 / 朗读分析
- 家长端语音学习复盘

这份文档就是下一阶段唯一正式派单入口。
