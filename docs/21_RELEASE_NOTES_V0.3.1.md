# StudyClaw v0.3.1 发布说明

发布日期：`2026-03-12`

## 版本定位

`v0.3.1` 是在 `v0.3.0` 基线上的正式补丁发布。核心业务链路、API 契约和 Pad 端语音 / 鼓励能力保持不变，本次重点补强家长端移动 H5 交互与交付文档同步。

## 本次发布包含什么

### 1. 家长端移动 H5 正式收口

- 家长端默认按手机 H5 工位运行，不再停留在“PC 仪表盘缩窄后勉强可用”的状态
- 四大主屏固定为 `发布 / 反馈 / 积分 / 单词`
- 发布主屏继续拆成 `范围 / 原文 / 审核 / 发布 / 拆分 / 任务 / 摘要 / 任务板`
- 顶部首屏压缩成短头部，避免一进页面就是很长的后台式概览
- 发布区子菜单在手机视口下保持可见，更接近原生 App 多页面切换

### 2. 发布流程更适合碎片时间操作

- 录入、审核、确认发布不再堆叠在同一长页里
- 审核与发布完成阶段保留底部动作条，减少反复滚动查找按钮
- 复杂模块拆成子页面后，默认不需要横向拖动才能看到完整内容

### 3. 文档与交付同步

- 补齐 `v0.3.1` 用户手册
- 补齐家长端移动 H5 专项手册
- 同步 README、Runbook、Release Checklist、Delivery Readiness、UAT 用例、Release Sync Playbook
- 补齐本版本发布说明，便于 GitHub release、交接和演示复用

## 对使用者的实际影响

### 家长

- 手机上更容易直接完成“先发作业，再看反馈，再调积分”的主线
- 不再需要在很长的单页里来回找模块
- 发布复杂作业时更容易在子页面之间切换核对

### 孩子

- 继续使用 `v0.3.0` 已发布的语音助手与正向鼓励能力
- 本次没有新增学习负担，也没有改变已有任务和听写操作路径

### 集成 / 运维

- API 无新增破坏性变更
- 现有启动命令、端口和联调方式保持一致
- 正式交付文档已统一到 `v0.3.1`

## 验证摘要

本次发布前已完成：

- `bash scripts/check_no_tracked_runtime_env.sh`
- `bash scripts/preflight_local_env.sh`
- `bash scripts/check_release_scope.sh`
- `go test ./... -count=1`
- `cd apps/parent-web && npm test -- --run`
- `cd apps/parent-web && npm run build`
- `cd apps/pad-app && flutter analyze`
- `cd apps/pad-app && flutter test --no-pub`
- `cd apps/pad-app && flutter build web --dart-define=API_BASE_URL=http://127.0.0.1:38080`
- `bash scripts/smoke_local_stack.sh`
- `bash scripts/demo_local_stack.sh`

## 相关文档

- [README.md](/Users/admin/Documents/WORK/ai/studyclaw/README.md)
- [docs/06_RUNBOOK.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/06_RUNBOOK.md)
- [docs/13_RELEASE_CHECKLIST.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/13_RELEASE_CHECKLIST.md)
- [docs/17_DELIVERY_READINESS.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/17_DELIVERY_READINESS.md)
- [docs/19_DELIVERY_UAT_CASES.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/19_DELIVERY_UAT_CASES.md)
- [docs/20_RELEASE_SYNC_PLAYBOOK.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/20_RELEASE_SYNC_PLAYBOOK.md)
- [docs/USER_MANUAL_V0.3.1.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/USER_MANUAL_V0.3.1.md)
- [docs/PARENT_WEB_H5_MANUAL.md](/Users/admin/Documents/WORK/ai/studyclaw/docs/PARENT_WEB_H5_MANUAL.md)
