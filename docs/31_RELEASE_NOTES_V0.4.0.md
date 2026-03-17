# StudyClaw v0.4.0 发布说明

发布日期：`2026-03-16`

## 版本定位

`v0.4.0` 是 StudyClaw 第二阶段 `Agentic 学习助手` 的正式收口版本。

这版的关键意义不是继续补第一阶段发布材料，而是把下面这条核心主线真正闭环成可复盘、可验收、可继续发版的正式能力：

1. 家长在发布任务阶段前置准备学习素材
2. 孩子在 Pad 端通过持续监听完成背诵 / 朗读语音学习
3. 系统按 grounded 分析输出标题识别、完成度、逐句对照和重练建议
4. 家长端可以直接看到某次语音学习结果摘要并做复盘判断

## 本次发布包含什么

### 1. 学习素材前置化正式收口

- 家长端发布 / 审核链路继续保留 `reference_title`、`reference_author`、`reference_text`、`reference_source`、`hide_reference_from_child`、`analysis_mode`
- 背诵 / 朗读任务可以在发布阶段就确认学习素材来源与孩子端隐藏策略
- 老师原文抽取、人工覆盖和 LLM 补缺的来源口径保持可追溯

### 2. Pad 学习语音工作台主链收口

- 孩子可手动开始、持续说话、手动结束
- 中途短暂停顿不会把主流程直接打断
- transcript 会整理出真实停顿分段与时间点
- 背诵 / 朗读分析继续输出标题 / 作者识别、完成度、逐句对照和建议重练

### 3. 家长端语音学习结果摘要闭环

- 新增语音学习会话持久化与查询接口
- Parent Web 反馈区新增 `语音` 视图
- 家长现在可以直接看到：
  - 本次语音学习摘要
  - 标题 / 作者识别
  - 完成度
  - 逐句对照重点
  - 孩子真实开口记录
  - 是否建议重练

## 对交付的实际影响

### 家长

- 不再只能看“任务完成了没有”
- 现在可以看到孩子某次背诵 / 朗读到底背到什么程度、哪些句子不稳、是否需要再练

### 孩子

- 语音工作台不只是当场反馈
- 这次学习结果会真正沉淀到家长复盘链路里

### 团队

- `v0.4.0` 的成功标准已不再停留在 Pad 端单端演示
- 现在已经有跨端的 `Pad -> API -> Parent` 语音学习复盘闭环

## 验证摘要

本次发布已完成：

- `bash scripts/check_no_tracked_runtime_env.sh`
- `bash scripts/preflight_local_env.sh`
- `bash scripts/check_release_scope.sh`
- `cd apps/api-server && go test ./... -count=1`
- `cd apps/parent-web && npm test -- --run`
- `cd apps/parent-web && npm run build`
- `cd apps/pad-app && flutter analyze`
- `cd apps/pad-app && flutter test --no-pub`

## 已知说明

- `flutter build web` 仍会输出第三方依赖 `flutter_tts` 的 wasm dry-run warning
- 当前 `flutter analyze`、`flutter test` 已通过；如执行 web build，warning 仍不阻塞当前交付结论

## 相关文档

- [README.md](/Users/admin/Documents/WORK/ai/studyclaw/README.md)
- [docs/03_ROADMAP.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/03_ROADMAP.md)
- [docs/13_RELEASE_CHECKLIST.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/13_RELEASE_CHECKLIST.md)
- [docs/17_DELIVERY_READINESS.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/17_DELIVERY_READINESS.md)
- [docs/20_RELEASE_SYNC_PLAYBOOK.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/20_RELEASE_SYNC_PLAYBOOK.md)
- [docs/USER_MANUAL_V0.4.0.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/USER_MANUAL_V0.4.0.md)
