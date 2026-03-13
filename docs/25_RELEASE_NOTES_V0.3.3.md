# StudyClaw v0.3.3 发布说明

发布日期：`2026-03-13`

## 版本定位

`v0.3.3` 是在 `v0.3.2` 热修复基线上的正式能力增强版。

本次发布的目标不是继续堆页面，而是把“背诵 / 朗读任务的标准学习素材”和“孩子语音学习工作台”正式纳入可交付主链。

## 本次发布包含什么

### 1. 学习素材自动补全

- 家长发布背诵 / 朗读任务时，支持 `reference_title`、`reference_author`、`reference_text`、`hide_reference_from_child`、`analysis_mode`
- 家长手动输入始终优先
- 如果家长没填，先从老师原文中自动抽取
- 如果老师原文仍缺正文，再用 LLM 只补全文本空缺
- 背诵任务默认对孩子隐藏标准原文，避免照读冒充背诵

### 2. 孩子学习语音工作台

- Pad 支持短指令、长段朗读 / 背诵和陪伴式持续监听三种语音场景
- “好了 / 下一个 / 继续 / Next / 数学订正好了 / 一课一练做完了”这类短指令继续可用
- 长段学习过程不再要求孩子频繁重开监听，更适合古诗词、课文、英语朗读等场景

### 3. 背诵分析闭环

- 新增 `/api/v1/recitation/analyze`
- 能在 noisy transcript 条件下识别标题、作者和正文主体
- 能给出逐句匹配、完成度、是否建议重背、总结和建议
- 可以直接复用发布阶段保存下来的隐藏参考原文

### 4. 解析器与发布链路增强

- 背诵 / 朗读任务会自动推断 `task_type`
- 老师原文里紧跟在背诵任务后的正文块不再并进任务标题
- `/api/v1/tasks/parse`、草稿保存、确认发布和任务板读写链路统一保留学习素材元数据

## 对使用者的实际影响

### 家长

- 发布古诗词 / 课文任务时，通常不需要再手工重复录入标题、作者和正文
- 审核卡里就能看到系统已带出的标准内容，并决定是否保留给孩子可见
- 移动 H5 发布路径更接近“录入 -> 审核 -> 发布”的真实 App 流程

### 孩子

- 可以用更长时间、更连续的语音输入完成学习任务
- 背诵时不用自己再输入参考内容
- 背诵结束后能获得更像“学习反馈”的分析，而不是只有原始识别文本

### 团队 / 交付

- API 契约新增的是已有解析结果中的元数据字段，不是破坏性改动
- 文档、版本号、发布说明和一页摘要统一同步到 `v0.3.3`
- 可以把 `v0.3.3` 作为下一阶段启动前的正式 GitHub release 基线

## 验证摘要

本次发布前已完成：

- `bash scripts/check_no_tracked_runtime_env.sh`
- `bash scripts/preflight_local_env.sh`
- `bash scripts/check_release_scope.sh`
- `cd apps/api-server && go test ./... -count=1`
- `cd apps/parent-web && npm test -- --run`
- `cd apps/parent-web && npm run build`
- `cd apps/pad-app && flutter test --no-pub`
- `cd apps/pad-app && flutter analyze`
- `cd apps/pad-app && flutter build web --dart-define=API_BASE_URL=http://127.0.0.1:38080`

## 相关文档

- [README.md](/Users/admin/Documents/WORK/ai/studyclaw/README.md)
- [docs/06_RUNBOOK.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/06_RUNBOOK.md)
- [docs/13_RELEASE_CHECKLIST.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/13_RELEASE_CHECKLIST.md)
- [docs/17_DELIVERY_READINESS.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/17_DELIVERY_READINESS.md)
- [docs/19_DELIVERY_UAT_CASES.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/19_DELIVERY_UAT_CASES.md)
- [docs/20_RELEASE_SYNC_PLAYBOOK.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/20_RELEASE_SYNC_PLAYBOOK.md)
- [docs/USER_MANUAL_V0.3.3.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/USER_MANUAL_V0.3.3.md)
- [docs/PARENT_WEB_H5_MANUAL.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/PARENT_WEB_H5_MANUAL.md)
